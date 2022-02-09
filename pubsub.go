package main

import (
	"math/rand"
)

type PubSub struct {
	subs map[string]map[int]chan SanitizedBlurbVersion
}

func GetPubSub() PubSub {
	return PubSub{map[string]map[int]chan SanitizedBlurbVersion{}}
}

func (ps PubSub) sub(itemId string) (chan SanitizedBlurbVersion, int) {
	subId := 0
	_, ok := ps.subs[itemId]
	if !ok {
		ps.subs[itemId] = map[int]chan SanitizedBlurbVersion{}
	}
	for ok := true; ok; _, ok = ps.subs[itemId][subId] {
		subId = rand.Int()
	}
	ch := make(chan SanitizedBlurbVersion)
	ps.subs[itemId][subId] = ch
	return ch, subId
}

func (ps PubSub) unsub(itemId string, subId int) {
	ch, ok := ps.subs[itemId][subId]
	if ok {
		close(ch)
	}
	delete(ps.subs[itemId], subId)
}

func (ps PubSub) pub(sbv SanitizedBlurbVersion) {
	// if a channel is closed, this blocks forever. potential resource leak, but it shouldn't hurt clients
	for _, channel := range ps.subs[sbv.Id] {
		channel <- sbv
	}
}
