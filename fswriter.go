package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const stdPerms = 0600

type roomUpdate struct {
	room     *room
	roomname roomname
}

type fswriter struct {
	queue chan roomUpdate
	dir   string
}

func (fswriter *fswriter) readRoomSize(roomname roomname) uint32 {
	fi, err := os.Stat(fmt.Sprintf("%s/%s", fswriter.dir, string(roomname)))
	switch err.(type) {
	case nil:
	case *os.PathError:
		return 0
	default:
		panic("unexpected error while reading file stats")
	}
	return uint32(fi.Size())
}

func (fswriter *fswriter) registerRoomUpdate(room *room, roomname roomname) {
	fswriter.queue <- roomUpdate{room, roomname}
}

func (fswriter *fswriter) startWriteTask() {
	dir := fswriter.dir
	for {
		writeTask := <-fswriter.queue
		room := writeTask.room
		roomname := writeTask.roomname
		time.Sleep(time.Millisecond * 800)
		room.mux.Lock()
		debug("fswriter: created room lock")
		pendingWrites := room.pendingWrites
		dataAvailable := false
		if len(pendingWrites) > 0 {
			// New data is available.
			dataAvailable = true
			// This goroutine will save dataAvailable and confirm pending confirmations
			room.pendingWrites = nil
		}
		for _, sub := range room.pendingSubs {
			if !room.hasSession(sub.session) {
				f, _ := os.OpenFile(fmt.Sprintf("%s/%s", dir, string(roomname)), os.O_RDONLY|os.O_CREATE, stdPerms)
				if sub.offset > 0 {
					f.Seek(int64(sub.offset), 0)
				}
				data, _ := ioutil.ReadAll(f)
				data = append(data, pendingWrites...)
				confirmedOffset := uint64(sub.offset) + uint64(len(data))
				// TODO: combine sub and update here
				sub.session.sendUpdate(roomname, data, confirmedOffset)
				sub.session.sendConfirmedByHost(roomname, confirmedOffset)
				room.subs = append(room.subs, sub.session)
				f.Close()
			}
		}
		room.pendingSubs = nil
		room.registered = false
		if dataAvailable {
			debug("fswriter: enter dataAvailable - write file")
			f, err := os.OpenFile(fmt.Sprintf("%s/%s", dir, string(roomname)), os.O_APPEND|os.O_WRONLY|os.O_CREATE, stdPerms)
			if err != nil {
				panic(err)
			}
			debug("fswriter: opened file")

			if _, err = f.Write(pendingWrites); err != nil {
				panic(err)
			}
			debug("fswriter: writing file")
			f.Close()
			debug("fswriter: closed file")
			// confirm after we can assure that data has been written
			for _, sub := range room.subs {
				sub.sendConfirmedByHost(roomname, uint64(room.offset))
			}
			debug("fswriter: left dataAvailable - sent confirmedByHost")
		}
		room.mux.Unlock()
		debug("fswriter: removed lock")
	}
}

func newFSWriter(dir string, fsAccessQueueLen uint, writeConcurrency int) (fswriter fswriter) {
	fswriter.dir = dir
	// must include x permission for user, otherwise user can't write files
	if err := os.MkdirAll(dir, stdPerms|0100); err != nil {
		panic(err)
	}

	fswriter.queue = make(chan roomUpdate, fsAccessQueueLen)
	// TODO: start several write tasks
	/*
		for i := 0; i < writeConcurrency; i++ {wsConn
			go fswriter.startWriteTask()
		}
	*/
	go fswriter.startWriteTask()
	return
}
