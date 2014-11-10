/*

This server is a light-weight and almost scalable way
to exchange information between different systems in real-time.

When running, the HTTP server accepts requests on three
different URLs:

    * POST /pub?id=xyz -- Publish message on channel xyz. [1]
    * GET /sub?id=xyz -- Get messages from channel xyz. [2]
    * GET/POST /del?id=xyz -- Tear down channel xyz, closing all client connections. [3]

[1] This request publishes a message. The Content-Type is irrelevant (yet), but to be
sure, set it to text/plain or application/json or application/xml (or any meaningful)

[2] This request establishes a long-lived connection with Transfer-Encoding: chunked.
This means that incoming messages on the requested channel are streamed to the client.

[3] This request frees resources associated with this channels and terminates
all connections relying on it by sending a zero-length chunk (i.e. exiting the goroutine)

TODO: [Work]
    * [3] Implement proper means of configuration
    * [2] Allow listening for messages on multiple channels at once (?id=a&id=b)
    * [2] Allow requesting a non-chunked connection (new request for every message)
    * [4] Implement correct buffering (could be difficult w/o making the server too heavy)
    * [2] Implement different channelsets (i.e. /pub/chanset?id=xyz)
    * [3] Propagate Content-Type (i.e. implement struct for messages, add Content-Type field)

Copyright © 2014, Google Inc. <lewinb@google.com>

This software is licensed under the terms and conditions of the Apache license (http://www.apache.org/licenses/LICENSE-2.0.html)

 */
package main

import (
	"container/list"
	"fmt"
	"log"
	"flag"
	"net/http"
	"time"
)

var logger log.Logger

var defaultChannelSet channelSet



func main() {
	fmt.Print("\n")

	defaultChannelSet.channels = make(map[string]*list.List)

	http.HandleFunc("/pub/", PubFunc)
	http.HandleFunc("/pub", PubFunc)

	http.HandleFunc("/sub/", SubFunc)
	http.HandleFunc("/sub", SubFunc)

	http.HandleFunc("/del/", DelFunc)
	http.HandleFunc("/del", DelFunc)

	http.HandleFunc("/test", TestFunc)

	http.ListenAndServe("localhost:8080", nil)
}

