package metadata

import (
	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func (s *Store) EnqueueOp(op core.QueuedOp) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO queue (device_id, folder_id, kind, payload, created_at, attempts)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		string(op.DeviceID), op.FolderID, op.Kind, op.Payload, unixOrZero(op.CreatedAt), op.Attempts,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) PendingOps(deviceID core.DeviceID) ([]core.QueuedOp, error) {
	return s.queryOps(
		`SELECT id, device_id, folder_id, kind, payload, created_at, attempts
		 FROM queue WHERE device_id = ? ORDER BY id`,
		string(deviceID),
	)
}

func (s *Store) AllPendingOps() ([]core.QueuedOp, error) {
	return s.queryOps(
		`SELECT id, device_id, folder_id, kind, payload, created_at, attempts
		 FROM queue ORDER BY id`,
	)
}

func (s *Store) CompleteOp(id int64) error {
	res, err := s.db.Exec(`DELETE FROM queue WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) IncrementAttempts(id int64) error {
	res, err := s.db.Exec(`UPDATE queue SET attempts = attempts + 1 WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) QueueLen(deviceID core.DeviceID) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE device_id = ?`, string(deviceID)).Scan(&n)
	return n, err
}

func (s *Store) queryOps(query string, args ...any) ([]core.QueuedOp, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []core.QueuedOp
	for rows.Next() {
		var (
			op        core.QueuedOp
			deviceID  string
			createdAt int64
		)
		if err := rows.Scan(&op.ID, &deviceID, &op.FolderID, &op.Kind, &op.Payload, &createdAt, &op.Attempts); err != nil {
			return nil, err
		}
		op.DeviceID = core.DeviceID(deviceID)
		op.CreatedAt = timeFromUnix(createdAt)
		ops = append(ops, op)
	}
	return ops, rows.Err()
}
