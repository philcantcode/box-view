// Package example provides a small model suitable for both viewers.
package example

import model "github.com/philcantcode/box-view"

func Patch(base uint64) model.Patch {
	b := model.NewPatch(base, "example", "Record the application's storage boundary")
	app := b.Entity("app", "system", "Application")
	sync := b.Entity("sync", "component", "Sync")
	storage := b.Entity("storage", "component", "Storage")
	db := b.Entity("db", "datastore", "Message database")
	api := b.Entity("storage-api", "component", "Storage API")
	b.Contains("app-sync", "architecture", app, sync)
	b.Contains("app-storage", "architecture", app, storage)
	b.Contains("storage-db", "architecture", storage, db)
	b.Contains("storage-api-parent", "architecture", storage, api)
	a := b.Relate("sync-writes", "sync", "writes", "db")
	a.Basis = "observed"
	a.Scope = model.Scope{Target: "demo-v1", Run: "session-1"}
	a.Evidence = []model.EvidenceLink{{EvidenceID: "trace", Role: "supports", Note: "Events 84–91 record a write"}}
	b.Relate("sync-calls", "sync", "calls", "storage-api")
	e := model.Evidence{ID: "trace", URI: "example://trace/session-1", Selector: "events:84-91", Producer: "demo", Scope: a.Scope}
	b.Patch.Operations = append(b.Patch.Operations, model.Operation{Action: "create", Evidence: &e})
	return b.Patch
}
