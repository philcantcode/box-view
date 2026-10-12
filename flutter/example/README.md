# Box View example

Start `go run ./cmd/box-view -demo -origin http://localhost:9000` from the repo root.
Then run from this directory:

```sh
flutter pub get
flutter run -d chrome --web-port 9000 --dart-define=BOX_VIEW_URL=http://127.0.0.1:8080
```

The example reads backend projections, explains selected relationship groups,
queries component boundaries and submits explicit revision-checked JSON patches.
