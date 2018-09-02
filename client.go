package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type roomstate struct {
	offset uint64
	data   []byte
}

type client struct {
	conn     *websocket.Conn
	closedWG sync.WaitGroup
	send     chan []byte
	// outgoing messages that were not confirmed by the server
	unconfirmed              map[uint64][]byte
	nextExpectedConfirmation uint64

	rooms map[roomname]roomstate
}

func newClient() *client {
	return &client{
		send: make(chan []byte, 10),
	}
}

func (client *client) waitForConfs() {
	for len(client.unconfirmed) != 0 {
		time.Sleep(time.Millisecond * 10)
	}
}

func (client *client) readMessage(message []byte) {
	buf := bytes.NewBuffer(message)
	switch messageType, _ := buf.ReadByte(); messageType {
	case messageUpdate:
		confirmation, _ := binary.ReadUvarint(buf)
		roomname, _ := readRoomname(buf)
		bytes, _ := readPayload(buf)
		room := client.rooms[roomname]
		room.data = append(room.data, bytes...)
		client.send <- createMessageConfirmation(confirmation)
	case messageConfirmation:
		conf, _ := binary.ReadUvarint(buf)
		for conf >= client.nextExpectedConfirmation {
			delete(client.unconfirmed, client.nextExpectedConfirmation)
			client.nextExpectedConfirmation++
		}
	}
}

func (client *client) connect(url string) (err error) {
	if client.conn == nil {
		client.closedWG = sync.WaitGroup{}
		client.closedWG.Add(2)
		client.conn, _, err = websocket.DefaultDialer.Dial(url, nil)
		doneReading := make(chan struct{}, 0)
		// read pump
		go func() {
			defer func() {
				client.closedWG.Done()
				close(doneReading)
			}()
			for {
				_, message, err := client.conn.ReadMessage()
				if err != nil {
					fmt.Printf("ydb-client error: %s", err)
					close(doneReading)
					client.disconnect()
					break
				}
				client.readMessage(message)
			}
		}()
		// write pump
		go func() {
			defer func() {
				client.conn.Close()
				client.closedWG.Done()
			}()
			for m := range client.send {
				client.conn.WriteMessage(websocket.BinaryMessage, m)
			}
			err := client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("ydb-client error: error while closing conn", err)
				return
			}
			select {
			case <-doneReading:
			case <-time.After(time.Second):
			}
		}()
	}
	return
}

func (client *client) disconnect() {
	if client.conn != nil {
		close(client.send)
		client.closedWG.Wait()
		client.conn.Close()
		client.conn = nil
	}
}
