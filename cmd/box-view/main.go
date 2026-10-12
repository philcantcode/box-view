package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	model "github.com/philcantcode/box-view"
	"github.com/philcantcode/box-view/api"
	"github.com/philcantcode/box-view/example"
	"github.com/philcantcode/box-view/sqlite"
	"github.com/philcantcode/box-view/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	version := flag.Bool("version", false, "print version and exit")
	maxPerRow := flag.Int("max-boxes-per-row", 3, "default maximum boxes per row (1..12)")
	minWidth := flag.Int("min-box-width", 220, "default minimum box width in pixels (120..1200)")
	address := flag.String("addr", "127.0.0.1:8080", "listen address")
	database := flag.String("db", "box-view.db", "SQLite database path")
	demo := flag.Bool("demo", false, "seed an empty database with the example")
	origin := flag.String("origin", "", "allowed frontend origin")
	flag.Parse()
	if *version {
		fmt.Println(model.Version)
		return nil
	}
	layout, err := (model.LayoutConfig{MaxBoxesPerRow: *maxPerRow, MinBoxWidth: *minWidth}).Resolve(model.DefaultLayout())
	if err != nil {
		return err
	}
	token := os.Getenv("BOX_VIEW_TOKEN")
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if token == "" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("set BOX_VIEW_TOKEN to listen on a non-loopback address")
	}
	store, err := sqlite.Open(*database)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *demo {
		s, err := store.Snapshot(ctx, nil)
		if err != nil {
			return err
		}
		if s.Revision == 0 {
			if _, err = store.Apply(ctx, example.Patch(0)); err != nil {
				return err
			}
		}
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(store, api.Options{Token: token, AllowedOrigin: *origin, Layout: layout}))
	mux.Handle("/", web.Handler())
	server := &http.Server{Addr: *address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("box-view listening on http://%s", *address)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
