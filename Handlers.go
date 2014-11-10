package main

import (
	"net/http"
	"time"
)

// Invoked for all /pub/ requests.
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

// Tests that a client supports chunked encoding and displays
// it correctly (debugging)
func TestFunc(w http.ResponseWriter, r *http.Request) {
	for i := 0; i < 3; i++ {
		w.Write([]byte("abc\n"))
		time.Sleep(1 * time.Second)
		w.(http.Flusher).Flush()
	}
}
