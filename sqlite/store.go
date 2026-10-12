// Package sqlite persists atomic semantic commits and materialized snapshots.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	model "github.com/philcantcode/box-view"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS snapshots (revision INTEGER PRIMARY KEY, data BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS commits (revision INTEGER PRIMARY KEY, data BLOB NOT NULL);
 INSERT OR IGNORE INTO snapshots(revision,data) VALUES(0,'{"revision":0,"entities":{},"assertions":{},"evidence":{}}');`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Snapshot(ctx context.Context, r *uint64) (model.Snapshot, error) {
	var b []byte
	var err error
	if r == nil {
		err = s.db.QueryRowContext(ctx, "SELECT data FROM snapshots ORDER BY revision DESC LIMIT 1").Scan(&b)
	} else {
		err = s.db.QueryRowContext(ctx, "SELECT data FROM snapshots WHERE revision=?", *r).Scan(&b)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return model.Snapshot{}, model.ErrNotFound
	}
	if err != nil {
		return model.Snapshot{}, err
	}
	var result model.Snapshot
	err = json.Unmarshal(b, &result)
	return result, err
}
func (s *Store) Apply(ctx context.Context, p model.Patch) (model.Commit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return model.Commit{}, err
	}
	defer conn.Close()
	// An immediate transaction serializes writers across separate server processes,
	// before the revision is read, rather than relying on an in-process mutex alone.
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return model.Commit{}, err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	var data []byte
	if err = conn.QueryRowContext(ctx, "SELECT data FROM snapshots ORDER BY revision DESC LIMIT 1").Scan(&data); err != nil {
		return model.Commit{}, err
	}
	var current model.Snapshot
	if err = json.Unmarshal(data, &current); err != nil {
		return model.Commit{}, err
	}
	next, c, err := model.Resolve(current, p)
	if err != nil {
		return c, err
	}
	snapshotData, err := json.Marshal(next)
	if err != nil {
		return c, err
	}
	commitData, err := json.Marshal(c)
	if err != nil {
		return c, err
	}
	if _, err = conn.ExecContext(ctx, "INSERT INTO snapshots(revision,data) VALUES(?,?)", next.Revision, snapshotData); err != nil {
		return c, err
	}
	if _, err = conn.ExecContext(ctx, "INSERT INTO commits(revision,data) VALUES(?,?)", c.Revision, commitData); err != nil {
		return c, err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return c, fmt.Errorf("commit: %w", err)
	}
	return c, nil
}
func (s *Store) History(ctx context.Context, after uint64, limit int) ([]model.Commit, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM commits WHERE revision>? ORDER BY revision LIMIT ?", after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Commit{}
	for rows.Next() {
		var b []byte
		var c model.Commit
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
