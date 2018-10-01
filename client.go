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
	offset int // TODO: this should reflect the offset stored by the server
	data   []byte
}

type client struct {
	conn     *websocket.Conn
	closedWG sync.WaitGroup
	send     chan []byte
	// outgoing messages that were not confirmed by the server
	unconfirmed              map[uint64][]byte
	nextExpectedConfirmation uint64
	nextConfirmationNumber   uint64
	rooms                    map[roomname]roomstate
}

func newClient() *client {
	return &client{
		send:        make(chan []byte, 10),
		unconfirmed: make(map[uint64][]byte),
		rooms:       make(map[roomname]roomstate),
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
		client.rooms[roomname] = room
		client.send <- createMessageConfirmation(confirmation)
	case messageConfirmation:
		conf, _ := binary.ReadUvarint(buf)
		for conf >= client.nextExpectedConfirmation {
			delete(client.unconfirmed, client.nextExpectedConfirmation)
			client.nextExpectedConfirmation++
		}
	}
}

func (client *client) WaitForConfs() {
	for len(client.unconfirmed) != 0 {
		time.Sleep(time.Millisecond * 10)
	}
}

func (client *client) Connect(url string) (err error) {
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
				if client.conn == nil {
					fmt.Println("Empty conn when reading, client prematurely disconnected")
				}
				messageType, message, err := client.conn.ReadMessage()
				if err != nil {
					fmt.Printf("ydb-client error: %s\n", err)
					close(doneReading)
					client.Disconnect()
					break
				}
				if messageType == websocket.BinaryMessage {
					client.readMessage(message)
				}
			}
		}()
		// write pump
		go func() {
			defer func() {
				client.conn.Close()
				client.closedWG.Done()
			}()
			for m := range client.send {
				fmt.Println("debug: write message to server: ", m)
				if client.conn == nil {
					fmt.Println("Empty conn when writing, client prematurely disconnected")
				}
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

func (client *client) Disconnect() {
	if client.conn != nil {
		close(client.send)
		client.closedWG.Wait()
		client.conn.Close()
		client.conn = nil
	}
}

func (client *client) Subscribe(subs ...subDefinition) {
	conf := client.nextConfirmationNumber
	m := createMessageSubscribe(conf, subs...)
	client.unconfirmed[conf] = m
	client.nextConfirmationNumber++
	client.send <- m
}

func (client *client) UpdateRoom(roomname roomname, data []byte) {
	conf := client.nextConfirmationNumber
	m := createMessageUpdate(roomname, conf, data)
	roomstate := client.rooms[roomname]
	roomstate.data = append(roomstate.data, data...)
	client.rooms[roomname] = roomstate
	client.unconfirmed[conf] = m
	client.nextConfirmationNumber++
	client.send <- m
}
