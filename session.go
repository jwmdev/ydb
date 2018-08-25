package main

import (
	"sync"
)

// serverConfirmation keeps track of confirmations created by the server.
// serverConfirmation.next is attached to data sent from the server to a client (via some conn).
// When the client confirms a serverConfirmation number, it assures that it consumed and persisted the sent data data.
// Hence the server does not have to keep track of roomsChanged since last confirmed message.
type serverConfirmation struct {
	// current confirmation number
	next uint64
	// next expected confirmation from client
	nextClient uint64
	// rooms changed since last confirmation.
	roomsChanged map[roomname]uint64
}

func (serverConfirmation *serverConfirmation) createConfirmation() uint64 {
	conf := serverConfirmation.next
	serverConfirmation.next++
	return conf
}

// client confirmed that it received and persisted data.
func (serverConfirmation *serverConfirmation) clientConfirmed(confirmed uint64) {
	if serverConfirmation.nextClient <= confirmed {
		serverConfirmation.nextClient = confirmed + 1
		// recreate a new roomsChanged map to assure that memory does not grow
		roomsChanged := serverConfirmation.roomsChanged
		serverConfirmation.roomsChanged = make(map[roomname]uint64, 1)
		// re-insert all rooms that are not yet confirmed
		for roomname, n := range roomsChanged {
			if n > confirmed {
				serverConfirmation.roomsChanged[roomname] = n
			}
		}
	}
}

// clientConfirmation keeps track of confirmation numbers received from the client.
// ydb confirms messages when they are consumed and persisted on the disk.
// The client will receive confirmations in-order. But since the process of persisting
// data is asynchronous, it may happen that confirmations are created out-of-order.
// I.e. The client wrote data to room "a" (confirmation 0) then "b" (confirmation 1).
// But the server creates confirmation 1, then 0. We keep track of the confirmations in a map.
type clientConfirmation struct {
	// next expected confirmation
	next uint64
	// set of out-of-order created confirmations (none of them is `next`)
	confs map[uint64]struct{}
}

func (conf *clientConfirmation) serverConfirmed(confirmed uint64) (updated bool) {
	if conf.next == confirmed {
		conf.next = confirmed + 1
		if conf.confs != nil {
			_, ok := conf.confs[conf.next]
			for ok {
				_, ok = conf.confs[conf.next]
				conf.next++
			}
		}
		if conf.next != confirmed+1 {
			// conf updated based on confs
			// conf.confs needs to be updated
			oldConfs := conf.confs
			newConfs := make(map[uint64]struct{}, 0)
			for n := range oldConfs {
				newConfs[n] = struct{}{}
			}
			if len(newConfs) > 0 {
				conf.confs = newConfs
			}
		}
		return true
	}
	conf.confs[confirmed] = struct{}{}
	return false
}

type session struct {
	mux sync.Mutex
	// currently active connection
	conn conn
	// set of all conns
	conns map[conn]struct{}
	// client confirming messages to server
	serverConfirmation serverConfirmation
	// server confirming messages to client
	clientConfirmation clientConfirmation
}

func newSession() *session {
	return &session{
		conns: make(map[conn]struct{}, 1),
	}
}

func (s *session) sendConfirmation(confirmation uint64) {
	s.mux.Lock()
	if s.clientConfirmation.serverConfirmed(confirmation) {
		confMessage := createMessageConfirmation(confirmation)
		s.conn.Write(confMessage)
	}
	s.mux.Unlock()
}

func (s *session) sendUpdate(roomname roomname, data []byte) {
	if len(data) > 0 {
		s.mux.Lock()
		m := createMessageUpdate(roomname, s.serverConfirmation.createConfirmation(), data)
		s.conn.Write(m)
		s.mux.Unlock()
	}
}

func (s *session) add(conn conn) {
	s.mux.Lock()
	if s.conn == nil {
		s.conn = conn
		// initialize conn here
	}
	s.mux.Unlock()
}
