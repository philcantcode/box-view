# Box View for Flutter

An API client and accessible nested-box widget for the Go Box View server.
The Go backend owns data, revisions, queries and view projection. This package
renders the same projected structure used by the HTML adapter.

```dart
final client = BoxClient('https://your-server', token: 'workspace-token');
final projection = await client.view({
  'filter': {
    'hierarchy': 'architecture',
    'dispositions': ['accepted'],
    'applicabilities': ['applicable'],
  },
  'collapsed': ['storage'],
});

BoxDiagram(
  projection: projection,
  onToggle: (id) { /* update collapsed IDs and request another projection */ },
  onSelectEdge: (edge) async {
    final explanation = await client.explain(edge.members,
        revision: projection.revision);
    // Show assertion records and their evidence in your inspector.
  },
);
```

Place the widget inside a scrollable parent. `BoxClient` also exposes `snapshot`,
`query`, and `apply`, with the server's JSON protocol. Call `close()` when finished.
`BoxApiException.status == 409` means reconcile against the latest revision.
Updates replace full records, and patches require an actor, reason and expected
revision. Server operations never run in the rendering widget.

Run `example` against the Go server using `--dart-define=BOX_VIEW_URL=<url>`.
For Flutter web, configure the server's `-origin` to match the frontend origin.
For native Flutter, provide the host application's internet permission and any
development cleartext-network configuration; use HTTPS in deployed applications.
`BOX_VIEW_TOKEN` is supported as a compile-time example setting; production apps
should supply authenticated credentials at runtime rather than embedding secrets.

See the repository's `docs/api.md` for scopes, evidence and query limits.

Matching coloured relationship dots appear at the bottom-left of both linked
boxes, with a wrapping row for multiple connections. Hover or focus a box/dot
to reveal directed arrows and verb/count labels. Tap/click a dot to pin the arrow
and invoke `onSelectEdge` with exact aggregate members; select again or press
Escape to unpin. Dots and arrows reposition when layout changes. A separate
relationship list remains available for accessible navigation. Colours identify
predicates (e.g. red for calls, amber for writes, blue for reads).

Entity descriptions are plain text. A view may include
`'styles': {'storage': {'colour': '#4f83cc', 'span': 2}}` and
`'layout': {'maxBoxesPerRow': 3, 'minBoxWidth': 220}`. The server supplies defaults
for omitted layout fields. Nested sibling rows reduce their column count to fit
the available width, and spans clamp to that count. For a local widget override,
pass `layout: const LayoutConfig(maxBoxesPerRow: 2)` to `BoxDiagram`.
