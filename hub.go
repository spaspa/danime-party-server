// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/google/uuid"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	sync.RWMutex

	// Registered clients.
	clients map[uuid.UUID]*Client

	// Inbound messages from the clients.
	broadcast chan *BroadcastMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Map from room id to client id.
	rooms map[string]map[uuid.UUID]bool
}

type BroadcastMessage struct {
	message []byte
	roomId string
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
		rooms:      make(map[string]map[uuid.UUID]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			u, err := uuid.NewRandom()
			if err != nil {
				return
			}
			client.id = u

			h.Lock()
			h.clients[client.id] = client
			h.Unlock()
		case client := <-h.unregister:
			h.RLock()
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				close(client.send)
				if _, ok = h.rooms[client.roomId]; ok {
					h.Lock()
					delete(h.rooms[client.roomId], client.id)
					h.Unlock()
				}
			}
			h.RUnlock()
		case message := <-h.broadcast:
			h.RLock()
			for clientId, client := range h.clients {
				if message.roomId != client.roomId {
					continue
				}
				select {
				case client.send <- message.message:
				default:
					close(client.send)
					h.Lock()
					delete(h.clients, clientId)
					h.Unlock()
				}
			}
			h.RUnlock()
		}
	}
}
