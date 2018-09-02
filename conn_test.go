package main

import (
	"bytes"
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

func runTestConn(conn *testConn) {
	for {
		m := <-conn.incoming
		b := bytes.NewBuffer(m)
		readMessage(b, conn.session)
	}
}

/*
func (conn *testConn) connect() {
	if conn.sessionid == 0 {
		sessionid, session 	= ydb.createSession()
		conn.session = sess	on
		conn.sessionid = sessionid
	}
	go runTestClient(conn)
	ydb.sessions[conn.sessionid].add(conn)
}



func (conn *testConn) updateRoomData(roomname roomname, data []byte) {
	if len(data) > 0 {
		m := createMessageUpdate(roomname, conn.nextConfirmation, data)
		conn.nextConfirmation++
		conn.incoming <- m
		conn.roomData[roomname] = append(conn.roomData[roomname], data...)
	}
}

func indexArrayItems(bs []byte) map[byte]uint64 {
	m := make(map[byte]uint64)
	for _, b := range bs {
		m[b]++
	}
	return m
}

func compareConns(t *testing.T, c1, c2 *testConn) {
	if len(c1.roomData) != len(c2.roomData) {
		t.Errorf("Conns roomData length does not match! %d != %d", len(c1.roomData), len(c2.roomData))
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

func waitForConnConfs(conns ...*testConn) {
	for {
		synced := true
		for _, conn := range conns {
			if conn.expectedConfirmation != conn.nextConfirmation {
				synced = false
				conn.waitForConfs()
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




*/
