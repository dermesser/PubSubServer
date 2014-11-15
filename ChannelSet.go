package main

import (
	"container/list"
	"errors"
	"sync"
)

type message struct {
	msg          []byte
	content_type string
}

type channelSet struct {
	channels      map[string]*list.List
	channels_lock sync.Mutex
}

type subscription struct {
	// Pointer to list element for removal from linked lists
	listKeys []*list.Element
	// Channel ids subscribed to
	channel_ids []string
	channel     chan message
}

func (cs *channelSet) subscribe(chan_ids []string) subscription {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	var sub subscription

	sub.channel_ids = chan_ids
	sub.listKeys = make([]*list.Element, len(sub.channel_ids))
	// Buffered length is 5, if one subscriber is subscribed to multiple
	// channels which receive messages simultaneously. Else the publisher
	// loop would give up because the channel would block.
	sub.channel = make(chan message, 5)

	for ix, chan_id := range sub.channel_ids {
		if cs.channels[chan_id] == nil {
			cs.channels[chan_id] = list.New()
		}

		sub.listKeys[ix] = cs.channels[chan_id].PushFront(sub.channel)
	}
	logger.Printf("Subscribed client to channel(s) %.5s...", chan_ids)

	return sub
}

func (cs *channelSet) cancelSubscription(sub *subscription) {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	for i, e := range sub.channel_ids {
		cs.channels[e].Remove(sub.listKeys[i])

		if cs.channels[e].Front() == nil {
			delete(cs.channels, e)
		}

		logger.Printf("Cancelled subscription to channel %.5s...", e)
	}

	close(sub.channel)

	return
}

func (cs *channelSet) publish(chan_ids []string, msg message) (delivered uint, e error) {

	var n_published uint = 0

	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	for _, chan_id := range chan_ids {

		if cs.channels[chan_id] == nil { // Channel doesn't exist, skip
			continue
		}

		for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
			select {
			case e.Value.(chan message) <- msg:
				n_published++
			default:
				continue
			}
		}
	}

	if n_published == 0 {
		logger.Printf("Lost message %.5s... to channel(s) %.5s...", msg.msg, chan_ids)
	} else {
		logger.Printf("Published message %.5s... %d times to channel(s) %.5s...", msg.msg, n_published, chan_ids)
	}

	return n_published, nil
}

func (cs *channelSet) deleteChannel(chan_id string) error {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	if cs.channels[chan_id] == nil { // Channel doesn't exist
		return errors.New("Channel doesn't exist (anymore)")
	}

	for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
		close(e.Value.(chan message))
	}

	delete(cs.channels, chan_id)

	logger.Println("Deleted channel ", chan_id)

	return nil
}
