package main

import (
	"fmt"
)

type Hub struct {
	clients    map[string]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	store      *Store
}

type Message struct {
	room string
	data []byte
	name string
}

func newHub(store *Store) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      store,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.room] == nil {
				h.clients[client.room] = make(map[*Client]bool)
			}
			h.clients[client.room][client] = true

			if client.room != "" && client.name != "" {
				for _, msg := range h.store.History(client.room, historyLimit) {
					select {
					case client.send <- msg:
					default:
					}
				}
			}

		case client := <-h.unregister:
			room := h.clients[client.room]
			if _, ok := room[client]; ok {
				delete(room, client)
				close(client.send)
				if len(room) == 0 {
					delete(h.clients, client.room)
				}
			}

		case msg := <-h.broadcast:
			h.store.Save(msg.room, msg.name, string(msg.data))
			for client := range h.clients[msg.room] {
				select {
				case client.send <- []byte(fmt.Sprintf("%s: %s", msg.name, msg.data)):
				default:
					close(client.send)
					delete(h.clients[msg.room], client)
				}
			}
		}
	}
}
