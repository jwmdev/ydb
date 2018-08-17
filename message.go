package ydb

import (
	"bytes"
	"encoding/binary"
	"io"
)

// message type constants
const (
	messageUpdate       = 0
	messageSub          = 1
	messageConfirmation = 2
)

// a message is structured as [length of payload, payload], where payload is [messageType, typePayload]
type message interface {
	ReadByte() (byte, error)
	Read(p []byte) (int, error)
}

func readMessage(m message, session *session) (err error) {
	// read length of message
	len, err := binary.ReadUvarint(m)
	if err != nil {
		return err
	}
	// prebuffering data to assure that all data exist, and we don't have to worry about err anymore
	payload := make([]byte, len)
	_, err = m.Read(payload)
	if err != nil && err != io.EOF {
		return err
	}
	buffered := bytes.NewReader(payload)
	messageType, _ := binary.ReadUvarint(buffered)
	switch messageType {
	case messageSub:
		err = readSubMessage(buffered, session)
	case messageUpdate:
		err = readUpdateMessage(buffered, session)
	}
	return err
}

func readSubMessage(m message, session *session) error {
	confirmation, _ := binary.ReadUvarint(m)
	nSubs, _ := binary.ReadUvarint(m)
	var i uint64
	for i = 0; i < nSubs; i++ {
		roomname, _ := readRoomname(m)
		offset, _ := binary.ReadUvarint(m)
		room := getRoom(roomname)
		room.subscribe(session, offset)
	}
	session.confirm(confirmation)
}

func createMessageUpdate(roomname roomname, conf uint64, data []byte) io.Reader {
	buf := &bytes.Buffer{}
	writeUvarint(buf, messageUpdate)
	writeUvarint(buf, conf)
	writeRoomname(buf, roomname)
	buf.Write(data)
	return buf
}

func readUpdateMessage(m message, session *session) error {
	confirmation, _ := binary.ReadUvarint(m)
	roomname, _ := readRoomname(m)
	room := getRoom(roomname)
	// send the rest of message
	// fswriter must confirm write to session
	room.update(session, confirmation, m)
}

func readString(m message) (string, error) {
	len, _ := binary.ReadUvarint(m)
	bs := make([]byte, len)
	m.Read(bs)
	return string(bs), nil
}

func readRoomname(m message) (roomname, error) {
	name, err := readString(m)
	return roomname(name), err
}

func writeUvarint(buf io.Writer, n uint64) error {
	bs := make([]byte, binary.MaxVarintLen64)
	len := binary.PutUvarint(bs, n)
	buf.Write(bs[:len])
	return nil
}

func writeString(buf io.Writer, str string) error {
	bs := []byte(str)
	writeUvarint(buf, uint64(len(bs)))
	buf.Write(bs)
	return nil
}
