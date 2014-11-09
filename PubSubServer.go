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

This software is licensed under the terms and conditions of the MIT license (http://opensource.org/licenses/MIT)

 */
package main

import (
	"container/list"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type channelSet struct {
	channels      map[string]*list.List
	channels_lock sync.Mutex
}

type subscription struct {
	listKey    *list.Element
	channel_id string
	channel    chan string
}

var defaultChannelSet channelSet

// Effectively returns the value of the id= parameter of the URL.
func getChannelId(u *url.URL) string {
	v, err := url.ParseQuery(u.RawQuery)

	if err != nil {
		return ""
	}

	return v["id"][0]
}

// Invoked for all /pub/ requests
func PubFunc(w http.ResponseWriter, r *http.Request) {
	chan_id := getChannelId(r.URL)

	var b []byte = make([]byte, r.ContentLength)

	l, _ := r.Body.Read(b)

	if l == 0 {
		w.WriteHeader(400)
		return
	}
	w.WriteHeader(200)

	defaultChannelSet.publish(chan_id, string(b))

}

func SubFunc(w http.ResponseWriter, r *http.Request) {
	chan_id := getChannelId(r.URL)
	var channel_closed bool = false

	sub := defaultChannelSet.subscribe(chan_id)
	defer func() {
		if !channel_closed { // Don't close the channel twice (runtime error!)
			defaultChannelSet.cancelSubscription(&sub)
		}
	}()

	for {
		msg, ok := <-sub.channel

		if !ok { // Channel has been closed
			channel_closed = true
			break
		}

		_, err := w.Write([]byte(msg + "\n")) // Line-end because otherwise the chunk is not transmitted

		if err != nil {
			break
		}

		// If the server has the capability, use chunked encoding. Else don't.
		if w.(http.Flusher) != nil {
			w.(http.Flusher).Flush()
		} else {
			break
		}
	}
}

func DelFunc(w http.ResponseWriter, r *http.Request) {
	chan_id := getChannelId(r.URL)

	defaultChannelSet.deleteChannel(chan_id)

}

func TestFunc(w http.ResponseWriter, r *http.Request) {
	for i := 0; i < 3; i++ {
		w.Write([]byte("abc\n"))
		time.Sleep(1 * time.Second)
		w.(http.Flusher).Flush()
	}
}

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

func (cs *channelSet) subscribe(chan_id string) subscription {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	var sub subscription
	sub.channel_id = chan_id

	if cs.channels[chan_id] == nil {
		cs.channels[chan_id] = list.New()
	}

	sub.channel = make(chan string, 10)
	sub.listKey = cs.channels[chan_id].PushFront(sub.channel)

	return sub
}

func (cs *channelSet) cancelSubscription(sub *subscription) {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	log.Println("Cancelled subscription ", sub.listKey, " ", sub.channel_id)

	cs.channels[sub.channel_id].Remove(sub.listKey)
	close(sub.channel)
}

func (cs *channelSet) publish(chan_id, message string) error {

	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	if cs.channels[chan_id] == nil || cs.channels[chan_id].Front() == nil {
		return nil
	}

	for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
		e.Value.(chan string) <- message
	}

	return nil
}

func (cs *channelSet) deleteChannel(chan_id string) {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
		close(e.Value.(chan string))
	}

	delete(cs.channels, chan_id)
}
