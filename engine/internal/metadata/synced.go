package metadata

import (
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

func (s *Store) GetSyncedBaseline(folderID string) (map[string]hashing.Hash, error) {
	rows, err := s.db.Query(`SELECT path, hash FROM synced WHERE folder_id = ?`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]hashing.Hash)
	for rows.Next() {
		var path, hashHex string
		if err := rows.Scan(&path, &hashHex); err != nil {
			return nil, err
		}
		h, err := hashing.Parse(hashHex)
		if err != nil {
			return nil, err
		}
		out[path] = h
	}
	return out, rows.Err()
}

func (s *Store) SetSyncedBaseline(folderID string, baseline map[string]hashing.Hash) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM synced WHERE folder_id = ?`, folderID); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO synced (folder_id, path, hash) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for path, h := range baseline {
		if _, err = stmt.Exec(folderID, path, h.String()); err != nil {
			return err
		}
	}
	return tx.Commit()
}
