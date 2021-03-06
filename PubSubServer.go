/*

This server is a light-weight and almost scalable way
to exchange information between different systems in real-time.

When running, the HTTP server accepts requests on three
different URLs:

    * POST /pub?id=xyz -- Publish message on channel xyz. [1]
    * GET /sub?id=xyz -- Get messages from channel xyz. [2]
    * GET/POST /del?id=xyz -- Tear down channel xyz, closing all client connections. [3]
    * GET /gen_channel?length=16 -- Give back a $length character long random string which can be
				    used as channel id.

    Note: The id= parameter can have another name if you specified one explicitly with the
    --channel_id_key command line parameter.

    Note: The set= parameter can have another name if you specified one explicitly with the
    --channelset_key command line parameter.

    Use the --logfile parameter to explicitly specify a log file (Default: stdout).

[1] This request publishes a message on channel id "xyz". The Content-Type is forwarded
to any subscribers. A message is lost if there are no subscribers at the moment of publishing.

Multiple &id= parameters may be used to multicast a message to the specified channels.

An optional &set= (--channelset_key) parameter can be used to publish the message
to the channels in set <set>; a channel set is like a namespace:

    POST /pub?id=a&set=x
    POST /pub?id=a&set=y

These requests publish to completely different channels. There is an anonymous default set which
is used if there is no explicit set given.

[2] This request establishes a long-lived connection with Transfer-Encoding: chunked.
This means that incoming messages on the requested channel are streamed to the client.

Use the URL parameter &no_chunked (or one with the name given with the command line
parameter --no_chunked_key) to request a connection that only receives one message
and is closed afterwards.

Use multiple &id= parameters to listen on more than one channel.

Use the &set= parameter to listen to the given channel
in a specific set. Without this URL parameter, the default set is used.

[3] This request frees resources associated with this channels and terminates
all connections relying on it by sending a zero-length chunk (and exiting the goroutine)


TODO: [Work]
    * ✓ [3] Implement proper means of configuration
    * ✓ [2] Allow listening for messages on multiple channels at once (?id=a&id=b)
    * ✓ [2] Allow publishing on several channels at once (like the item above)
    * ✓ [2] Allow requesting a non-chunked connection (new request for every message)
    *   [4] Implement correct buffering (If-Modified-Since etc; could be difficult w/o making the server too heavy)
    * ✓ [3] Implement different channelsets (i.e. /pub/chanset?id=xyz)
    * ✓ [3] Propagate Content-Type (i.e. implement struct for messages, add Content-Type field)


Copyright © 2014 lbo@spheniscida.de

This software is an official part of the cHaTTP application stack (github.com/Spheniscida/cHaTTP)

This software is licensed under the terms and conditions of the Apache license (http://www.apache.org/licenses/LICENSE-2.0.html)

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Those are the command-line parameters
var (
	channel_id_key,
	channel_id_length_key,
	channelset_key,
	no_chunked_key,
	log_file,
	bind_host,
	bind_port string
)

const logFlags int = log.Ldate | log.Ltime | log.Lmicroseconds

var logger *log.Logger

var channelSetMap map[string](*channelSet)

const defaultChannelId = "__DEFAULT_CHANNEL_SET"

func setupLogger() {
	if log_file == "<stdout>" {
		logger = log.New(os.Stderr, "", logFlags)
		return
	}

	if log_file == "" {
		var nullwriter NullWriter
		logger = log.New(&nullwriter, "", logFlags)
		return
	}

	file, err := os.OpenFile(log_file, os.O_APPEND|os.O_WRONLY, 0600)

	if err != nil {
		log.Panic("Couldn't open logfile")
	}

	logger = log.New(file, "", logFlags)
}

func parseFlags() {
	flag.StringVar(&bind_host, "host", "0.0.0.0", "Specifies the IP address to listen on")
	flag.StringVar(&bind_port, "port", "8080", "Specifies the TCP port to listen on")

	flag.StringVar(&channel_id_key, "channel_id_key", "id", "Specifies which (URL) parameter holds the channel ID")
	flag.StringVar(&no_chunked_key, "no_chunked_key", "no_chunked", "Specifies which (URL) parameter tells us to not use chunked encoding (value of parameter is irrelevant)")
	flag.StringVar(&channel_id_length_key, "channel_id_length_key", "length", "The parameter which tells the /gen_channel handler (which generates a random channel ID) how long the channel ID should be")
	flag.StringVar(&channelset_key, "channelset_key", "set", "The name of the URL parameter holding the name of the channel set")
	flag.StringVar(&log_file, "logfile", "<stdout>", "Where to send log messages (file name!). Specify as empty to discard messages.")

	helpRequested := flag.Bool("help", false, "Give help on commands")

	flag.Parse()

	if *helpRequested {
		flag.PrintDefaults()
		os.Exit(0)
	}

	setupLogger()
}

func main() {
	fmt.Print("\n")

	parseFlags()

	channelSetMap = make(map[string](*channelSet))
	channelSetMap[defaultChannelId] = initializeChannelSet(defaultChannelId)

	http.HandleFunc("/pub/", PubFunc)
	http.HandleFunc("/pub", PubFunc)

	http.HandleFunc("/sub/", SubFunc)
	http.HandleFunc("/sub", SubFunc)

	http.HandleFunc("/del/", DelFunc)
	http.HandleFunc("/del", DelFunc)

	http.HandleFunc("/gen_channel", GenChannelFunc)
	http.HandleFunc("/gen_channel/", GenChannelFunc)

	http.HandleFunc("/test", TestFunc)

	http.ListenAndServe(bind_host+":"+bind_port, nil)
}
