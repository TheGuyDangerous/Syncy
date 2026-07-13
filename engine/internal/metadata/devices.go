package metadata

import (
	"database/sql"
	"errors"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func (s *Store) PutDevice(d core.Device) error {
	if d.ID == "" {
		return errors.New("metadata: device ID must not be empty")
	}
	_, err := s.db.Exec(
		`INSERT INTO devices (id, name, trusted, pending_outgoing, endpoints, last_seen, added_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name             = excluded.name,
		     trusted          = excluded.trusted,
		     pending_outgoing = excluded.pending_outgoing,
		     endpoints        = excluded.endpoints,
		     last_seen        = excluded.last_seen`,
		string(d.ID), d.Name, boolToInt(d.Trusted), boolToInt(d.PendingOutgoing),
		listToJSON(d.Endpoints), unixOrZero(d.LastSeen), unixOrZero(d.AddedAt),
	)
	return err
}

func (s *Store) GetDevice(id core.DeviceID) (core.Device, error) {
	row := s.db.QueryRow(
		`SELECT id, name, trusted, pending_outgoing, endpoints, last_seen, added_at FROM devices WHERE id = ?`,
		string(id),
	)
	d, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Device{}, ErrNotFound
	}
	return d, err
}

func (s *Store) ListDevices() ([]core.Device, error) {
	rows, err := s.db.Query(
		`SELECT id, name, trusted, pending_outgoing, endpoints, last_seen, added_at FROM devices ORDER BY added_at, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []core.Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *Store) RemoveDevice(id core.DeviceID) error {
	res, err := s.db.Exec(`DELETE FROM devices WHERE id = ?`, string(id))
	if err != nil {
		return err
	}
	return requireAffected(res)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDevice(sc scanner) (core.Device, error) {
	var (
		id, name, endpoints string
		trusted, pending    int
		lastSeen, addedAt   int64
	)
	if err := sc.Scan(&id, &name, &trusted, &pending, &endpoints, &lastSeen, &addedAt); err != nil {
		return core.Device{}, err
	}
	return core.Device{
		ID:              core.DeviceID(id),
		Name:            name,
		Trusted:         trusted != 0,
		PendingOutgoing: pending != 0,
		Endpoints:       listFromJSON(endpoints),
		LastSeen:        timeFromUnix(lastSeen),
		AddedAt:         timeFromUnix(addedAt),
	}, nil
}
