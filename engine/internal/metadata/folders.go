package metadata

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func (s *Store) PutFolder(f core.Folder) error {
	if f.ID == "" {
		return errors.New("metadata: folder ID must not be empty")
	}
	if f.Path == "" {
		return errors.New("metadata: folder path must not be empty")
	}
	if f.Direction == "" {
		f.Direction = core.SendReceive
	}
	if !f.Direction.Valid() {
		return fmt.Errorf("metadata: invalid sync direction %q", f.Direction)
	}
	_, err := s.db.Exec(
		`INSERT INTO folders (id, label, path, direction, paused, added_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     label     = excluded.label,
		     path      = excluded.path,
		     direction = excluded.direction,
		     paused    = excluded.paused`,
		f.ID, f.Label, f.Path, string(f.Direction), boolToInt(f.Paused), unixOrZero(f.AddedAt),
	)
	return err
}

func (s *Store) GetFolder(id string) (core.Folder, error) {
	row := s.db.QueryRow(
		`SELECT id, label, path, direction, paused, added_at FROM folders WHERE id = ?`,
		id,
	)
	f, err := scanFolder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Folder{}, ErrNotFound
	}
	return f, err
}

func (s *Store) ListFolders() ([]core.Folder, error) {
	rows, err := s.db.Query(
		`SELECT id, label, path, direction, paused, added_at FROM folders ORDER BY added_at, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []core.Folder
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (s *Store) RemoveFolder(id string) error {
	res, err := s.db.Exec(`DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func scanFolder(sc scanner) (core.Folder, error) {
	var (
		id, label, path, direction string
		paused                     int
		addedAt                    int64
	)
	if err := sc.Scan(&id, &label, &path, &direction, &paused, &addedAt); err != nil {
		return core.Folder{}, err
	}
	return core.Folder{
		ID:        id,
		Label:     label,
		Path:      path,
		Direction: core.SyncDirection(direction),
		Paused:    paused != 0,
		AddedAt:   timeFromUnix(addedAt),
	}, nil
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
