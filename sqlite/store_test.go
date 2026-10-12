package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	model "github.com/philcantcode/box-view"
	"github.com/philcantcode/box-view/example"
	"github.com/philcantcode/box-view/sqlite"
)

func TestDurabilityAtomicityAndCrossConnectionConflict(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "model.db")
	a, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Apply(ctx, example.Patch(0)); err != nil {
		t.Fatal(err)
	}
	expected, _ := a.Snapshot(ctx, nil)
	a.Close()
	a, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got, _ := a.Snapshot(ctx, nil)
	if !reflect.DeepEqual(expected, got) {
		t.Fatal("durability")
	}
	b, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var wg sync.WaitGroup
	ch := make(chan error, 2)
	for _, store := range []*sqlite.Store{a, b} {
		wg.Add(1)
		go func(s *sqlite.Store) { defer wg.Done(); _, err := s.Apply(ctx, example.Patch(1)); ch <- err }(store)
	}
	wg.Wait()
	close(ch)
	n := 0
	for err := range ch {
		if errors.Is(err, model.ErrConflict) {
			n++
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if n != 1 {
		t.Fatal(n)
	}
	bad := model.NewPatch(2, "test", "invalid")
	bad.Entity("partial", "component", "Partial")
	bad.Relate("bad", "missing", "calls", "db")
	if _, err = a.Apply(ctx, bad.Patch); err == nil {
		t.Fatal("invalid accepted")
	}
	got, _ = a.Snapshot(ctx, nil)
	if got.Revision != 2 || len(got.Entities) != 5 {
		t.Fatal("partial commit")
	}
	cs, _ := a.History(ctx, 0, 10)
	replay := model.NewMemory()
	for _, c := range cs {
		_, err = replay.Apply(ctx, model.Patch{ExpectedRevision: c.Parent, Actor: c.Actor, Reason: c.Reason, Operations: c.Operations})
		if err != nil {
			t.Fatal(err)
		}
	}
	memory, _ := replay.Snapshot(ctx, nil)
	if !reflect.DeepEqual(got, memory) {
		t.Fatal("replay")
	}
	revision := uint64(1)
	old, _ := a.Snapshot(ctx, &revision)
	if !reflect.DeepEqual(old, expected) {
		t.Fatal("historical state")
	}
}
