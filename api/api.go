// Package api exposes the same Go model operations to browser, Flutter and other clients.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	model "github.com/philcantcode/box-view"
)

type Options struct {
	Token         string
	AllowedOrigin string
	Layout        model.LayoutConfig
}

func Handler(store model.Store, options Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			fail(w, err)
			return
		}
		s, err := store.Snapshot(r.Context(), rev)
		respond(w, s, err)
	})
	mux.HandleFunc("POST /api/apply", func(w http.ResponseWriter, r *http.Request) {
		var p model.Patch
		if err := decode(w, r, &p); err != nil {
			fail(w, err)
			return
		}
		c, err := store.Apply(r.Context(), p)
		respond(w, c, err)
	})
	mux.HandleFunc("POST /api/query", func(w http.ResponseWriter, r *http.Request) {
		var q model.Query
		if err := decode(w, r, &q); err != nil {
			fail(w, err)
			return
		}
		s, err := snapshot(store, r)
		if err != nil {
			fail(w, err)
			return
		}
		result, err := s.Query(q)
		respond(w, result, err)
	})
	mux.HandleFunc("POST /api/view", func(w http.ResponseWriter, r *http.Request) {
		var v model.View
		if err := decode(w, r, &v); err != nil {
			fail(w, err)
			return
		}
		s, err := snapshot(store, r)
		if err != nil {
			fail(w, err)
			return
		}
		p, err := s.ProjectWithLayout(v, options.Layout)
		respond(w, p, err)
	})
	mux.HandleFunc("POST /api/explain", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			IDs []string `json:"ids"`
		}
		if err := decode(w, r, &input); err != nil {
			fail(w, err)
			return
		}
		if len(input.IDs) > 10000 {
			fail(w, errors.New("too many IDs"))
			return
		}
		s, err := snapshot(store, r)
		if err != nil {
			fail(w, err)
			return
		}
		e, err := s.Explain(input.IDs)
		respond(w, e, err)
	})
	mux.HandleFunc("GET /api/diff", func(w http.ResponseWriter, r *http.Request) {
		from, err := parseUint(r.URL.Query().Get("from"))
		if err != nil {
			fail(w, err)
			return
		}
		to, err := parseUint(r.URL.Query().Get("to"))
		if err != nil {
			fail(w, err)
			return
		}
		a, err := store.Snapshot(r.Context(), &from)
		if err != nil {
			fail(w, err)
			return
		}
		b, err := store.Snapshot(r.Context(), &to)
		if err != nil {
			fail(w, err)
			return
		}
		respond(w, model.Compare(a, b), nil)
	})
	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		after, limit, err := historyArgs(r)
		if err != nil {
			fail(w, err)
			return
		}
		cs, err := store.History(r.Context(), after, limit)
		id := r.URL.Query().Get("id")
		if id != "" {
			filtered := []model.Commit{}
			for _, c := range cs {
				for _, affected := range c.Affected {
					if affected == id {
						filtered = append(filtered, c)
						break
					}
				}
			}
			cs = filtered
		}
		respond(w, cs, err)
	})
	mux.HandleFunc("GET /api/watch", func(w http.ResponseWriter, r *http.Request) {
		after, _, err := historyArgs(r)
		if err != nil {
			fail(w, err)
			return
		}
		if last := r.Header.Get("Last-Event-ID"); last != "" {
			after, err = parseUint(last)
			if err != nil {
				fail(w, err)
				return
			}
		}
		f, ok := w.(http.Flusher)
		if !ok {
			fail(w, errors.New("streaming unavailable"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			cs, err := store.History(r.Context(), after, 100)
			if err != nil {
				return
			}
			for _, c := range cs {
				b, _ := json.Marshal(c)
				if _, err = fmt.Fprintf(w, "id: %d\nevent: commit\ndata: %s\n\n", c.Revision, b); err != nil {
					return
				}
				after = c.Revision
			}
			if _, err = fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			f.Flush()
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
			}
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if origin := r.Header.Get("Origin"); origin != "" && origin == options.AllowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if options.Token != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+options.Token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Reject cross-origin mutations even when running locally without a token.
		if r.Method == "POST" {
			origin := r.Header.Get("Origin")
			if origin != "" && origin != options.AllowedOrigin && origin != "http://"+r.Host && origin != "https://"+r.Host {
				http.Error(w, "origin denied", http.StatusForbidden)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func snapshot(store model.Store, r *http.Request) (model.Snapshot, error) {
	rev, err := revision(r)
	if err != nil {
		return model.Snapshot{}, err
	}
	return store.Snapshot(r.Context(), rev)
}
func revision(r *http.Request) (*uint64, error) {
	text := r.URL.Query().Get("revision")
	if text == "" {
		return nil, nil
	}
	v, err := parseUint(text)
	return &v, err
}
func parseUint(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, errors.New("invalid revision")
	}
	return v, nil
}
func historyArgs(r *http.Request) (uint64, int, error) {
	after := uint64(0)
	var err error
	if v := r.URL.Query().Get("after"); v != "" {
		after, err = parseUint(v)
		if err != nil {
			return 0, 0, err
		}
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit < 1 || limit > 1000 {
			return 0, 0, errors.New("history limit must be 1..1000")
		}
	}
	return after, limit, nil
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}
func respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, model.ErrConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, model.ErrNotFound) {
		status = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": strings.TrimSpace(err.Error())})
}
