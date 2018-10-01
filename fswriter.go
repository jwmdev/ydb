package main

import (
	"fmt"
	"io/ioutil"
	"os"
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

func (fswriter *fswriter) registerRoomUpdate(room *room, roomname roomname) {
	fswriter.queue <- roomUpdate{room, roomname}
}

func (fswriter *fswriter) startWriteTask() {
	dir := fswriter.dir
	for {
		writeTask := <-fswriter.queue
		room := writeTask.room
		roomname := writeTask.roomname
		room.mux.Lock()
		pendingWrites := room.pendingWrites
		dataAvailable := false
		if len(pendingWrites) > 0 {
			// New data is available.
			dataAvailable = true
			// This goroutine will save dataAvailable and confirm pending confirmations
			room.pendingWrites = nil
		}
		for _, sub := range room.pendingSubs {
			if _, ok := room.subs[sub.session]; !ok {
				f, _ := os.OpenFile(fmt.Sprintf("%s/%s", dir, string(roomname)), os.O_RDONLY|os.O_CREATE, stdPerms)
				if sub.offset > 0 {
					f.Seek(int64(sub.offset), 0)
				}
				data, _ := ioutil.ReadAll(f)
				for _, pWrite := range pendingWrites {
					if pWrite.session != sub.session {
						data = append(data, pWrite.data...)
					}
				}
				sub.session.sendUpdate(roomname, data)
				room.subs[sub.session] = struct{}{}
				f.Close()
			}
		}
		room.pendingSubs = nil
		room.registered = false
		if dataAvailable {
			var pendingData []byte
			for _, pWrite := range pendingWrites {
				pendingData = append(pendingData, pWrite.data...)
				pWrite.session.sendConfirmation(pWrite.conf)
			}
			f, err := os.OpenFile(fmt.Sprintf("%s/%s", dir, string(roomname)), os.O_APPEND|os.O_WRONLY|os.O_CREATE, stdPerms)
			if err != nil {
				panic(err)
			}

			if _, err = f.Write(pendingData); err != nil {
				panic(err)
			}
			f.Close()
			// confirm after we can assure that data has been written
		}
		room.mux.Unlock()
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
