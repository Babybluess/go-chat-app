package main

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type PubSub struct {
	rdb *redis.Client
}

func newPubSub(addr string) *PubSub {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis ping:", err)
	}
	return &PubSub{rdb: rdb}
}

func (p *PubSub) channel(room string) string {
	return "room:" + room
}

func (p *PubSub) Publish(ctx context.Context, room, payload string) error {
	return p.rdb.Publish(ctx, p.channel(room), payload).Err()
}

func (p *PubSub) Subscribe(ctx context.Context, room string) (<-chan string, func()) {
	sub := p.rdb.Subscribe(ctx, p.channel(room))
	ch := make(chan string)

	go func() {
		defer close(ch)
		for msg := range sub.Channel() {
			ch <- msg.Payload
		}
	}()

	return ch, func() { sub.Close() }
}
