package main

import (
	"container/list"
	"log"
	"sync"
)

type message struct {
	msg []byte
	content_type string
}

type channelSet struct {
	channels      map[string]*list.List
	channels_lock sync.Mutex
}

type subscription struct {
	listKey    *list.Element
	channel_id string
	channel    chan message
}


func (cs *channelSet) subscribe(chan_id string) subscription {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	var sub subscription
	sub.channel_id = chan_id

	if cs.channels[chan_id] == nil {
		cs.channels[chan_id] = list.New()
	}

	sub.channel = make(chan message)
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

func (cs *channelSet) publish(chan_id string, msg message) (delivered uint, e error) {

	var successful_published uint = 0

	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	if cs.channels[chan_id] == nil || cs.channels[chan_id].Front() == nil {
		return 0, nil
	}

	for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
		/*
		// Would block if receiver is not ready to receive message.
		e.Value.(chan message) <- message
		successful_published++
		*/

		select {
		case e.Value.(chan message) <- msg:
			successful_published++
		default:
			continue
		}

	}

	logger.Printf("Published message %.5s... to channel %.5s...", msg.msg, chan_id)

	return successful_published, nil
}

func (cs *channelSet) deleteChannel(chan_id string) {
	cs.channels_lock.Lock()
	defer cs.channels_lock.Unlock()

	for e := cs.channels[chan_id].Front(); e != nil; e = e.Next() {
		close(e.Value.(chan message))
	}

	delete(cs.channels, chan_id)

	logger.Println("Deleted channel ", chan_id)
}

