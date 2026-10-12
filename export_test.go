package boxmodel_test

import (
	"bytes"
	"strings"
	"testing"

	model "github.com/philcantcode/box-view"
)

func TestD2ExportsQualifiedEndpointsAndEscapesLabels(t *testing.T) {
	s := snap(t, seeded(t))
	e := s.Entities["db"]
	e.Label = "Quote \" and newline\n"
	s.Entities[e.ID] = e
	p, _ := s.Project(model.View{})
	var out bytes.Buffer
	if err := model.WriteD2(&out, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `Quote \" and newline\n`) || !strings.Contains(out.String(), " -> ") || !strings.Contains(out.String(), "writes · 1 assertions") {
		t.Fatal(out.String())
	}
}
