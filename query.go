package boxmodel

import (
	"errors"
	"fmt"
)

type Filter struct {
	Scope           Scope    `json:"scope"`
	Predicates      []string `json:"predicates,omitempty"`
	Dispositions    []string `json:"dispositions,omitempty"`
	Applicabilities []string `json:"applicabilities,omitempty"`
	Hierarchy       string   `json:"hierarchy,omitempty"`
	Interpretation  string   `json:"interpretation,omitempty"`
}

func includes(list []string, value string) bool {
	if len(list) == 0 {
		return true
	}
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
func (f Filter) matches(a Assertion) bool {
	return compatible(a.Scope, f.Scope) && includes(f.Predicates, a.Predicate) && includes(f.Dispositions, a.Disposition) && includes(f.Applicabilities, a.Applicability)
}
func (s Snapshot) parents(f Filter) (map[string]string, error) {
	if f.Hierarchy == "" {
		f.Hierarchy = "architecture"
	}
	result := map[string]string{}
	for _, id := range keys(s.Assertions) {
		a := s.Assertions[id]
		if a.Predicate != "contains" || a.Hierarchy != f.Hierarchy || a.Interpretation != f.Interpretation || a.Disposition != "accepted" || a.Applicability != "applicable" || !compatible(a.Scope, f.Scope) {
			continue
		}
		if old, ok := result[a.To]; ok && old != a.From {
			return nil, errors.New("ambiguous containment: select a scope or interpretation")
		}
		result[a.To] = a.From
	}
	return result, checkForest(result)
}
func descendants(root string, parents map[string]string) map[string]bool {
	result := map[string]bool{root: true}
	for id := range parents {
		for n := id; n != ""; n = parents[n] {
			if n == root {
				result[id] = true
				break
			}
		}
	}
	return result
}

type Query struct {
	Kind     string `json:"kind"`
	EntityID string `json:"entityId,omitempty"`
	To       string `json:"to,omitempty"`
	Filter   Filter `json:"filter"`
	Limit    int    `json:"limit,omitempty"`
	MaxDepth int    `json:"maxDepth,omitempty"`
}
type QueryResult struct {
	Revision              uint64     `json:"revision"`
	Query                 Query      `json:"query"`
	EntityIDs             []string   `json:"entityIds"`
	AssertionIDs          []string   `json:"assertionIds"`
	Paths                 [][]string `json:"paths"`
	Evidence              []Evidence `json:"evidence"`
	Truncated             bool       `json:"truncated"`
	KnowledgeCompleteness string     `json:"knowledgeCompleteness"`
}

// Query runs paths over canonical endpoints, never over projected aggregate edges.
func (s Snapshot) Query(q Query) (QueryResult, error) {
	r := QueryResult{Revision: s.Revision, Query: q, EntityIDs: []string{}, AssertionIDs: []string{}, Paths: [][]string{}, Evidence: []Evidence{}, KnowledgeCompleteness: "unknown"}
	if q.Limit == 0 {
		q.Limit = 200
	}
	if q.MaxDepth == 0 {
		q.MaxDepth = 8
	}
	if q.Limit < 1 || q.Limit > 10000 || q.MaxDepth < 1 || q.MaxDepth > 64 {
		return r, errors.New("limit must be 1..10000; maxDepth 1..64")
	}
	r.Query = q
	if q.EntityID != "" {
		if _, ok := s.Entities[q.EntityID]; !ok {
			return r, ErrNotFound
		}
	}
	if q.Kind == "paths" {
		if _, ok := s.Entities[q.To]; !ok {
			return r, ErrNotFound
		}
	}
	parents, err := s.parents(q.Filter)
	if err != nil {
		return r, err
	}
	region := descendants(q.EntityID, parents)
	addEntity := func(id string) {
		if len(r.EntityIDs) < q.Limit {
			r.EntityIDs = append(r.EntityIDs, id)
		} else {
			r.Truncated = true
		}
	}
	addAssertion := func(id string) {
		if len(r.AssertionIDs) < q.Limit {
			r.AssertionIDs = append(r.AssertionIDs, id)
		} else {
			r.Truncated = true
		}
	}
	switch q.Kind {
	case "children", "descendants":
		if q.EntityID == "" {
			return r, errors.New("entityId required")
		}
		for _, id := range keys(s.Entities) {
			if (q.Kind == "children" && parents[id] == q.EntityID) || (q.Kind == "descendants" && region[id] && id != q.EntityID) {
				addEntity(id)
			}
		}
	case "boundary", "neighbours", "assertions", "unresolved":
		if (q.Kind == "boundary" || q.Kind == "neighbours") && q.EntityID == "" {
			return r, errors.New("entityId required")
		}
		for _, id := range keys(s.Assertions) {
			a := s.Assertions[id]
			if !q.Filter.matches(a) {
				continue
			}
			match := false
			switch q.Kind {
			case "boundary":
				match = a.Predicate != "contains" && a.To != "" && (region[a.From] != region[a.To])
			case "neighbours":
				match = a.From == q.EntityID || a.To == q.EntityID
			case "assertions":
				match = q.EntityID == "" || a.From == q.EntityID || a.To == q.EntityID
			case "unresolved":
				match = a.Disposition == "proposed" || a.Applicability != "applicable"
				for _, e := range a.Evidence {
					match = match || e.Role == "contradicts"
				}
				match = match && (q.EntityID == "" || region[a.From] || region[a.To])
			}
			if match {
				addAssertion(id)
			}
		}
	case "paths":
		if q.EntityID == "" {
			return r, errors.New("entityId required")
		}
		edges := map[string][]Assertion{}
		for _, id := range keys(s.Assertions) {
			a := s.Assertions[id]
			if a.Predicate != "contains" && a.To != "" && q.Filter.matches(a) {
				edges[a.From] = append(edges[a.From], a)
			}
		}
		visits := 0
		stop := false
		seen := map[string]bool{q.EntityID: true}
		var walk func(string, []string)
		walk = func(node string, path []string) {
			if stop {
				return
			}
			visits++
			if visits > 100000 {
				r.Truncated = true
				stop = true
				return
			}
			if node == q.To {
				if len(r.Paths) == q.Limit {
					r.Truncated = true
					stop = true
					return
				}
				r.Paths = append(r.Paths, append([]string{}, path...))
				return
			}
			if len(path) >= q.MaxDepth {
				if len(edges[node]) > 0 {
					r.Truncated = true
				}
				return
			}
			for _, a := range edges[node] {
				if seen[a.To] {
					continue
				}
				seen[a.To] = true
				walk(a.To, append(path, a.ID))
				delete(seen, a.To)
			}
		}
		walk(q.EntityID, nil)
	default:
		return r, fmt.Errorf("unknown query kind %q", q.Kind)
	}
	evidence := map[string]bool{}
	for _, id := range r.AssertionIDs {
		for _, link := range s.Assertions[id].Evidence {
			evidence[link.EvidenceID] = true
		}
	}
	for _, path := range r.Paths {
		for _, id := range path {
			for _, link := range s.Assertions[id].Evidence {
				evidence[link.EvidenceID] = true
			}
		}
	}
	for _, id := range keys(evidence) {
		r.Evidence = append(r.Evidence, s.Evidence[id])
	}
	return r, nil
}

type Explanation struct {
	Revision   uint64      `json:"revision"`
	Assertions []Assertion `json:"assertions"`
	Evidence   []Evidence  `json:"evidence"`
}

func (s Snapshot) Explain(ids []string) (Explanation, error) {
	r := Explanation{Revision: s.Revision, Assertions: []Assertion{}, Evidence: []Evidence{}}
	seen := map[string]bool{}
	for _, id := range ids {
		a, ok := s.Assertions[id]
		if !ok {
			return r, ErrNotFound
		}
		r.Assertions = append(r.Assertions, a)
		for _, link := range a.Evidence {
			seen[link.EvidenceID] = true
		}
	}
	for _, id := range keys(seen) {
		r.Evidence = append(r.Evidence, s.Evidence[id])
	}
	return r, nil
}

type Change struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}
type Diff struct {
	From    uint64   `json:"from"`
	To      uint64   `json:"to"`
	Changes []Change `json:"changes"`
}

func Compare(before, after Snapshot) Diff {
	d := Diff{From: before.Revision, To: after.Revision, Changes: []Change{}}
	compareRecords(before.Entities, after.Entities, "entity", &d)
	compareRecords(before.Assertions, after.Assertions, "assertion", &d)
	compareRecords(before.Evidence, after.Evidence, "evidence", &d)
	return d
}
func compareRecords[T any](before, after map[string]T, kind string, d *Diff) {
	all := map[string]bool{}
	for id := range before {
		all[id] = true
	}
	for id := range after {
		all[id] = true
	}
	for _, id := range keys(all) {
		b, bok := before[id]
		a, aok := after[id]
		if bok && aok && same(b, a) {
			continue
		}
		c := Change{Kind: kind, ID: id}
		if bok {
			c.Before = b
		}
		if aok {
			c.After = a
		}
		d.Changes = append(d.Changes, c)
	}
}
