# gochat

A real-time WebSocket chat server in Go using a single-Hub concurrency model.

## Setup

```bash
go mod tidy
go run .
# server listening on :8080
```

Open `http://localhost:8080` in two browser tabs — messages sent in one appear instantly in both.

## Project layout

```
gochat/
├── main.go          entrypoint, HTTP routes
├── hub.go           Hub: central goroutine, owns clients map
├── client.go        Client: WebSocket connection + read/write pumps
└── static/
    └── index.html   browser chat UI
```

## How it works

Each connected browser gets a `Client` with two goroutines:
- `readPump` — reads from the WebSocket, pushes to `hub.broadcast`
- `writePump` — reads from `client.send`, writes to the WebSocket

The Hub runs in a single goroutine and owns the `clients` map with no mutex needed. All state changes flow through channels.

## Next steps

- Add usernames (first message after connect sets the name)
- Add chat rooms (`map[string]map[*Client]bool` keyed by room)
- Add message history (query last N messages on register)
- Add a JSON message protocol (`type`, `user`, `body`, `at` fields)
- Add client-side reconnection with exponential backoff
- Scale horizontally with Redis Pub/Sub replacing the in-process broadcast channel
