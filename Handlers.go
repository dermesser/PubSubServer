package main

import (
	"fmt"
	"net/http"
	"time"
)

// Invoked for all /pub/ requests.
func PubFunc(w http.ResponseWriter, r *http.Request) {
	params := getAllURLParameters(r.URL)

	var msg message
	var n_succeeded uint
	var err error

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

	var channelset_id string

	if len(params[channelset_key]) == 0 {
		channelset_id = defaultChannelId
	} else {
		channelset_id = params[channelset_key][0]
	}

	if channelSetMap[channelset_id] == nil { // Drop message (channelset doesn't exist therefore nobody listens).
		n_succeeded = 0
		err = nil
		logger.Printf("Lost message because channelset %.5s... doesn't exist", channelset_id)
	} else {
		n_succeeded, err = channelSetMap[channelset_id].publish(params[channel_id_key], msg)
	}

	if err != nil { // Is always nil yet.
		w.WriteHeader(500)
		return
	}

	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("{\"succ\": %d }", n_succeeded)))

	return
}

func SubFunc(w http.ResponseWriter, r *http.Request) {
	params := getAllURLParameters(r.URL)

	chan_ids := params[channel_id_key]

	channel_closed := false

	is_chunked := len(params[no_chunked_key]) == 0

	var sub subscription
	var channelset_id string

	if len(params[channelset_key]) == 0 {
		channelset_id = defaultChannelId
	} else if len(params[channelset_key]) > 0 {
		channelset_id = params[channelset_key][0]
	}

	if channelSetMap[channelset_id] == nil {
		channelSetMap[channelset_id] = initializeChannelSet(channelset_id)
	}

	sub = channelSetMap[channelset_id].subscribe(chan_ids)

	defer func() {
		if !channel_closed { // Don't close the channel twice (runtime error!)
			channelSetMap[channelset_id].cancelSubscription(&sub)
		}
	}()

	for {
		msg, not_closed := <-sub.channel

		if !not_closed { // Channel probably has been closed
			channel_closed = true
			break
		}

		w.Header()["Content-Type"] = []string{msg.content_type} // Just use the content-type of the first message
		_, err := w.Write(msg.msg)                              // Line-end because otherwise the chunk is not transmitted
		w.Write([]byte("\r\n"))

		if err != nil { // Probably, the client has disconnected
			break
		}

		if is_chunked { // If the client wants a chunked connection, try to deliver it.
			if w.(http.Flusher) != nil { // If the server has the capability, use chunked encoding. Else don't.
				w.(http.Flusher).Flush()
			} else {
				break
			}
		} else { // Terminate connection, unsubscribe (defer'ed)
			break
		}
	}
}

func DelFunc(w http.ResponseWriter, r *http.Request) {
	params := getAllURLParameters(r.URL)
	var err error
	var channelset_id string

	if len(params[channel_id_key]) == 0 {
		w.WriteHeader(400)
	}

	if len(params[channelset_key]) == 0 {
		channelset_id = defaultChannelId
	} else {
		channelset_id = params[channelset_key][0]
	}

	err = channelSetMap[channelset_id].deleteChannels(params[channel_id_key])

	if err != nil {
		w.WriteHeader(404) // The only case where deleteChannel returns err != nil is if the channel doesn't exist
		return
	}

	w.WriteHeader(200)

}

func GenChannelFunc(w http.ResponseWriter, r *http.Request) {
	var length uint
	length_string := getURLParameter(r.URL, channel_id_length_key)

	if length_string != "" {
		fmt.Sscan(getURLParameter(r.URL, channel_id_length_key), &length)
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
