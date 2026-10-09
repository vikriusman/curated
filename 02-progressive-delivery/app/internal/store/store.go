// Package store talks to Postgres. The pool is the only state the process
// holds, and it is disposable: nothing in it survives a restart (12-factor VI).
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Note struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Create(ctx context.Context, body string) (Note, error) {
	var n Note
	err := s.pool.QueryRow(ctx,
		"INSERT INTO notes (body) VALUES ($1) RETURNING id, body, created_at", body,
	).Scan(&n.ID, &n.Body, &n.CreatedAt)
	return n, err
}

func (s *Store) List(ctx context.Context, limit int) ([]Note, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT id, body, created_at FROM notes ORDER BY id DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
