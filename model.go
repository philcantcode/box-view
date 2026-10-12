// Package boxmodel records explainable software models independently of their views.
package boxmodel

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const Version = "0.1.0"

var ErrConflict = errors.New("revision conflict")
var ErrNotFound = errors.New("not found")

type Scope struct {
	Target        string `json:"target,omitempty"`
	Run           string `json:"run,omitempty"`
	Configuration string `json:"configuration,omitempty"`
}
type Entity struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Kind        string   `json:"kind"`
	Description string   `json:"description,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
type Evidence struct {
	ID         string `json:"id"`
	URI        string `json:"uri"`
	Digest     string `json:"digest,omitempty"`
	Selector   string `json:"selector,omitempty"`
	Producer   string `json:"producer,omitempty"`
	CapturedAt string `json:"capturedAt,omitempty"`
	Scope      Scope  `json:"scope"`
}
type EvidenceLink struct {
	EvidenceID string `json:"evidenceId"`
	Role       string `json:"role"`
	Note       string `json:"note,omitempty"`
}
type Assertion struct {
	ID             string         `json:"id"`
	From           string         `json:"from"`
	To             string         `json:"to,omitempty"`
	Predicate      string         `json:"predicate"`
	Value          string         `json:"value,omitempty"`
	Hierarchy      string         `json:"hierarchy,omitempty"`
	Interpretation string         `json:"interpretation,omitempty"`
	Basis          string         `json:"basis"`
	Disposition    string         `json:"disposition"`
	Applicability  string         `json:"applicability"`
	Scope          Scope          `json:"scope"`
	Evidence       []EvidenceLink `json:"evidence,omitempty"`
	Description    string         `json:"description,omitempty"`
}
type Snapshot struct {
	Revision   uint64               `json:"revision"`
	Entities   map[string]Entity    `json:"entities"`
	Assertions map[string]Assertion `json:"assertions"`
	Evidence   map[string]Evidence  `json:"evidence"`
}

func Empty() Snapshot {
	return Snapshot{Entities: map[string]Entity{}, Assertions: map[string]Assertion{}, Evidence: map[string]Evidence{}}
}
func (s Snapshot) Clone() Snapshot {
	b, _ := json.Marshal(s)
	var copy Snapshot
	_ = json.Unmarshal(b, &copy)
	return copy
}

// Operation creates an immutable identity, or explicitly updates an existing record.
// Withdraw assertions by changing their disposition; records are retained for explanation.
type Operation struct {
	Action    string     `json:"action"`
	Entity    *Entity    `json:"entity,omitempty"`
	Assertion *Assertion `json:"assertion,omitempty"`
	Evidence  *Evidence  `json:"evidence,omitempty"`
}
type Patch struct {
	ExpectedRevision uint64      `json:"expectedRevision"`
	Actor            string      `json:"actor"`
	Reason           string      `json:"reason"`
	Operations       []Operation `json:"operations"`
}
type Commit struct {
	Revision   uint64      `json:"revision"`
	Parent     uint64      `json:"parent"`
	Actor      string      `json:"actor"`
	Reason     string      `json:"reason"`
	Time       time.Time   `json:"time"`
	Operations []Operation `json:"operations"`
	Affected   []string    `json:"affected"`
}

func oneOf(value string, allowed ...string) bool {
	for _, v := range allowed {
		if v == value {
			return true
		}
	}
	return false
}
func compatible(a, b Scope) bool {
	return (a.Target == "" || b.Target == "" || a.Target == b.Target) && (a.Run == "" || b.Run == "" || a.Run == b.Run) && (a.Configuration == "" || b.Configuration == "" || a.Configuration == b.Configuration)
}
func validID(id string) bool {
	return id != "" && len(id) <= 256 && strings.TrimSpace(id) == id && !strings.ContainsAny(id, "\r\n\x00")
}
func same(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func put[T any](m map[string]T, id string, v T, action string) error {
	if !validID(id) {
		return fmt.Errorf("invalid ID %q", id)
	}
	old, ok := m[id]
	if action == "create" && ok {
		if same(old, v) {
			return nil
		}
		return fmt.Errorf("%s exists; use update", id)
	}
	if action == "update" && !ok {
		return fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	m[id] = v
	return nil
}

// Resolve validates the entire candidate snapshot before any state is committed.
func Resolve(s Snapshot, p Patch) (Snapshot, Commit, error) {
	if p.ExpectedRevision != s.Revision {
		return s, Commit{}, ErrConflict
	}
	if strings.TrimSpace(p.Actor) == "" || strings.TrimSpace(p.Reason) == "" || len(p.Operations) == 0 {
		return s, Commit{}, errors.New("actor, reason and operations required")
	}
	next := s.Clone()
	affected := map[string]bool{}
	for _, op := range p.Operations {
		if !oneOf(op.Action, "create", "update") {
			return s, Commit{}, errors.New("action must be create or update")
		}
		count := 0
		id := ""
		var err error
		if op.Entity != nil {
			count++
			id = op.Entity.ID
			err = put(next.Entities, id, *op.Entity, op.Action)
		}
		if op.Assertion != nil {
			count++
			id = op.Assertion.ID
			err = put(next.Assertions, id, *op.Assertion, op.Action)
		}
		if op.Evidence != nil {
			count++
			id = op.Evidence.ID
			err = put(next.Evidence, id, *op.Evidence, op.Action)
		}
		if count != 1 {
			return s, Commit{}, errors.New("operation requires exactly one record")
		}
		if err != nil {
			return s, Commit{}, err
		}
		affected[id] = true
	}
	if err := next.Validate(); err != nil {
		return s, Commit{}, err
	}
	next.Revision++
	c := Commit{Revision: next.Revision, Parent: s.Revision, Actor: p.Actor, Reason: p.Reason, Time: time.Now().UTC(), Operations: p.Operations, Affected: keys(affected)}
	return next, c, nil
}
func keys[T any](m map[string]T) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
func (s Snapshot) Validate() error {
	ids := map[string]bool{}
	check := func(id string) error {
		if !validID(id) || ids[id] {
			return fmt.Errorf("invalid or duplicate identity %q", id)
		}
		ids[id] = true
		return nil
	}
	for id, e := range s.Entities {
		if id != e.ID || e.Label == "" || e.Kind == "" {
			return fmt.Errorf("invalid entity %s", id)
		}
		if err := check(id); err != nil {
			return err
		}
	}
	for id, e := range s.Evidence {
		if id != e.ID || e.URI == "" {
			return fmt.Errorf("invalid evidence %s", id)
		}
		if err := check(id); err != nil {
			return err
		}
	}
	forests := map[string]map[string]string{}
	for id, a := range s.Assertions {
		if id != a.ID {
			return fmt.Errorf("assertion key mismatch %s", id)
		}
		if err := check(id); err != nil {
			return err
		}
		if _, ok := s.Entities[a.From]; !ok {
			return fmt.Errorf("%s: unknown source", id)
		}
		if a.To != "" {
			if _, ok := s.Entities[a.To]; !ok {
				return fmt.Errorf("%s: unknown destination", id)
			}
		} else if a.Value == "" {
			return fmt.Errorf("%s: destination or value required", id)
		}
		if !oneOf(a.Predicate, "contains", "calls", "reads", "writes", "publishes", "subscribes", "implements", "depends_on") && !strings.Contains(a.Predicate, ":") {
			return fmt.Errorf("%s: unknown predicate (extensions require namespace)", id)
		}
		if !oneOf(a.Basis, "declared", "static", "observed", "inferred") || !oneOf(a.Disposition, "proposed", "accepted", "rejected", "superseded") || !oneOf(a.Applicability, "applicable", "stale", "unresolved") {
			return fmt.Errorf("%s: invalid assessment", id)
		}
		for _, link := range a.Evidence {
			e, ok := s.Evidence[link.EvidenceID]
			if !ok || !oneOf(link.Role, "supports", "contradicts", "context") {
				return fmt.Errorf("%s: invalid evidence link", id)
			}
			if !compatible(a.Scope, e.Scope) {
				return fmt.Errorf("%s: incompatible evidence scope", id)
			}
		}
		if a.Predicate == "contains" {
			if a.Hierarchy == "" || a.To == "" || a.From == a.To {
				return fmt.Errorf("%s: invalid containment", id)
			}
			if a.Disposition != "accepted" || a.Applicability != "applicable" {
				continue
			}
			// Distinct scoped interpretations are independent forests. Queries also validate
			// their selected forest, preventing ambiguous unscoped combinations.
			b, _ := json.Marshal(a.Scope)
			key := a.Hierarchy + "\x00" + a.Interpretation + "\x00" + string(b)
			if forests[key] == nil {
				forests[key] = map[string]string{}
			}
			if parent, ok := forests[key][a.To]; ok && parent != a.From {
				return fmt.Errorf("%s: multiple containment parents", id)
			}
			forests[key][a.To] = a.From
		}
	}
	for _, parents := range forests {
		if err := checkForest(parents); err != nil {
			return err
		}
	}
	return nil
}
func checkForest(parents map[string]string) error {
	for id := range parents {
		seen := map[string]bool{}
		for n := id; n != ""; n = parents[n] {
			if seen[n] {
				return errors.New("containment cycle")
			}
			seen[n] = true
		}
	}
	return nil
}

// Builder stages explicit operations. IDs are workspace keys independent of containment.
type Builder struct{ Patch Patch }

func NewPatch(base uint64, actor, reason string) *Builder {
	return &Builder{Patch: Patch{ExpectedRevision: base, Actor: actor, Reason: reason}}
}
func (b *Builder) Entity(id, kind, label string) string {
	e := Entity{ID: id, Kind: kind, Label: label}
	b.Patch.Operations = append(b.Patch.Operations, Operation{Action: "create", Entity: &e})
	return id
}
func (b *Builder) Relate(id, from, predicate, to string) *Assertion {
	a := &Assertion{ID: id, From: from, To: to, Predicate: predicate, Basis: "declared", Disposition: "accepted", Applicability: "applicable"}
	b.Patch.Operations = append(b.Patch.Operations, Operation{Action: "create", Assertion: a})
	return a
}
func (b *Builder) Contains(id, hierarchy, parent, child string) *Assertion {
	a := b.Relate(id, parent, "contains", child)
	a.Hierarchy = hierarchy
	return a
}
