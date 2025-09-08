package core

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed persistence layer.
type Store struct{ db *sql.DB }

//go:embed migrations/*.sql
var migrationFS embed.FS

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema, err := migrationFS.ReadFile("migrations/0001_init.sql")
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	if s.db == nil {
		return errors.New("db not initialized")
	}
	return s.db.PingContext(ctx)
}
