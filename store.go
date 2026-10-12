package boxmodel

import (
	"context"
	"sync"
)

// Store is the only dependency needed by the HTTP adapter. Revision zero is empty.
type Store interface {
	Snapshot(context.Context, *uint64) (Snapshot, error)
	Apply(context.Context, Patch) (Commit, error)
	History(context.Context, uint64, int) ([]Commit, error)
}
type Memory struct {
	mu        sync.RWMutex
	snapshots []Snapshot
	commits   []Commit
}

func NewMemory() *Memory { return &Memory{snapshots: []Snapshot{Empty()}} }
func (m *Memory) Snapshot(ctx context.Context, revision *uint64) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	r := uint64(len(m.snapshots) - 1)
	if revision != nil {
		r = *revision
	}
	if r >= uint64(len(m.snapshots)) {
		return Snapshot{}, ErrNotFound
	}
	return m.snapshots[r].Clone(), nil
}
func (m *Memory) Apply(ctx context.Context, p Patch) (Commit, error) {
	if err := ctx.Err(); err != nil {
		return Commit{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, c, err := Resolve(m.snapshots[len(m.snapshots)-1], p)
	if err != nil {
		return c, err
	}
	m.snapshots = append(m.snapshots, s)
	c = cloneCommit(c)
	m.commits = append(m.commits, c)
	return cloneCommit(c), nil
}
func cloneCommit(c Commit) Commit { // reuse the JSON boundary to avoid sharing caller-owned slices
	return copyCommit(c)
}
func (m *Memory) History(ctx context.Context, after uint64, limit int) ([]Commit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []Commit{}
	if limit <= 0 {
		limit = 200
	}
	for _, c := range m.commits {
		if c.Revision > after {
			result = append(result, cloneCommit(c))
			if len(result) == limit {
				break
			}
		}
	}
	return result, nil
}
