package main

import (
	"net/http"
	"fmt"
	"time"
)

// Invoked for all /pub/ requests.
func PubFunc(w http.ResponseWriter, r *http.Request) {
	chan_id := getChannelId(r.URL)

	var msg message
	msg.msg = make([]byte, r.ContentLength)

	l, _ := r.Body.Read(msg.msg)

	if l == 0 {
		w.WriteHeader(400)
		return
	}

	if len(r.Header["Content-Type"]) > 0 {
		msg.content_type = r.Header["Content-Type"][0]
	} else {
		msg.content_type = "text/plain"
	}

	successed, err := defaultChannelSet.publish(chan_id, msg)

	if err != nil { // Is always nil yet.
		w.WriteHeader(500)
		return
	}

	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("{\"succ\": %d }",successed)))

	return
}

func SubFunc(w http.ResponseWriter, r *http.Request) {
	params := getAllURLParameters(r.URL)

	chan_id := params[channel_id_key][0]

	channel_closed := false
	is_chunked := len(params[no_chunked_key]) == 0

	sub := defaultChannelSet.subscribe(chan_id)
	defer func() {
		if !channel_closed { // Don't close the channel twice (runtime error!)
			defaultChannelSet.cancelSubscription(&sub)
		}
	}()

	for {
		msg, not_closed := <-sub.channel

		if !not_closed { // Channel probably has been closed
			channel_closed = true
			break
		}

		w.Header()["Content-Type"] = []string{msg.content_type} // Just use the first content-type
		_, err := w.Write(msg.msg) // Line-end because otherwise the chunk is not transmitted
		w.Write([]byte("\r\n"))

		if err != nil {
			break
		}

		if is_chunked {
			// If the server has the capability, use chunked encoding. Else don't.
			if w.(http.Flusher) != nil {
				w.(http.Flusher).Flush()
			} else {
				break
			}
		} else {
			break
		}
	}
}

func DelFunc(w http.ResponseWriter, r *http.Request) {
	chan_id := getChannelId(r.URL)

	defaultChannelSet.deleteChannel(chan_id)

}

func GenChannelFunc(w http.ResponseWriter, r *http.Request) {
	var length uint
	length_string := getURLParameter(r.URL,channel_id_length_key)

	if length_string != "" {
		fmt.Sscan(getURLParameter(r.URL,channel_id_length_key),&length)
	} else {
		length = 16
	}

	w.Header()["Content-Type"] = []string{"text/plain"}
	w.WriteHeader(200)
	w.Write(generateRandomAscii(length))

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
