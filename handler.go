package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"strconv"
	"time"
)

func (c *Client) sendBroadcastToRoom(message string) {
	c.hub.broadcast <- &BroadcastMessage{
		message: []byte(message),
		roomId:  c.roomId,
	}
}

func (c *Client) sendTextMessage(message string) error {
	err := c.conn.WriteMessage(
		websocket.TextMessage,
		[]byte(message),
	)
	if err != nil {
		return ErrorUnknown
	}
	return nil
}

func (c *Client) sendOk() error {
	return c.sendTextMessage(MessageOk)
}

func (c *Client) sendReject(message string) error {
	return c.sendTextMessage(fmt.Sprintf("%s:%s", MessageReject, message))
}

func (c *Client) handleCommand(command []string) error {
	switch command[0] {
	case CommandPlay:
		return c.handleCommandPlay(command)
	case CommandPause:
		return c.handleCommandPause()
	case CommandSeek:
		return c.handleCommandSeek(command)
	case CommandResume:
		return c.handleCommandResume()
	case CommandSync:
		return c.handleCommandSync(command)
	case CommandJoin:
		return c.handleCommandJoin(command)
	case CommandLeave:
		return c.handleCommandLeave()
	case CommandReady:
		return c.handleCommandReady(command)
	}
	return ErrorBadRequest
}

// handleCommandStart retrieves `play!` command from client.
// command scheme: `play!:videoTime` -> `play:videoTime:unixTime` (broadcast)
func (c *Client) handleCommandPlay(command []string) error {
	if len(command) != 2 {
		return ErrorBadRequest
	}
	videoTime, err := strconv.ParseFloat(command[1], 64)
	if err != nil {
		return ErrorBadRequest
	}
	currentTime := float64(time.Now().UnixNano()) / 1000 / 1000 / 1000
	playTime := currentTime + 2

	if c.hub.rooms[c.roomId] == nil {
		return c.sendReject("no room")
	}
	for clientId := range c.hub.rooms[c.roomId] {
		if c.hub.clients[clientId] != nil && !c.hub.clients[clientId].ready {
			return c.sendReject("non-ready client exists")
		}
	}
	c.sendBroadcastToRoom(fmt.Sprintf("%s:%v:%v", MessagePlay, videoTime, playTime))
	return nil
}

// handleCommandPause retrieves `pause!` command from client.
// command scheme: `pause!` -> `pause` (broadcast)
func (c *Client) handleCommandPause() error {
	if c.hub.rooms[c.roomId] == nil {
		return c.sendReject("no room")
	}
	for clientId := range c.hub.rooms[c.roomId] {
		if client := c.hub.clients[clientId]; client != nil {
			client.ready = false
		}
	}
	c.sendBroadcastToRoom(MessagePause)
	return nil
}

// handleCommandStart retrieves `seek!` command from client.
// command scheme: `seek!:videoTime` -> `seek:videoTime` (broadcast)
func (c *Client) handleCommandSeek(command []string) error {
	if len(command) != 2 {
		return ErrorBadRequest
	}
	videoTime, err := strconv.ParseFloat(command[1], 64)
	if err != nil {
		return ErrorBadRequest
	}
	if c.hub.rooms[c.roomId] == nil {
		return c.sendReject("no room")
	}
	c.sendBroadcastToRoom(fmt.Sprintf("%s:%v", MessageSeek, videoTime))
	return nil
}

// handleCommandResume retrieves `resume!` command from client.
// command scheme: `resume!` -> `resume` (broadcast)
func (c *Client) handleCommandResume() error {
	if c.hub.rooms[c.roomId] == nil {
		return c.sendReject("no room")
	}
	c.sendBroadcastToRoom(MessageResume)
	return nil
}

// handleCommandSync retrieves `sync` command from client.
// command scheme: `sync:unixTime` -> `sync:timeDiff`
func (c *Client) handleCommandSync(command []string) error {
	if len(command) != 2 {
		return ErrorBadRequest
	}
	clientTime, err := strconv.ParseFloat(command[1], 64)
	if err != nil {
		return ErrorBadRequest
	}
	serverTime := float64(time.Now().UnixNano()) / 1000 / 1000 / 1000
	return c.sendTextMessage(fmt.Sprintf("%s:%v", MessageSync, serverTime-clientTime))
}

// handleCommandStart retrieves `join` command from client.
// command scheme: `join:roomId` -> `accept:clientId`
func (c *Client) handleCommandJoin(command []string) error {
	if len(command) != 2 {
		return ErrorBadRequest
	}
	roomId := command[1]
	if _, ok := c.hub.rooms[roomId]; ok {
		c.hub.rooms[roomId][c.id] = true
	} else {
		c.hub.rooms[roomId] = map[uuid.UUID]bool{c.id: true}
	}
	c.roomId = roomId
	return c.sendTextMessage(fmt.Sprintf("%s:%s:%s", MessageAccept, c.id.String(), c.roomId))
}

// handleCommandStart retrieves `leave` command from client.
// command scheme: `leave` -> `ok`
func (c *Client) handleCommandLeave() error {
	delete(c.hub.rooms[c.roomId], c.id)
	c.roomId = ""
	c.ready = false
	return c.sendOk()
}

// handleCommandStart retrieves `ready` command from client.
// command scheme: `ready:readyState` -> `ok`
func (c *Client) handleCommandReady(command []string) error {
	if len(command) != 2 {
		return ErrorBadRequest
	}
	c.ready = command[1] == "true"
	for clientId := range c.hub.rooms[c.roomId] {
		if c.hub.clients[clientId] != nil && !c.hub.clients[clientId].ready {
			return c.sendOk()
		}
	}
	return c.sendTextMessage(MessageReady)
}
