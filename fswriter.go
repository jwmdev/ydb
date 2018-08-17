package ydb

import (
	"bytes"
	"fmt"
	"os"
)

type roomUpdate struct {
	room     *room
	roomname roomname
}

type fswriter struct {
	queue chan roomUpdate
}

func (fswriter *fswriter) registerRoomUpdate(room *room, roomname roomname) {
	fswriter.queue <- roomUpdate{room, roomname}
}

var fs *fswriter

func writeTask() {
	for {
		writeTask := <-fs.queue
		room := writeTask.room
		roomname := writeTask.roomname
		room.mux.Lock()
		pendingData := room.pendingData
		var pendingConfirmations []pendingConfirmation
		dataAvailable := false
		if pendingData.Len() > 0 {
			// New data is available.
			dataAvailable = true
			// This goroutine will save dataAvailable and confirm pendingConfirmations
			room.pendingData = bytes.Buffer{}
			pendingConfirmations = room.pendingConfirmations
			room.pendingConfirmations = nil
		}
		room.mux.Unlock()
		if dataAvailable {
			f, err := os.OpenFile(fmt.Sprintf("_files/%s", string(roomname)), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}

			if _, err = pendingData.WriteTo(f); err != nil {
				panic(err)
			}
			f.Close()
			// confirm after we can assure that data has been written
		}
	}
}

func initFSWriter(fsAccessQueueLen uint, writeConcurrency int) {
	fs = new(fswriter)
	fs.queue = make(chan roomUpdate, fsAccessQueueLen)
	for i := 0; i < writeConcurrency; i++ {
		go writeTask()
	}
}
