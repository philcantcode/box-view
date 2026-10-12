# Release validation — 2026-10-12

Validated locally on Windows amd64, with Linux Docker used for race tests.

| Check | Result |
|---|---|
| Go 1.26.9 `go test ./...` | Passed |
| `go vet ./...` | Passed |
| Linux Go 1.26.9 `go test -race ./...` | Passed |
| Go vulnerability scan of the Windows server binary | No vulnerabilities found |
| JavaScript adapter tests | 3 passed, including directional geometry |
| Chromium end-to-end tests | 3 passed, including relationship-dot interaction |
| Flutter 3.44.5 / Dart 3.12.2 analysis | No issues |
| Flutter package client and widget tests | 4 passed, including dot overlays and responsive spans |
| Flutter example widget test | Passed |
| Flutter example release web build | Passed |
| Flutter package publication dry run | 0 warnings |
| Docker production image build | Passed |
| Windows packaged server API smoke test | Passed |
| Windows/Linux/macOS amd64/arm64 release builds | Passed |
| Release archive SHA256 verification | Passed |

The browser test covers model rendering, collapse, aggregate evidence selection,
lens filtering, boundary query, patch editing, conflict reporting and historical
revision selection. Go tests additionally cover identity across rename/move,
invalid references, containment cycles/ambiguity, scoped evidence, exact aggregate
membership, false path prevention, atomic concurrent conflicts across SQLite
connections, reopen durability, journal replay and result truncation.

The Go toolchain was upgraded from the host's Go 1.25.4 to Go 1.26.9 after a binary
scan identified standard-library advisories. The final dependency set uses
modernc.org/sqlite 1.60.1 and golang.org/x/sys 0.48.0. The final binary scan is clean.

Flutter's web build emits an upstream Cupertino font-family notice; this example
uses Material icons and its widget tests pass. It also reports a successful Wasm
dry run. Native Android/iOS/macOS application shells are outside this Flutter
package's artifact: host applications embed the tested widget and client.
Cross-compiled Go targets other than Windows amd64 and Linux amd64 were built
successfully but were not executed on their respective hardware.

Local archives are under `dist/`, and the Flutter web application is under
`flutter/example/build/web/`. CI and tagged-release workflows are configured;
they have not been run on GitHub. No remote release or pub.dev package was published.
