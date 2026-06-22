package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	config, err := GetConfig()
	if err != nil {
		log.Fatal("config load:", err)
	}

	store := newStore(config.DB_URL)
	pubsub := newPubSub(config.REDIS_ADDR)

	hub := newHub(store, pubsub)
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Printf("server listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
