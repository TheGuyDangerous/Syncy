// Package metadata is the SQLite-backed store for the engine's local state.
package metadata

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("metadata: not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)",
		path,
	)
	return open(dsn)
}

func OpenMemory() (*Store, error) {
	return open("file::memory:?_pragma=foreign_keys(on)")
}

func open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

var migrations = [][]string{
	{
		`CREATE TABLE devices (
			id        TEXT PRIMARY KEY,
			name      TEXT    NOT NULL DEFAULT '',
			trusted   INTEGER NOT NULL DEFAULT 0,
			last_seen INTEGER NOT NULL DEFAULT 0,
			added_at  INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE folders (
			id        TEXT PRIMARY KEY,
			label     TEXT    NOT NULL DEFAULT '',
			path      TEXT    NOT NULL,
			direction TEXT    NOT NULL DEFAULT 'sendreceive',
			paused    INTEGER NOT NULL DEFAULT 0,
			added_at  INTEGER NOT NULL DEFAULT 0
		)`,
	},
	{
		`CREATE TABLE queue (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id  TEXT    NOT NULL,
			folder_id  TEXT    NOT NULL DEFAULT '',
			kind       TEXT    NOT NULL,
			payload    TEXT    NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			attempts   INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX idx_queue_device ON queue(device_id, id)`,
	},
}

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	for i := version; i < len(migrations); i++ {
		if err := s.applyMigration(i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyMigration(index int) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, stmt := range migrations[index] {
		if _, err = tx.Exec(stmt); err != nil {
			return fmt.Errorf("apply migration v%d: %w", index+1, err)
		}
	}
	if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", index+1)); err != nil {
		return fmt.Errorf("set schema version v%d: %w", index+1, err)
	}
	return tx.Commit()
}

func (s *Store) schemaVersion() (int, error) {
	var v int
	err := s.db.QueryRow("PRAGMA user_version").Scan(&v)
	return v, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func timeFromUnix(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}
