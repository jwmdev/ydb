package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsServer struct {
}

type wsConn struct {
	conn    *websocket.Conn
	session *session
	send    chan *websocket.PreparedMessage
}

func newWsConn(session *session, conn *websocket.Conn) *wsConn {
	return &wsConn{
		conn:    conn,
		session: session,
		send:    make(chan *websocket.PreparedMessage, 5),
	}
}

func (wsConn *wsConn) WriteMessage(m []byte, pm *websocket.PreparedMessage) {
	wsConn.send <- pm
}

func (wsConn *wsConn) readPump() {
	defer func() {
		wsConn.conn.Close()
		// TODO: unregister conn from ydb
		wsConn.session.removeConn(wsConn)
	}()
	wsConn.conn.SetReadLimit(maxMessageSize)
	wsConn.conn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.conn.SetPongHandler(func(string) error { wsConn.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		mbuffer := bytes.NewBuffer(message)
		for {
			err := readMessage(mbuffer, wsConn.session)
			if err != nil {
				break
			}
		}
	}
}

func (wsConn *wsConn) writePump() {
	conn := wsConn.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case message, ok := <-wsConn.send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := conn.WritePreparedMessage(message)
			if err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsConn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func setupWebsocketsListener(addr string) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("error: error upgrading client %s", err.Error())
			return
		}
		var sessionid uint64 // TODO: get the sessionid from http headers
		var session *session
		if sessionid == 0 {
			session = ydb.createSession()
		} else {
			session = ydb.getSession(sessionid)
		}
		wsConn := newWsConn(session, conn)
		go wsConn.readPump()
		go wsConn.writePump()
	})
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		exitBecause(err.Error())
	}
}
