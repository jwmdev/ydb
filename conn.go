package main

import (
	"github.com/gorilla/websocket"
)

type conn interface {
	// sends data to the client
	WriteMessage(m []byte, pm *websocket.PreparedMessage)
}
