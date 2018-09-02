package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var ydb Ydb

// Ydb maintains rooms and connections
type Ydb struct {
	roomsMux sync.RWMutex
	rooms    map[roomname]*room
	// TODO: use guid instead of uint64
	sessionidSeed *rand.Rand
	sessionsMux   sync.Mutex
	sessions      map[uint64]*session
	fswriter      fswriter
}

func initYdb(dir string) {
	ydb = Ydb{
		rooms:         make(map[roomname]*room, 1000),
		sessionidSeed: rand.New(rand.NewSource(time.Now().UnixNano())),
		sessions:      make(map[uint64]*session),
		fswriter:      newFSWriter(dir, 1000, 10), // TODO: have command line arguments for this
	}
}

// getRoom from the global ydb instance. safe for parallel access.
func getRoom(name roomname) *room {
	ydb.roomsMux.RLock()
	r := ydb.rooms[name]
	ydb.roomsMux.RUnlock()
	if r == nil {
		ydb.roomsMux.Lock()
		r = ydb.rooms[name]
		if r == nil {
			r = newRoom()
		}
		ydb.rooms[name] = r
		ydb.roomsMux.Unlock()
	}
	return r
}

func (ydb *Ydb) getSession(sessionid uint64) *session {
	ydb.sessionsMux.Lock()
	defer ydb.sessionsMux.Unlock()
	return ydb.sessions[sessionid]
}

func (ydb *Ydb) createSession() (s *session) {
	ydb.sessionsMux.Lock()
	sessionid := ydb.sessionidSeed.Uint64()
	if _, ok := ydb.sessions[sessionid]; ok {
		panic("Generated the same session id twice! (this is a security vulnerability)")
	}
	s = newSession(sessionid)
	ydb.sessions[sessionid] = s
	ydb.sessionsMux.Unlock()
	return s
}

func (ydb *Ydb) removeSession(sessionid uint64) (err error) {
	ydb.sessionsMux.Lock()
	session, ok := ydb.sessions[sessionid]
	if !ok {
		err = fmt.Errorf("tried to close session %d, but session does not exist", sessionid)
	} else if len(session.conns) > 0 || session.conn != nil {
		err = errors.New("Cannot close this session because conns are still using it")
	} else {
		delete(ydb.sessions, sessionid)
	}
	ydb.sessionsMux.Unlock()
	return
}
