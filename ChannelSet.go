package main

import (
	"container/list"
	"log"
	"sync"
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

