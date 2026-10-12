package api_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	model "github.com/philcantcode/box-view"
	"github.com/philcantcode/box-view/api"
	"github.com/philcantcode/box-view/example"
)

func TestAPIWorkflow(t *testing.T) {
	store := model.NewMemory()
	handler := api.Handler(store, api.Options{Token: "secret", AllowedOrigin: "http://localhost:9000"})
	call := func(method, path string, body any, token string) *httptest.ResponseRecorder {
		t.Helper()
		var b bytes.Buffer
		if body != nil {
			json.NewEncoder(&b).Encode(body)
		}
		r := httptest.NewRequest(method, path, &b)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/snapshot", nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/apply", example.Patch(0), "secret"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := call("POST", "/api/apply", example.Patch(0), "secret"); w.Code != 409 {
		t.Fatal(w.Code)
	}
	w := call("POST", "/api/view", model.View{Collapsed: []string{"storage"}}, "secret")
	var p model.Projection
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || p.Revision != 1 || len(p.Boxes) != 3 {
		t.Fatal(w.Body.String(), err)
	}
	w = call("POST", "/api/explain?revision=1", map[string]any{"ids": []string{"sync-writes"}}, "secret")
	var e model.Explanation
	json.Unmarshal(w.Body.Bytes(), &e)
	if len(e.Assertions) != 1 || len(e.Evidence) != 1 {
		t.Fatal(w.Body.String())
	}
	w = call("GET", "/api/diff?from=0&to=1", nil, "secret")
	var d model.Diff
	json.Unmarshal(w.Body.Bytes(), &d)
	if len(d.Changes) != 12 {
		t.Fatal(w.Body.String())
	}
	if w = call("GET", "/api/snapshot?revision=900", nil, "secret"); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/query", map[string]any{"kind": "paths", "entityId": "sync", "to": "db", "unknown": true}, "secret"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	r := httptest.NewRequest("POST", "/api/apply", strings.NewReader(`{} {}`))
	r.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	r = httptest.NewRequest("POST", "/api/apply", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://untrusted.invalid")
	r.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	r = httptest.NewRequest("OPTIONS", "/api/view", nil)
	r.Header.Set("Origin", "http://localhost:9000")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:9000" {
		t.Fatal(w.Header())
	}
}
func TestWatchOnlyCommittedChangesAndResume(t *testing.T) {
	store := model.NewMemory()
	store.Apply(context.Background(), example.Patch(0))
	server := httptest.NewServer(api.Handler(store, api.Options{}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/watch", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(res.Body)
	found := false
	for scanner.Scan() {
		if scanner.Text() == "id: 1" {
			found = true
			break
		}
	}
	res.Body.Close()
	if !found {
		t.Fatal("missing persisted event")
	}
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/api/watch", nil)
	req.Header.Set("Last-Event-ID", "1")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	scanner = bufio.NewScanner(res.Body)
	if !scanner.Scan() || scanner.Text() != ": heartbeat" {
		t.Fatal("resume replayed old event")
	}
	res.Body.Close()
}
