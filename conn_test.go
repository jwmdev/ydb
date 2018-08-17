package ydb

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// testConn does not support reconnection
type testConn struct {
	// messages sent by conn to server
	messages             chan []byte
	roomData             map[roomname][]byte
	expectedConfirmation uint64
	nextConfirmation     uint64
}

func (conn *testConn) updateRoomData(roomname roomname, data []byte) {
	createMessageUpdate(roomname, conn.nextConfirmation, data)
	conn.nextConfirmation++
	conn.roomData[roomname] = append(conn.roomData[roomname], data...)
}

func (conn *testConn) send(m []byte) {
	conf := conn.nextConfirmation
	conn.nextConfirmation++
}

func (conn *testConn) readMessage() (m []byte, err error) {
	return <-conn.messages, nil
}

func (conn *testConn) Write(m []byte) (n int, err error) {
	buf := bytes.NewBuffer(m)
	switch messageType, _ := buf.ReadByte(); messageType {
	case messageUpdate:
		confirmation, _ := binary.ReadUvarint(buf)
		roomname, _ := readRoomname(buf)
		conn.roomData[roomname] = append(conn.roomData[roomname], buf.Bytes()...)
		conn.send(createMessageConfirmation(confirmation))
	case messageConfirmation:
		conf, _ := binary.ReadUvarint(buf)
		if conf != conn.expectedConfirmation {
			panic("Unexpected confirmation number!")
		}
		conn.expectedConfirmation++
	}
	return len(m), nil
}

func TestConn(t *testing.T) {

}
