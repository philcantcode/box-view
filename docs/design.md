# Implementation decisions

The core has no storage or rendering dependencies. `Store` has three operations:
snapshot, apply, history. Memory and SQLite use the same `Resolve` validation,
query and projection code. HTML and Flutter display server projections and send
typed JSON operations back to the backend. The frontend owns no model facts.

The initial release uses materialized JSON snapshots for every revision plus
the resolved semantic commit journal in SQLite. Both are written in one immediate
transaction. This intentionally favors a small, inspectable implementation over
indexed current-state tables and periodic snapshots from the research proposal.
Historical lookup is indexed by revision; graph queries operate in memory.
Storage and query cost grow with model size and revision count. This is suitable
for explanatory models, not a claim of corpus-scale indexed performance.

Containment is a forest per hierarchy, interpretation and exact scope. Queries
also validate the combined selected forest, including unconstrained scope
matches. Competing accepted scoped models may coexist but an ambiguous combined
selection requires a narrower scope. Proposed containment does not silently
become layout structure. Accepted means admitted to the working model.

Caller-owned objects are copied before storage; snapshots and commits returned
by memory storage are detached. Stable keys are canonical identities. Renaming,
updating evidence, changing scope or moving containment never changes these IDs.
Repeated identical creation does not duplicate records, but still records an
attributed commit; this is deliberate rather than content-addressed deduplication.

Views use nested native HTML/Flutter layout with matching coloured endpoint dots
at the bottom-left of both connected boxes and a
separate accessible relationship list. Hover or focus reveals directed, labelled
curves; selecting a dot pins the connection and opens the existing inspector.
Overlays measure the endpoint dots after layout and update when layout changes.
Dots wrap in box footers; parallel arrows use separate curved lanes;
this is not obstacle-aware routing for dense graphs. Keyboard and screen-reader
navigation remain available through dot buttons and the relationship list.
Projection grouping retains exact assertion membership,
and paths traverse canonical directed endpoints. D2 source exports provide a
route to routed diagram rendering without making D2 a runtime dependency.

Optional presentation features from the proposal—ELK layout, ports, aliases in
multiple display locations, automatic semantic zoom, layout hints, artifact-store
resolution, per-record incremental invalidation, branching/rebasing, and automated
architecture discovery—are extension points, not part of the 0.1.0 protocol.
Undo is an ordinary explicit compensating update patch; history is never erased.
No automatic confidence or promotion is performed. View specs are ordinary JSON
that callers can persist alongside their investigation.

The server defaults to loopback, enforces token authentication outside loopback,
limits request bodies and validates cross-origin writes. A token is a workspace
credential, not a multi-user authorization system. Actor fields are attributed
producer claims, not authenticated identities. Deploy behind an authenticated
HTTPS proxy when user identity, access control or rate limits are needed.

The determinism tests cover rename/history, invalid endpoints, cycles, multiple
parents, incompatible evidence, conflict atomicity across connections, durability,
journal replay, bounded result reporting, aggregate members and collapsed false
paths. Queries state knowledge completeness as unknown.
