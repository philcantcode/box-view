package boxmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// WriteD2 exports a projected view. Edge labels count assertion records, not observations.
// Canonical IDs and member sets remain in the Projection used to produce this drawing.
func WriteD2(w io.Writer, p Projection) error {
	children := map[string][]Box{}
	ids := map[string]string{}
	paths := map[string]string{}
	for _, b := range p.Boxes {
		hash := sha256.Sum256([]byte(b.Entity.ID))
		ids[b.Entity.ID] = "n" + hex.EncodeToString(hash[:8])
		children[b.Parent] = append(children[b.Parent], b)
	}
	var out strings.Builder
	var draw func(string, string, int)
	draw = func(parent, path string, depth int) {
		for _, b := range children[parent] {
			id := ids[b.Entity.ID]
			full := id
			if path != "" {
				full = path + "." + id
			}
			paths[b.Entity.ID] = full
			indent := strings.Repeat("  ", depth)
			fmt.Fprintf(&out, "%s%s: %s {\n", indent, id, strconv.Quote(b.Entity.Label))
			draw(b.Entity.ID, full, depth+1)
			fmt.Fprintf(&out, "%s}\n", indent)
		}
	}
	draw("", "", 0)
	for _, e := range p.Edges {
		fmt.Fprintf(&out, "%s -> %s: %s\n", paths[e.From], paths[e.To], strconv.Quote(fmt.Sprintf("%s · %d assertions", e.Predicate, len(e.Members))))
	}
	_, err := io.WriteString(w, out.String())
	return err
}
