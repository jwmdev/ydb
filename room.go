package main

import (
	"sync"
)

type roomname string

type pendingWrite struct {
	data    []byte
	session *session
	conf    uint64
}

type room struct {
	mux           sync.Mutex
	registered    bool
	pendingWrites []pendingWrite
	subs          map[*session]struct{}
	pendingSubs   []pendingSub
}

func newRoom() *room {
	return &room{
		subs: make(map[*session]struct{}),
	}
}

func modifyRoom(roomname roomname, f func(room *room) (modified bool)) {
	room := getRoom(roomname)
	var register bool
	room.mux.Lock()
	modified := f(room)
	if room.registered == false && modified {
		register = true
		room.registered = true
	}
	room.mux.Unlock()
	if register {
		ydb.fswriter.registerRoomUpdate(room, roomname)
	}
}

// update in-memory buffer of writable data. Registers in fswriter if new data is available.
// Writes to buffer until fswriter owns the buffer.
func updateRoom(roomname roomname, session *session, conf uint64, bs []byte) {
	modifyRoom(roomname, func(room *room) bool {
		room.pendingWrites = append(room.pendingWrites, pendingWrite{bs, session, conf})
		for s := range room.subs {
			if s != session {
				s.sendUpdate(roomname, bs)
			}
		}
		return true
	})
}

type pendingSub struct {
	session *session
	offset  uint64
}

func subscribeRoom(roomname roomname, session *session, offset uint64) {
	modifyRoom(roomname, func(room *room) bool {
		_, ok := room.subs[session]
		if !ok {
			room.pendingSubs = append(room.pendingSubs, pendingSub{session, offset})
		}
		return !ok // whether room data needs to access fswriter
	})
}
