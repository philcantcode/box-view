package web_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	model "github.com/philcantcode/box-view"
	"github.com/philcantcode/box-view/web"
)

func TestOfflineExportEscapesData(t *testing.T) {
	m := model.NewMemory()
	b := model.NewPatch(0, "test", "escaping")
	b.Entity("weird", "component", "</script><script>alert(1)</script>")
	m.Apply(context.Background(), b.Patch)
	s, _ := m.Snapshot(context.Background(), nil)
	p, _ := s.Project(model.View{})
	var out bytes.Buffer
	if err := web.ExportHTML(&out, p); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>alert(1)") {
		t.Fatal("script injection")
	}
	if !strings.Contains(out.String(), "\\u003c/script\\u003e") {
		t.Fatal("missing escaped label")
	}
}
