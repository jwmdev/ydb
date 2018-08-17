package ydb

import (
	"bytes"
	"sync"
)

type roomname string

type room struct {
	mux                  sync.Mutex
	registered           bool
	pendingData          bytes.Buffer
	pendingConfirmations []pendingConfirmation
	subs                 map[*session]struct{}
	pendingSubs          []pendingSub
}

func modifyRoom(roomname roomname, f func(room *room)) {
	room := getRoom(roomname)
	var register bool
	room.mux.Lock()
	if room.registered == false {
		register = true
		room.registered = true
	}
	f(room)
	room.mux.Unlock()
	if register {
		fs.registerRoomUpdate(room, roomname)
	}
}

// update in-memory buffer of writable data. Registers in fswriter if new data is available.
// Writes to buffer until fswriter owns the buffer.
func updateRoom(roomname roomname, conf pendingConfirmation, m message) {
	modifyRoom(roomname, func(room *room) {
		room.pendingData.ReadFrom(m)
		room.pendingConfirmations = append(room.pendingConfirmations, conf)
	})
}

type pendingSub struct {
	session *session
	offset  uint64
}

func subscribeRoom(roomname roomname, conf pendingConfirmation, session *session, offset uint64) {
	modifyRoom(roomname, func(room *room) {

		room.pendingSubs = append(room.pendingSubs, pendingSub{session, offset})
	})
}
