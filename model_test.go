package boxmodel_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	model "github.com/philcantcode/box-view"
	"github.com/philcantcode/box-view/example"
)

func seeded(t *testing.T) *model.Memory {
	t.Helper()
	m := model.NewMemory()
	if _, err := m.Apply(context.Background(), example.Patch(0)); err != nil {
		t.Fatal(err)
	}
	return m
}
func snap(t *testing.T, m model.Store) model.Snapshot {
	t.Helper()
	s, err := m.Snapshot(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestAtomicValidationAndIdentity(t *testing.T) {
	m := seeded(t)
	s := snap(t, m)
	entity := s.Entities["sync"]
	entity.Label = "Renamed"
	p := model.Patch{ExpectedRevision: 1, Actor: "test", Reason: "rename", Operations: []model.Operation{{Action: "update", Entity: &entity}}}
	if _, err := m.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	rev := uint64(1)
	old, _ := m.Snapshot(context.Background(), &rev)
	if old.Entities["sync"].Label != "Sync" || snap(t, m).Assertions["sync-writes"].From != "sync" {
		t.Fatal("rename changed identity or history")
	}
	p.ExpectedRevision = 1
	if _, err := m.Apply(context.Background(), p); !errors.Is(err, model.ErrConflict) {
		t.Fatal(err)
	}
	bad := model.NewPatch(2, "test", "cycle")
	bad.Entity("new", "component", "New")
	bad.Contains("cycle", "architecture", "db", "app")
	if _, err := m.Apply(context.Background(), bad.Patch); err == nil {
		t.Fatal("accepted cycle")
	}
	if snap(t, m).Revision != 2 || len(snap(t, m).Entities) != 5 {
		t.Fatal("partial commit")
	}
}
func TestValidation(t *testing.T) {
	tests := []struct {
		name   string
		change func(*model.Snapshot)
	}{
		{"endpoint", func(s *model.Snapshot) { a := s.Assertions["sync-writes"]; a.To = "missing"; s.Assertions[a.ID] = a }},
		{"multiple parents", func(s *model.Snapshot) {
			a := s.Assertions["storage-db"]
			a.ID = "other-parent"
			a.From = "sync"
			s.Assertions[a.ID] = a
		}},
		{"evidence scope", func(s *model.Snapshot) { e := s.Evidence["trace"]; e.Scope.Target = "other"; s.Evidence[e.ID] = e }},
		{"assessment", func(s *model.Snapshot) { a := s.Assertions["sync-writes"]; a.Basis = "proven"; s.Assertions[a.ID] = a }},
		{"duplicate identity", func(s *model.Snapshot) { e := s.Evidence["trace"]; e.ID = "sync"; s.Evidence[e.ID] = e }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := snap(t, seeded(t))
			tt.change(&s)
			if err := s.Validate(); err == nil {
				t.Fatal("accepted invalid model")
			}
		})
	}
}
func TestQueryProjectionExplanation(t *testing.T) {
	s := snap(t, seeded(t))
	q, err := s.Query(model.Query{Kind: "boundary", EntityID: "storage"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(q.AssertionIDs, []string{"sync-calls", "sync-writes"}) {
		t.Fatal(q)
	}
	p, err := s.Project(model.View{Collapsed: []string{"storage"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Boxes) != 3 || len(p.Edges) != 2 {
		t.Fatal(p)
	}
	for _, e := range p.Edges {
		if e.To != "storage" {
			t.Fatal(e)
		}
		x, err := s.Explain(e.Members)
		if err != nil || len(x.Assertions) != len(e.Members) {
			t.Fatal(x, err)
		}
	}
	q, _ = s.Query(model.Query{Kind: "descendants", EntityID: "app", Limit: 1})
	if !q.Truncated || len(q.EntityIDs) != 1 {
		t.Fatal(q)
	}
	q, _ = s.Query(model.Query{Kind: "assertions", Filter: model.Filter{Scope: model.Scope{Target: "other"}, Predicates: []string{"writes"}}})
	if len(q.AssertionIDs) != 0 {
		t.Fatal("scope leak")
	}
	p, _ = s.Project(model.View{Budget: 2})
	if len(p.Boxes) != 2 || p.OmittedEntities != 3 {
		t.Fatal(p)
	}
	if p.Boxes[1].Parent != p.Boxes[0].Entity.ID {
		t.Fatal("budget orphan")
	}
}
func TestCollapsedEdgesDoNotInventPaths(t *testing.T) {
	b := model.NewPatch(0, "test", "false path fixture")
	for _, id := range []string{"a", "b", "b1", "b2", "c"} {
		b.Entity(id, "component", id)
	}
	b.Contains("b-b1", "architecture", "b", "b1")
	b.Contains("b-b2", "architecture", "b", "b2")
	b.Relate("in", "a", "calls", "b1")
	b.Relate("out", "b2", "calls", "c")
	m := model.NewMemory()
	if _, err := m.Apply(context.Background(), b.Patch); err != nil {
		t.Fatal(err)
	}
	s := snap(t, m)
	p, _ := s.Project(model.View{Collapsed: []string{"b"}})
	if len(p.Edges) != 2 {
		t.Fatal(p)
	}
	q, err := s.Query(model.Query{Kind: "paths", EntityID: "a", To: "c"})
	if err != nil || len(q.Paths) != 0 {
		t.Fatal("invented path", q, err)
	}
}
func TestConcurrentConflictAndReplay(t *testing.T) {
	m := seeded(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := m.Apply(context.Background(), example.Patch(1)); results <- err }()
	}
	wg.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		if errors.Is(err, model.ErrConflict) {
			conflicts++
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if conflicts != 1 {
		t.Fatal(conflicts)
	}
	cs, _ := m.History(context.Background(), 0, 20)
	replay := model.NewMemory()
	for _, c := range cs {
		_, err := replay.Apply(context.Background(), model.Patch{ExpectedRevision: c.Parent, Actor: c.Actor, Reason: c.Reason, Operations: c.Operations})
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(snap(t, m), snap(t, replay)) {
		t.Fatal("replay differs")
	}
	// Inputs and results must not alias committed state.
	cs[0].Operations[0].Entity.Label = "mutation"
	if snap(t, m).Entities["app"].Label != "Application" {
		t.Fatal("history mutation")
	}
}
func TestExactAggregateMembersAndDiff(t *testing.T) {
	m := seeded(t)
	before := snap(t, m)
	b := model.NewPatch(1, "test", "second writer")
	b.Relate("extra", "sync", "writes", "storage-api")
	m.Apply(context.Background(), b.Patch)
	after := snap(t, m)
	p, _ := after.Project(model.View{Collapsed: []string{"storage"}})
	for _, e := range p.Edges {
		if e.Predicate == "writes" && !reflect.DeepEqual(e.Members, []string{"extra", "sync-writes"}) {
			t.Fatal(e)
		}
	}
	d := model.Compare(before, after)
	if len(d.Changes) != 1 || d.Changes[0].ID != "extra" {
		t.Fatal(d)
	}
}

func TestMoveAndCompetingScopedForests(t *testing.T) {
	m := seeded(t)
	s := snap(t, m)
	a := s.Assertions["storage-db"]
	a.From = "sync"
	if _, err := m.Apply(context.Background(), model.Patch{ExpectedRevision: 1, Actor: "test", Reason: "move database", Operations: []model.Operation{{Action: "update", Assertion: &a}}}); err != nil {
		t.Fatal(err)
	}
	s = snap(t, m)
	q, err := s.Query(model.Query{Kind: "children", EntityID: "sync"})
	if err != nil || !reflect.DeepEqual(q.EntityIDs, []string{"db"}) {
		t.Fatal(q, err)
	}
	if s.Assertions["sync-writes"].To != "db" {
		t.Fatal("move invalidated endpoint")
	}
	// Accepted alternatives can coexist in distinct scopes, but an unscoped view
	// must report ambiguity rather than silently picking a parent.
	a.Scope = model.Scope{Target: "one"}
	s.Assertions[a.ID] = a
	other := a
	other.ID = "alternative"
	other.From = "storage"
	other.Scope = model.Scope{Target: "two"}
	s.Assertions[other.ID] = other
	if err = s.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Project(model.View{}); err == nil {
		t.Fatal("silently chose a scoped parent")
	}
	p, err := s.Project(model.View{Filter: model.Filter{Scope: model.Scope{Target: "two"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, box := range p.Boxes {
		if box.Entity.ID == "db" && box.Parent != "storage" {
			t.Fatal(box)
		}
	}
}

func TestPresentationDefaultsOverridesAndValidation(t *testing.T) {
	s := snap(t, seeded(t))
	e := s.Entities["sync"]
	e.Description = "Synchronizes messages"
	s.Entities[e.ID] = e
	v := model.View{Layout: model.LayoutConfig{MaxBoxesPerRow: 4}, Styles: map[string]model.BoxStyle{"sync": {Colour: "#4f83cc", Span: 2}}}
	p, err := s.ProjectWithLayout(v, model.LayoutConfig{MaxBoxesPerRow: 2, MinBoxWidth: 300})
	if err != nil {
		t.Fatal(err)
	}
	if p.View.Layout.MaxBoxesPerRow != 4 || p.View.Layout.MinBoxWidth != 300 {
		t.Fatal(p.View.Layout)
	}
	for _, box := range p.Boxes {
		if box.Entity.ID == "sync" {
			if box.Style.Span != 2 || box.Style.Colour != "#4f83cc" || box.Entity.Description != e.Description {
				t.Fatal(box)
			}
		} else if box.Style.Span != 1 {
			t.Fatal("missing default span")
		}
	}
	for _, invalid := range []model.View{
		{Layout: model.LayoutConfig{MaxBoxesPerRow: 13}},
		{Layout: model.LayoutConfig{MinBoxWidth: 20}},
		{Styles: map[string]model.BoxStyle{"sync": {Colour: "red;url(invalid)"}}},
		{Styles: map[string]model.BoxStyle{"sync": {Span: -1}}},
		{Styles: map[string]model.BoxStyle{"missing": {Span: 1}}},
	} {
		if _, err := s.Project(invalid); err == nil {
			t.Fatal("invalid presentation accepted", invalid)
		}
	}
	empty, err := s.Project(model.View{})
	if err != nil || empty.View.Layout != model.DefaultLayout() {
		t.Fatal(empty.View.Layout, err)
	}
}
