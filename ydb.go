package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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
	// remember to update unsafeClearAllYdbContent when updating here
	ydb = Ydb{
		rooms:         make(map[roomname]*room, 1000),
		sessions:      make(map[uint64]*session),
		sessionidSeed: rand.New(rand.NewSource(time.Now().UnixNano())),
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
			ydb.rooms[name] = r
			r.mux.Lock()
			ydb.roomsMux.Unlock()
			// read room offset..
			r.offset = ydb.fswriter.readRoomSize(name)
			r.mux.Unlock()
		} else {
			ydb.roomsMux.Unlock()
		}
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

// TODO: refactor/remove..
func removeFSWriteDirContent(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	d.Chmod(0777)
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	debug("read dir names")
	for _, name := range names {
		os.Chmod(filepath.Join(dir, name), 0777)
		err = os.Remove(filepath.Join(dir, name))
		debug("removed a file")
		if err != nil {
			return err
		}
	}
	return nil
}

// Clear all content in Ydb (files, sessions, rooms, ..).
// Unsafe for production, only use for testing!
// only works if dir is tmp
func unsafeClearAllYdbContent() {
	dir := ydb.fswriter.dir
	debug("Clear Ydb content")
	ydb.rooms = make(map[roomname]*room, 1000)
	ydb.sessions = make(map[uint64]*session)
	removeFSWriteDirContent(dir)
}
