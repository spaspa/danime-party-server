package main

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub
	conn *websocket.Conn
	open bool
	send chan []byte

	sync.RWMutex

	id uuid.UUID
	roomId string
	ready bool
}

func (c *Client) readLoop() {
	// defer func() {
	// 	c.hub.unregister <- c
	// 	c.conn.Close()
	// }()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		t, m, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if t == websocket.TextMessage {
			m = bytes.TrimSpace(bytes.Replace(m, newline, space, -1))
			command := strings.Split(string(m), ":")
			err = c.handleCommand(command)
			if err != nil {
				_ = c.write(websocket.TextMessage, []byte(fmt.Sprintf("%s:%s", MessageError, err.Error())))
			}
		}
	}
}

func (c *Client) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.write(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.write(websocket.PingMessage, nil)
		}
	}
}

func (c *Client) write(messageType int, data []byte) error {
	c.Lock()
	defer c.Unlock()

	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, data)
}

func (c *Client) close() {
	if c.closed() {
		return
	}

	c.Lock()
	defer c.Unlock()

	c.open = false
	_ = c.conn.Close()
	close(c.send)
}

func (c *Client) closed() bool {
	c.RLock()
	defer c.RUnlock()
	return !c.open
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		// allow all origin for test
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		hub: hub,
		conn: conn,
		send: make(chan []byte, 256),
		open: true,
	}
	client.hub.register <- client

	go client.writeLoop()
	go client.readLoop()
}
