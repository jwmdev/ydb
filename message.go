package main

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
	messageType, err := binary.ReadUvarint(m)
	if err != nil {
		return err
	}
	switch messageType {
	case messageSub:
		err = readSubMessage(m, session)
	case messageUpdate:
		err = readUpdateMessage(m, session)
	case messageConfirmation:
		err = readConfirmationMessage(m, session)
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
		subscribeRoom(roomname, session, offset)
	}
	session.sendConfirmation(confirmation)
	return nil
}

func readConfirmationMessage(m message, session *session) (err error) {
	conf, err := binary.ReadUvarint(m)
	session.serverConfirmation.clientConfirmed(conf)
	return
}

type subDefinition struct {
	roomname roomname
	offset   uint64
}

func createMessageSubscribe(conf uint64, subs ...subDefinition) []byte {
	buf := &bytes.Buffer{}
	writeUvarint(buf, messageSub)
	writeUvarint(buf, conf)
	writeUvarint(buf, uint64(len(subs)))
	for _, sub := range subs {
		writeRoomname(buf, sub.roomname)
		writeUvarint(buf, sub.offset)
	}
	return buf.Bytes()
}

func createMessageUpdate(roomname roomname, conf uint64, data []byte) []byte {
	buf := &bytes.Buffer{}
	writeUvarint(buf, messageUpdate)
	writeUvarint(buf, conf)
	writeRoomname(buf, roomname)
	writePayload(buf, data)
	return buf.Bytes()
}

func createMessageConfirmation(conf uint64) []byte {
	buf := &bytes.Buffer{}
	writeUvarint(buf, messageConfirmation)
	writeUvarint(buf, conf)
	return buf.Bytes()
}

func readUpdateMessage(m message, session *session) error {
	confirmation, _ := binary.ReadUvarint(m)
	roomname, _ := readRoomname(m)
	bs, _ := readPayload(m)
	// send the rest of message
	updateRoom(roomname, session, confirmation, bs)
	return nil
}

func readString(m message) (string, error) {
	bs, err := readPayload(m)
	return string(bs), err
}

func readRoomname(m message) (roomname, error) {
	name, err := readString(m)
	return roomname(name), err
}

func readPayload(m message) ([]byte, error) {
	len, _ := binary.ReadUvarint(m)
	bs := make([]byte, len)
	m.Read(bs)
	return bs, nil
}

func writeUvarint(buf io.Writer, n uint64) error {
	bs := make([]byte, binary.MaxVarintLen64)
	len := binary.PutUvarint(bs, n)
	buf.Write(bs[:len])
	return nil
}

func writeString(buf io.Writer, str string) error {
	return writePayload(buf, []byte(str))
}

func writeRoomname(buf io.Writer, roomname roomname) error {
	return writeString(buf, string(roomname))
}

func writePayload(buf io.Writer, payload []byte) error {
	writeUvarint(buf, uint64(len(payload)))
	buf.Write(payload)
	return nil
}
