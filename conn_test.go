package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"time"
)

// testConn does not support reconnection
type testConn struct {
	// messages sent by conn to server
	incoming chan []byte
	roomData map[roomname][]byte
	outgoing chan []byte
	// next expected confirmation number from the server
	expectedConfirmation uint64
	// next confirmation number to create by the client
	nextConfirmation uint64
	sessionid        uint64
	session          *session
}

func newTestConn() *testConn {
	c := &testConn{
		incoming: make(chan []byte, 50),
		roomData: make(map[roomname][]byte, 1),
		outgoing: make(chan []byte, 50),
	}
	return c
}

func runTestClient(conn *testConn) {
	for {
		m := <-conn.incoming
		b := bytes.NewBuffer(m)
		readMessage(b, conn.session)
	}
}

func (conn *testConn) connect() {
	if conn.sessionid == 0 {
		sessionid, session := ydb.createSession()
		conn.session = session
		conn.sessionid = sessionid
	}
	go runTestClient(conn)
	ydb.sessions[conn.sessionid].add(conn)
}

func (conn *testConn) subscribe(roomname roomname, offset uint64) {
	m := createMessageSubscribe(conn.nextConfirmation, subDefinition{roomname, offset})
	conn.nextConfirmation++
	conn.incoming <- m
}

func (conn *testConn) updateRoomData(roomname roomname, data []byte) {
	if len(data) > 0 {
		m := createMessageUpdate(roomname, conn.nextConfirmation, data)
		conn.nextConfirmation++
		conn.incoming <- m
		conn.roomData[roomname] = append(conn.roomData[roomname], data...)
	}
}

func (conn *testConn) Write(m []byte) (n int, err error) {
	buf := bytes.NewBuffer(m)
	switch messageType, _ := buf.ReadByte(); messageType {
	case messageUpdate:
		confirmation, _ := binary.ReadUvarint(buf)
		roomname, _ := readRoomname(buf)
		bytes, _ := readPayload(buf)
		conn.roomData[roomname] = append(conn.roomData[roomname], bytes...)
		conn.incoming <- createMessageConfirmation(confirmation)
	case messageConfirmation:
		conf, _ := binary.ReadUvarint(buf)
		if conf != conn.expectedConfirmation {
			panic("Unexpected confirmation number!")
		}
		conn.expectedConfirmation++
	}
	return len(m), nil
}

func indexArrayItems(bs []byte) map[byte]uint64 {
	m := make(map[byte]uint64)
	for _, b := range bs {
		m[b]++
	}
	return m
}

func compareClients(t *testing.T, c1, c2 *testConn) {
	if len(c1.roomData) != len(c2.roomData) {
		t.Errorf("Clients roomData length does not match! %d != %d", len(c1.roomData), len(c2.roomData))
	}
	for roomname, c1data := range c1.roomData {
		c1index := indexArrayItems(c1data)
		c2data, _ := c2.roomData[roomname]
		c2index := indexArrayItems(c2data)
		if len(c1data) != len(c2data) {
			t.Errorf("Data length of room \"%s\" does not match (%d != %d)", roomname, len(c1data), len(c2data))
		}
		for b, occurences := range c1index {
			if occurences != c2index[b] {
				t.Errorf("room \"%s\": c1 has %d %ds. c2 has %d %ds", roomname, c1index[b], b, c2index[b], b)
			}
		}
	}
}

func (conn *testConn) waitForConfs() {
	for conn.expectedConfirmation != conn.nextConfirmation {
		time.Sleep(time.Millisecond * 10)
	}
}

func waitForClientConfs(clients ...*testConn) {
	for {
		synced := true
		for _, client := range clients {
			if client.expectedConfirmation != client.nextConfirmation {
				synced = false
				client.waitForConfs()
			}
		}
		ydb.sessionsMux.Lock()
		for _, session := range ydb.sessions {
			if session.serverConfirmation.next != session.serverConfirmation.nextClient {
				synced = false
			}
		}
		ydb.sessionsMux.Unlock()
		time.Sleep(time.Millisecond * 10)
		if synced {
			break
		}
	}
}

func initTestConns(numberOfConns int, f func(conns []*testConn)) {
	dir := "_ydb_conn_test"
	os.RemoveAll(dir)
	initYdb(dir)
	conns := make([]*testConn, numberOfConns)
	for i := 0; i < numberOfConns; i++ {
		c := newTestConn()
		c.connect()
		c.subscribe("test", 0)
		conns[i] = c
	}
	f(conns)
	os.RemoveAll(dir)
}

func TestWriteRoomDataAfterSubscription(t *testing.T) {
	initTestConns(2, func(conns []*testConn) {
		conns[0].updateRoomData("test", []byte{7})
		waitForClientConfs(conns...)
		compareClients(t, conns[0], conns[1])
		if len(conns[0].roomData["test"]) != 1 || conns[0].roomData["test"][0] != 7 {
			t.Error("not expected result")
		}
	})
}
