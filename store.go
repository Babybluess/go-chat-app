package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

const historyLimit = 20

type Store struct {
	db *sql.DB
}

func newStore(dsn string) *Store {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("store open:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("store ping:", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id         BIGSERIAL PRIMARY KEY,
		room       TEXT      NOT NULL,
		name       TEXT      NOT NULL,
		data       TEXT      NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS messages_room_id ON messages (room, id)`)
	return &Store{db: db}
}

func (s *Store) Save(room, name, data string) {
	s.db.Exec(`INSERT INTO messages (room, name, data) VALUES ($1, $2, $3)`, room, name, data)
}

// History returns the last n messages for a room, oldest-first.
func (s *Store) History(room string, n int) [][]byte {
	rows, err := s.db.Query(`
		SELECT name, data FROM (
			SELECT id, name, data FROM messages
			WHERE room = $1
			ORDER BY id DESC LIMIT $2
		) sub ORDER BY id ASC`, room, n)
	if err != nil {
		log.Println("store history:", err)
		return nil
	}
	defer rows.Close()

	var out [][]byte
	for rows.Next() {
		var name, data string
		rows.Scan(&name, &data)
		out = append(out, []byte(name+": "+data))
	}
	return out
}
