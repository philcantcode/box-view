# Box View

A small Go library for evidence-backed software models, with an optional SQLite
server and HTML and Flutter adapters. Entities keep the same identity through
renames and moves. Every model change is an attributed atomic commit. Views are
projections, so collapsing a box never alters the model.

## Run

Requires Go 1.26.9 or later. No C compiler or external database server required.

```sh
go run ./cmd/box-view -demo
```

Open http://127.0.0.1:8080 for the HTML viewer. Data persists in `box-view.db`.
Use `-db path/to/model.db`, `-addr 127.0.0.1:8080`, and `-origin http://localhost:9000`
to configure the server. Set `BOX_VIEW_TOKEN` for bearer authentication; it is
required when listening outside loopback. Serve behind HTTPS for remote access.
The token covers all API reads and writes; the static viewer is public.

Container deployment is available via `docker build -t box-view .`; mount a
writable `/data` directory for UID 65532, supply `BOX_VIEW_TOKEN`, and publish
port 8080. The container has no shell and runs as an unprivileged user.

## Use as a Go library

```go
package main

import (
    "context"
    "log"
    model "github.com/philcantcode/box-view"
)

func main() {
    m := model.NewMemory() // or sqlite.Open("model.db")
    p := model.NewPatch(0, "mapper", "Describe storage")
    app := p.Entity("app", "system", "Application")
    db := p.Entity("db", "datastore", "Messages")
    p.Contains("app-db", "architecture", app, db)
    p.Relate("app-writes", app, "writes", db)
    if _, err := m.Apply(context.Background(), p.Patch); err != nil { log.Fatal(err) }
    s, _ := m.Snapshot(context.Background(), nil)
    result, _ := s.Query(model.Query{Kind: "boundary", EntityID: db})
    log.Println(result.AssertionIDs)
}
```

IDs are caller-assigned workspace keys, never containment paths. Repeating a
`create` with identical content is safe; changing an existing record requires an
explicit `update`. Updates replace the complete record. Each patch must name the
expected current revision, actor and reason. Concurrent outdated patches receive
a conflict. Supersede or reject assertions rather than deleting history.

## Frontend adapters

Matching relationship dots appear at the bottom-left of both connected boxes.
Multiple connections form a small wrapping row of dots. Hover/focus a box or dot
to reveal coloured arrows and verb/count labels; click/tap a dot to pin its arrow
and inspect exact assertion members and evidence. Select again or press Escape
to unpin. Colours consistently identify relationship predicates. Collapsed
views retain aggregate counts, and the relationship list remains accessible.

Boxes support description text, per-view colour accents, and column spans.
Automatic layout defaults to at most three boxes per row and 220px minimum box
width, adapting to available space. Set server defaults with `-max-boxes-per-row`
and `-min-box-width`, or override a view's `layout` and `styles` fields. Both
adapters honour the same projected settings; see [docs/api.md](docs/api.md).

HTML uses a dependency-free `<box-view>` custom element and `BoxClient` from
`web/box-view.js`. Set `element.projection` to an `/api/view` result and handle
`toggle-box`, `select-entity` and `select-edge` events. Edge events contain exact
assertion IDs for a revision-pinned explanation. `web.ExportHTML` produces a
standalone static HTML view; `boxmodel.WriteD2` exports D2 source.

Flutter provides `BoxClient`, typed projection records, and the `BoxDiagram`
widget. The widget works on native Flutter and Flutter web. Run the example:

```sh
cd flutter/example
flutter pub get
flutter run -d chrome --web-port 9000 --dart-define=BOX_VIEW_URL=http://127.0.0.1:8080
```

Start the Go server with `-origin http://localhost:9000`. For native applications,
use a backend address reachable by the device and configure its platform's
network permissions. Read [flutter/README.md](flutter/README.md) for embedding.

## API and design

See [docs/api.md](docs/api.md) for the protocol, [docs/design.md](docs/design.md)
for invariants and implementation tradeoffs, and [docs/research.md](docs/research.md)
for the original proposal. JSON snapshots and patches are portable exports.
Queries report the selected revision, filters, evidence and truncation;
knowledge completeness is always `unknown` rather than an inferred negative claim.

## Verify and release

```sh
go test ./...
go vet ./...
go build ./cmd/box-view
node --test web/adapter.test.js
cd flutter
flutter pub get
flutter analyze
flutter test
cd example
flutter test
flutter build web
```

[docs/release.md](docs/release.md) covers release archives, CI and publication.
See [docs/validation.md](docs/validation.md) for executed checks and platform limits.
Licensed under MIT.
