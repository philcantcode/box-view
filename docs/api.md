# HTTP protocol v1

Both viewers draw matching coloured dots in the bottom-left of both boxes for
each projected `edge`, revealing directed arrows
on box/dot hover or focus. Dot selection pins an arrow and invokes the same
revision-pinned explanation as list selection. Aggregate dots show `members.length`;
no new assertion or API operation is created by drawing.

The Go core is authoritative. All API bodies and responses use JSON, except
`GET /api/watch` which returns Server-Sent Events. Optional bearer authentication
uses `Authorization: Bearer <BOX_VIEW_TOKEN>`. Browser clients may use the same
origin or the server's single `-origin` value. Requests are limited to 4 MiB.
Unknown JSON fields, extra JSON values and invalid records are rejected.

| Method | Endpoint | Input | Output |
|---|---|---|---|
| GET | `/api/snapshot` | optional `revision` query parameter | Complete canonical snapshot |
| POST | `/api/apply` | Patch | Commit |
| POST | `/api/query` | Query; optional `revision` parameter | Bounded query result |
| POST | `/api/view` | View; optional `revision` parameter | Projected boxes and edge groups |
| POST | `/api/explain` | `{"ids":["assertion-id"]}`; optional `revision` | Assertions and evidence |
| GET | `/api/diff?from=0&to=1` | Required revision pair | Semantic record changes |
| GET | `/api/history` | `after` (default 0), `limit` (1–1000, default 200), optional `id` | Commit page |
| GET | `/api/watch` | `after` or `Last-Event-ID` | Persisted commits after cursor |

Omit `revision` for current state. Revision zero is empty. Read responses include
their revision; pass it to subsequent query/explain requests for consistent
inspection. History is paginated by commit revision; optional ID filtering happens
within that page, so advance the cursor by the unfiltered page when searching history.
Watch emits `event: commit`, revision as SSE ID, JSON commit data, and heartbeat
comments. Reconnect with the last received ID; notifications never precede commit.
Use fetch streaming when bearer authentication is needed (native EventSource
does not support custom authorization headers).

## Patch

```json
{
  "expectedRevision": 0,
  "actor": "human:alice",
  "reason": "Record storage",
  "operations": [
    {"action":"create","entity":{"id":"sync","label":"Sync","kind":"component"}},
    {"action":"create","entity":{"id":"db","label":"Messages","kind":"datastore"}},
    {"action":"create","assertion":{"id":"writer","from":"sync","to":"db","predicate":"writes","basis":"inferred","disposition":"proposed","applicability":"unresolved","scope":{"target":"build-A"}}}
  ]
}
```

Operations are `create` or `update`, with exactly one `entity`, `assertion`, or
`evidence`. All references are validated after staging the whole patch, allowing
forward references. Entity fields are `id`, `label`, `kind`, optional `aliases`
and `tags`. The three record types share one identity namespace.

Entities also have optional `description` text, displayed in both adapters and
the inspector. It is ordinary text, never interpreted as HTML or Markdown.

Assertions have `id`, `from`, `to` (or `value` for a property), `predicate`,
`basis` (`declared`, `static`, `observed`, `inferred`), `disposition` (`proposed`,
`accepted`, `rejected`, `superseded`), `applicability` (`applicable`, `stale`,
`unresolved`), `scope`, optional `description` and `evidence` links. Containment
also requires `hierarchy`; `interpretation` selects alternative forests.

Predicates: `contains` (parent to child), `calls`, `reads`, `writes`, `publishes`,
`subscribes`, `implements`, `depends_on`. Extensions must be namespaced, e.g.
`acme:allocates`; producers should document their definition. Evidence does not
automatically change disposition or applicability.

Evidence fields: `id`, `uri`, optional `digest`, `selector`, `producer`,
`capturedAt`, and `scope`. Links: `evidenceId`, `role` (`supports`, `contradicts`,
`context`), optional `note`. URIs identify external artifacts; the server never
fetches or executes them. Digests and selectors are metadata, not verified contents.
Scope fields are `target`, `run`, `configuration`. Empty fields are unconstrained;
nonempty conflicting fields are incompatible. Evidence must be compatible with
its assertion. To investigate a new build, explicitly change applicability or
record a new scoped assertion; no automatic evidence reclassification occurs.

## Queries and views

```json
{"kind":"boundary","entityId":"storage","filter":{"hierarchy":"architecture","scope":{"target":"build-A"},"predicates":["writes"],"dispositions":["accepted"],"applicabilities":["applicable"]},"limit":200}
```

Kinds: `children`, `descendants`, `boundary`, `neighbours`, `assertions`,
`unresolved`, `paths`. `paths` also requires `to`; paths are directed lists of
canonical assertion IDs excluding containment. Default limit 200 (max 10000),
depth 8 (max 64), with a 100000 traversal budget. `truncated` conservatively
reports depth, traversal or result limits. Boundary includes descendant endpoints;
neighbours includes only direct endpoints. Unresolved includes proposals, stale
or unresolved applicability, and contradictions.

Filter lists omitted or empty mean all values, so request accepted/applicable
explicitly for a working view. Containment always uses accepted/applicable
assertions in `hierarchy` (default `architecture`) and `interpretation` (default
empty), independent of the relationship lens. Ambiguous selected forests fail
instead of choosing a parent.

```json
{"filter":{"hierarchy":"architecture","dispositions":["accepted"],"applicabilities":["applicable"]},"collapsed":["storage"],"budget":200,"edgeBudget":1000}
```

View fields also include optional `root`. Descendants of root are selected.

`layout` configures automatic sibling rows, including nested containers:
`maxBoxesPerRow` defaults to 3 (range 1–12) and `minBoxWidth` defaults to 220 pixels
(range 120–1200). Zero or omitted fields inherit application defaults. The server
flags `-max-boxes-per-row` and `-min-box-width` set those defaults; a view can
override either field. The resolved layout is returned in `projection.view.layout`.
Narrow containers automatically use fewer columns; a sole box fills its row.

`styles` maps entity IDs to optional `colour` (`#RRGGBB`, a border accent) and
`span` (1–12; zero/omitted defaults to 1). Span counts row columns and clamps to
available columns on small screens. Unknown entity IDs, invalid colours and
out-of-range spans fail validation. These are presentation settings, not semantic
entity changes, and can differ between views. For example:

```json
{
  "layout":{"maxBoxesPerRow":3,"minBoxWidth":220},
  "styles":{
    "sync":{"colour":"#b78aff","span":1},
    "storage":{"colour":"#4f83cc","span":2}
  }
}
```

In the HTML app, use the **Max boxes per row** input or **Box appearance** JSON
editor. Embed Flutter with `BoxDiagram(layout: LayoutConfig(...))` to optionally
override a server-projected layout at the host application's level.
Collapsed descendants map to their visible ancestor; similar edges group by
visible endpoints and predicate with exact sorted `members`. Internal hidden
relationships are omitted, not converted into new assertions. Boxes retain
entity IDs and `parent`. Budgets are 1–10000. Omission counts include collapsed
children/internal edges and budget omissions; root-excluded records are outside
the selection. Projection uses deterministic ancestor-first ordering.

Errors use `{"error":"message"}`: 400 invalid input, 401 authentication, 403
disallowed mutation origin, 404 absent record/revision, 409 outdated expected
revision. Refresh and explicitly reconcile your patch on 409; never blind retry.
