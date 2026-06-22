# gochat

A real-time WebSocket chat server in Go with rooms, message history, client-side reconnection, and horizontal scaling via Redis Pub/Sub.

## Requirements

- Go 1.22+
- PostgreSQL
- Redis

## Setup

**1. Create a `.env` file:**

```env
DATABASE_URL=postgres://user:password@localhost:5432/chatdb?sslmode=disable
REDIS_ADDR=localhost:6379
```

**2. Run:**

```bash
go mod tidy
go run .
# server listening on :8080
```

Open `http://localhost:8080` in two or more browser tabs — enter a room name, set your name, and start chatting.

## Project layout

```
gochat/
├── main.go          entrypoint, HTTP routes, startup
├── hub.go           Hub: central goroutine, owns local clients map
├── client.go        Client: WebSocket connection + read/write pumps
├── store.go         PostgreSQL: persist and query message history
├── pubsub.go        Redis Pub/Sub: cross-instance broadcast
├── config.go        env config loader (.env via godotenv)
└── static/
    └── index.html   browser chat UI with reconnection logic
```

## How it works

### Connection lifecycle

Each browser connection becomes a `Client` with two goroutines:

- `readPump` — reads frames from the WebSocket; routes registration commands (`Room name:`, `Name:`) and chat messages to the hub
- `writePump` — drains `client.send` and writes frames to the WebSocket; sends WebSocket pings on a timer to detect dead connections

The `Hub` runs in a single goroutine and owns the `clients` map — no mutex needed. All state changes (`register`, `unregister`, `broadcast`) flow through channels.

### Message history

When both `room` and `name` are set (registration complete), the hub queries the last 20 messages for that room from PostgreSQL and writes them into `client.send`. New joiners see recent history immediately without waiting for new activity.

Every broadcast is persisted to PostgreSQL before fan-out, so history survives server restarts.

### Horizontal scaling

The in-process `broadcast` channel is not used for fan-out directly. Instead:

1. On the first client joining a room, the hub subscribes to a Redis channel (`room:<name>`)
2. Every message is published to Redis via `Publish`
3. A per-room `listenRoom` goroutine receives from Redis and routes payloads back into the hub via `localSend` — a dedicated channel that keeps all map access on the single `Hub.Run` goroutine
4. When the last client leaves a room, the Redis subscription is cancelled

This means any number of server instances behind a load balancer will all receive and fan out every message. Redis channels are scoped per room, so idle rooms consume no subscription resources.

### Client-side reconnection

The browser UI wraps the `WebSocket` constructor in a `connect()` function that is called recursively on `onclose` with exponential backoff (1 s → 2 s → 4 s … capped at 30 s, with ±10% jitter). On reconnect, `room` and `name` are re-sent automatically so the server re-registers the client and replays history.

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL DSN (`postgres://user:pass@host/db?sslmode=disable`) |
| `REDIS_ADDR` | Redis address (`host:port`, e.g. `localhost:6379`) |

## Schema

The `messages` table is created automatically on first boot:

```sql
CREATE TABLE IF NOT EXISTS messages (
    id         BIGSERIAL PRIMARY KEY,
    room       TEXT      NOT NULL,
    name       TEXT      NOT NULL,
    data       TEXT      NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS messages_room_id ON messages (room, id);
```
