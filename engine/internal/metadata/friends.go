package metadata

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func (s *Store) PutFriendRequest(fr core.FriendRequest) error {
	if fr.FromID == "" {
		return errors.New("metadata: friend request device ID must not be empty")
	}
	_, err := s.db.Exec(
		`INSERT INTO friend_requests (from_id, name, endpoints, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(from_id) DO UPDATE SET
		     name      = excluded.name,
		     endpoints = excluded.endpoints`,
		string(fr.FromID), fr.Name, listToJSON(fr.Endpoints), unixOrZero(fr.CreatedAt),
	)
	return err
}

func (s *Store) GetFriendRequest(id core.DeviceID) (core.FriendRequest, error) {
	row := s.db.QueryRow(
		`SELECT from_id, name, endpoints, created_at FROM friend_requests WHERE from_id = ?`,
		string(id),
	)
	fr, err := scanFriendRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.FriendRequest{}, ErrNotFound
	}
	return fr, err
}

func (s *Store) ListFriendRequests() ([]core.FriendRequest, error) {
	rows, err := s.db.Query(
		`SELECT from_id, name, endpoints, created_at FROM friend_requests ORDER BY created_at, from_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []core.FriendRequest
	for rows.Next() {
		fr, err := scanFriendRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, fr)
	}
	return reqs, rows.Err()
}

func (s *Store) RemoveFriendRequest(id core.DeviceID) error {
	res, err := s.db.Exec(`DELETE FROM friend_requests WHERE from_id = ?`, string(id))
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func scanFriendRequest(sc scanner) (core.FriendRequest, error) {
	var (
		id, name, endpoints string
		createdAt           int64
	)
	if err := sc.Scan(&id, &name, &endpoints, &createdAt); err != nil {
		return core.FriendRequest{}, err
	}
	return core.FriendRequest{
		FromID:    core.DeviceID(id),
		Name:      name,
		Endpoints: listFromJSON(endpoints),
		CreatedAt: timeFromUnix(createdAt),
	}, nil
}

func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (s *Store) SetSetting(key, value string) error {
	if key == "" {
		return errors.New("metadata: setting key must not be empty")
	}
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func listToJSON(v []string) string {
	if len(v) == 0 {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func listFromJSON(s string) []string {
	if s == "" {
		return nil
	}
	var v []string
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return v
}
