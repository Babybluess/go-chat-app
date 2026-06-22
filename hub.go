package main

import (
	"context"
	"fmt"
	"log"
)

type localMsg struct {
	room    string
	payload string
}

type Hub struct {
	clients    map[string]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	store      *Store
	pubsub     *PubSub
	cancels    map[string]func()
	localSend  chan localMsg
}

type Message struct {
	room string
	data []byte
	name string
}

func newHub(store *Store, pubsub *PubSub) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      store,
		pubsub:     pubsub,
		cancels:    make(map[string]func()),
		localSend:  make(chan localMsg, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.room] == nil {
				h.clients[client.room] = make(map[*Client]bool)
				ch, cancel := h.pubsub.Subscribe(context.Background(), client.room)
				h.cancels[client.room] = cancel
				go h.listenRoom(client.room, ch)
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
					if cancel, ok := h.cancels[client.room]; ok {
						cancel()
						delete(h.cancels, client.room)
					}
				}
			}

		case msg := <-h.broadcast:
			h.store.Save(msg.room, msg.name, string(msg.data))
			payload := fmt.Sprintf("%s: %s", msg.name, msg.data)
			if err := h.pubsub.Publish(context.Background(), msg.room, payload); err != nil {
				log.Println("publish:", err)
			}

		case lm := <-h.localSend:
			for client := range h.clients[lm.room] {
				select {
				case client.send <- []byte(lm.payload):
				default:
					close(client.send)
					delete(h.clients[lm.room], client)
				}
			}
		}
	}
}

func (h *Hub) listenRoom(room string, ch <-chan string) {
	for payload := range ch {
		h.localSend <- localMsg{room: room, payload: payload}
	}
}
