// Package metadata is the SQLite-backed store that holds the engine's local
// state: the devices it knows, the folders it shares, and (in later milestones)
// the file index, blocks, versions and offline queue. It is the local source of
// truth that the sync engine reconciles against.
package metadata

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo), registers "sqlite"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("metadata: not found")

// Store is a handle to the metadata database.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the metadata database at path. It enables
// WAL journaling, foreign keys and a busy timeout, then applies any pending
// schema migrations.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)",
		path,
	)
	return open(dsn)
}

// OpenMemory opens a private in-memory database, primarily for tests.
func OpenMemory() (*Store, error) {
	return open("file::memory:?_pragma=foreign_keys(on)")
}

func open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection serializes access, which keeps the in-memory database
	// alive for its lifetime and avoids "database is locked" races on all
	// platforms. Metadata operations are cheap, so this is an acceptable
	// trade-off for now.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrations is an ordered list of schema versions. Each entry is a set of
// statements applied atomically. Never edit or reorder an existing entry once
// released — only append new ones.
var migrations = [][]string{
	{ // v1: devices and folders
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
}

// migrate brings the database schema up to the latest version, applying each
// pending migration in its own transaction.
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
	// PRAGMA user_version cannot be parameterized, but index is an int we
	// control, so formatting it is safe.
	if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", index+1)); err != nil {
		return fmt.Errorf("set schema version v%d: %w", index+1, err)
	}
	return tx.Commit()
}

// schemaVersion returns the current on-disk schema version (exposed for tests).
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
