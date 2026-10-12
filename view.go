package boxmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// LayoutConfig controls automatic sibling rows, including nested containers.
// Zero fields inherit the configured defaults. Spans clamp to available columns.
type LayoutConfig struct {
	MaxBoxesPerRow int `json:"maxBoxesPerRow,omitempty"`
	MinBoxWidth    int `json:"minBoxWidth,omitempty"`
}

func DefaultLayout() LayoutConfig { return LayoutConfig{MaxBoxesPerRow: 3, MinBoxWidth: 220} }
func (c LayoutConfig) Resolve(base LayoutConfig) (LayoutConfig, error) {
	if base.MaxBoxesPerRow == 0 {
		base.MaxBoxesPerRow = 3
	}
	if base.MinBoxWidth == 0 {
		base.MinBoxWidth = 220
	}
	if c.MaxBoxesPerRow == 0 {
		c.MaxBoxesPerRow = base.MaxBoxesPerRow
	}
	if c.MinBoxWidth == 0 {
		c.MinBoxWidth = base.MinBoxWidth
	}
	if c.MaxBoxesPerRow < 1 || c.MaxBoxesPerRow > 12 || c.MinBoxWidth < 120 || c.MinBoxWidth > 1200 {
		return c, errors.New("layout requires maxBoxesPerRow 1..12 and minBoxWidth 120..1200")
	}
	return c, nil
}

type BoxStyle struct {
	Colour string `json:"colour,omitempty"`
	Span   int    `json:"span,omitempty"`
}

var colourPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type View struct {
	Filter     Filter              `json:"filter"`
	Root       string              `json:"root,omitempty"`
	Collapsed  []string            `json:"collapsed,omitempty"`
	Budget     int                 `json:"budget,omitempty"`
	EdgeBudget int                 `json:"edgeBudget,omitempty"`
	Layout     LayoutConfig        `json:"layout"`
	Styles     map[string]BoxStyle `json:"styles,omitempty"`
}
type Box struct {
	Entity    Entity   `json:"entity"`
	Parent    string   `json:"parent,omitempty"`
	Collapsed bool     `json:"collapsed"`
	Style     BoxStyle `json:"style"`
}
type Edge struct {
	ID        string   `json:"id"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Predicate string   `json:"predicate"`
	Members   []string `json:"members"`
}
type Projection struct {
	Revision          uint64 `json:"revision"`
	View              View   `json:"view"`
	Boxes             []Box  `json:"boxes"`
	Edges             []Edge `json:"edges"`
	OmittedEntities   int    `json:"omittedEntities"`
	OmittedAssertions int    `json:"omittedAssertions"`
}

// Project owns semantic collapse and exact aggregate membership. Layout is an adapter concern.
func (s Snapshot) Project(v View) (Projection, error) {
	return s.ProjectWithLayout(v, DefaultLayout())
}

// ProjectWithLayout supplies application-wide defaults; each view can override them.
func (s Snapshot) ProjectWithLayout(v View, defaults LayoutConfig) (Projection, error) {
	p := Projection{Revision: s.Revision, View: v, Boxes: []Box{}, Edges: []Edge{}}
	var err error
	v.Layout, err = v.Layout.Resolve(defaults)
	if err != nil {
		return p, err
	}
	for id, style := range v.Styles {
		if _, ok := s.Entities[id]; !ok {
			return p, ErrNotFound
		}
		if style.Span < 0 || style.Span > 12 {
			return p, fmt.Errorf("%s: span must be 1..12 (or zero for default)", id)
		}
		if style.Colour != "" && !colourPattern.MatchString(style.Colour) {
			return p, fmt.Errorf("%s: colour must be #RRGGBB", id)
		}
	}
	if v.Budget == 0 {
		v.Budget = 200
	}
	if v.EdgeBudget == 0 {
		v.EdgeBudget = 1000
	}
	if v.Budget < 1 || v.Budget > 10000 || v.EdgeBudget < 1 || v.EdgeBudget > 10000 {
		return p, errors.New("view budgets must be 1..10000")
	}
	p.View = v
	parents, err := s.parents(v.Filter)
	if err != nil {
		return p, err
	}
	region := map[string]bool{}
	if v.Root != "" {
		if _, ok := s.Entities[v.Root]; !ok {
			return p, ErrNotFound
		}
		region = descendants(v.Root, parents)
	} else {
		for id := range s.Entities {
			region[id] = true
		}
	}
	collapsed := map[string]bool{}
	for _, id := range v.Collapsed {
		if _, ok := s.Entities[id]; !ok {
			return p, ErrNotFound
		}
		collapsed[id] = true
	}
	representative := func(id string) string {
		r := id
		for n := parents[id]; n != ""; n = parents[n] {
			if region[n] && collapsed[n] {
				r = n
			}
		}
		return r
	}
	candidates := map[string]bool{}
	for id := range region {
		candidates[representative(id)] = true
	}
	// Deterministic ancestor-first traversal keeps nested boxes valid at every budget.
	children := map[string][]string{}
	roots := []string{}
	for _, id := range keys(candidates) {
		parent := parents[id]
		if !candidates[parent] {
			roots = append(roots, id)
		} else {
			children[parent] = append(children[parent], id)
		}
	}
	visible := map[string]bool{}
	var add func(string)
	add = func(id string) {
		if len(p.Boxes) >= v.Budget {
			return
		}
		parent := parents[id]
		if !visible[parent] {
			parent = ""
		}
		visible[id] = true
		style := v.Styles[id]
		if style.Span == 0 {
			style.Span = 1
		}
		p.Boxes = append(p.Boxes, Box{Entity: s.Entities[id], Parent: parent, Collapsed: collapsed[id], Style: style})
		for _, child := range children[id] {
			add(child)
		}
	}
	for _, id := range roots {
		add(id)
	}
	for id := range region {
		if !visible[id] {
			p.OmittedEntities++
		}
	}
	groups := map[string]*Edge{}
	for _, id := range keys(s.Assertions) {
		a := s.Assertions[id]
		if a.Predicate == "contains" || a.To == "" || !v.Filter.matches(a) {
			continue
		}
		if !region[a.From] || !region[a.To] {
			continue
		}
		from, to := representative(a.From), representative(a.To)
		if !visible[from] || !visible[to] {
			p.OmittedAssertions++
			continue
		}
		if from == to && a.From != a.To {
			p.OmittedAssertions++
			continue
		}
		b, _ := json.Marshal([]string{from, to, a.Predicate})
		key := string(b)
		if groups[key] == nil {
			if len(groups) >= v.EdgeBudget {
				p.OmittedAssertions++
				continue
			}
			hash := sha256.Sum256(b)
			groups[key] = &Edge{ID: "aggregate-" + hex.EncodeToString(hash[:8]), From: from, To: to, Predicate: a.Predicate, Members: []string{}}
		}
		groups[key].Members = append(groups[key].Members, id)
	}
	for _, key := range keys(groups) {
		p.Edges = append(p.Edges, *groups[key])
	}
	return p, nil
}
