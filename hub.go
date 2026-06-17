package main

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run processes all Hub events sequentially in a single goroutine.
// This is the central concurrency guarantee of the whole system.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // signals writePump to exit
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
					// queued successfully
				default:
					// send buffer full — client is too slow, drop it
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
