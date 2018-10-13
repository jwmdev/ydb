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
	writeWait = 50 * time.Second // TODO: make this configurable

	// Time allowed to read the next pong message from the peer.
	pongWait = 50 * time.Second // TODO: make this configurable

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000000
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: implement origin checking
	},
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
	defer func() {
		recover() // recover if channel is already closed
	}()
	debugMessageType("sending message to client..", m)
	wsConn.send <- pm
}

func (wsConn *wsConn) readPump() {
	wsConn.conn.SetReadLimit(maxMessageSize)
	wsConn.conn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.conn.SetPongHandler(func(string) error {
		wsConn.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ydb error: %v", err)
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
	wsConn.conn.Close()
	debug("ending read pump for ws conn")
	// TODO: unregister conn from ydb
}

func (wsConn *wsConn) writePump() {
	conn := wsConn.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		debug("ending write pump for ws conn")
		ticker.Stop()
		wsConn.session.removeConn(wsConn)
		close(wsConn.send)
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
				fmt.Println("server error when writing prepared message to conn", err)
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
	// TODO: only set this if in testing mode!
	http.HandleFunc("/clearAll", func(w http.ResponseWriter, r *http.Request) {
		unsafeClearAllYdbContent()
		w.WriteHeader(200)
		fmt.Fprintf(w, "OK")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("new client..")
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
		session.add(wsConn)
		go wsConn.readPump()
		go wsConn.writePump()
	})
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		exitBecause(err.Error())
	}
}
