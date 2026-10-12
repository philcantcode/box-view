# Release 0.1.0

The repository prepares a Go module, bundled server binaries, and a separate
Flutter package. Publication is a maintainer action; no tag or remote release is
created by local build commands.

## Required checks

1. Run `go test ./...`, `go vet ./...`, and `go test -race ./...` on a host with
   a C compiler. Runtime builds use pure Go; Go's race detector requires a C toolchain.
   Use Go 1.26.9 or later. Run `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`
   against the source and use `-mode binary <binary>` to verify the packaged toolchain.
2. Run `node --test web/adapter.test.js`. In `web`, use `npm ci`,
   `npx playwright install chromium`, and `npx playwright test` for browser checks.
3. In `flutter`, run `flutter pub get`, `flutter analyze`, `flutter test`,
   and `flutter pub publish --dry-run`.
4. In `flutter/example`, run `flutter test` and `flutter build web`.
5. Run `./scripts/release.ps1` from PowerShell to build release archives and SHA256
   checksums under `dist/`. The script tests Go and creates Windows, Linux and
   macOS amd64/arm64 binaries with the embedded HTML viewer. It also archives
   Flutter sources. Flutter example web output is built separately.

GitHub CI repeats Go, browser and Flutter checks. The tag release workflow runs
only after successful tests, builds archives and attaches them to the GitHub
release. Review the changelog and license before tagging. For Go module release,
push an annotated `v0.1.0` tag after merging the tested source. Flutter packages
are published from the `flutter` directory with `flutter pub publish`; publication
requires a verified pub.dev account and an available package name. If `box_view`
is unavailable, rename the Dart package and its imports before publication.

## Operating and upgrading

- Use HTTPS and a workspace bearer token for remote access. Allow only the actual
  frontend origin. Tokens entered in the HTML viewer stay in memory.
- Stop the server before copying the SQLite database for backup. Keep the complete
  database and verify a restore by reading its latest and historical snapshots.
- The 0.1.0 database has two tables: commits and snapshots. It has no destructive
  migration, external artifact fetching or automatic retention policy.
- Breaking protocol changes require an explicit future version. This release
  uses `/api` and rejects unknown patch fields to avoid silently accepting typos.
- Do not interpret an empty query as proof of absence; knowledge completeness
  is unknown. Review [design.md](design.md) for scope and storage limits.

Generated binaries, databases, build directories, and dependency caches are
ignored. Go dependencies and Flutter example dependencies are locked; browser
dev dependencies use `web/package-lock.json`.
