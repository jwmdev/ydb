package ydb

import (
	"sync"
)

type pendingConfirmation struct {
	session      *session
	confirmation uint64
}

func (c *pendingConfirmation) confirm() {
	c.session.confirm(c.confirmation)
}

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

// client confirmed that it received and persisted data.
func (conf *serverConfirmation) clientConfirmed(confirmed uint64) {
	if conf.nextClient <= confirmed {
		conf.nextClient = confirmed + 1
		// recreate a new roomsChanged map to assure that memory does not grow
		roomsChanged := conf.roomsChanged
		conf.roomsChanged = make(map[roomname]uint64, 1)
		// re-insert all rooms that are not yet confirmed
		for roomname, n := range roomsChanged {
			if n > confirmed {
				conf.roomsChanged[roomname] = n
			}
		}
	}
}

// clientConfirmation keeps track of confirmations received from the client.
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
		_, ok := conf.confs[conf.next]
		for ok {
			_, ok = conf.confs[conf.next]
			conf.next++
		}
		if conf.next != confirmed+1 {
			// conf updated based on confs
			// conf.confs needs to be updated
			oldConfs := conf.confs
			conf.confs = make(map[uint64]struct{}, 0)
			for n := range oldConfs {
				conf.confs[n] = struct{}{}
			}
		}
		return true
	} else {
		conf.confs[confirmed] = struct{}{}
		return false
	}
}

type session struct {
	mux sync.Mutex
	// currently active connection
	conn conn
	// set of all conns
	conns              map[conn]struct{}
	serverConfirmation serverConfirmation
	clientConfirmation clientConfirmation
}

func (s *session) confirmToClient(confirmation uint64) {
	s.mux.Lock()
	if s.clientConfirmation.serverConfirmed(confirmation) {
		confMessage := messageConfirmation(confirmation)
		s.send(confMessage)
	}
	s.mux.Unlock()
}

func (s *session) send(m message) {
	s.conn.Write(m)

}

func (s *session) add(conn conn) {
	s.mux.Lock()
	if s.conn == nil {
		s.conn = conn
		// initialize conn here
	}
	s.mux.Unlock()
}
