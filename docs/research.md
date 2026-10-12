Yes. I would design it as **a versioned, evidence-backed model of software, with nested box diagrams as one way of viewing it**.

That distinction matters: the boxes should not be the source of truth. The source of truth should describe the components, what you believe about them, their relationships, and why you believe those things. The library then produces a diagram, a compact textual description, or a query result from the same model.

## Start with the smallest useful comparison

Before building a substantial system, compare two approaches on a small example: perhaps twelve components, twenty relationships, and three revisions.

| Question | Simpler approach: diagrams plus notes in Git | Proposed library |
|---|---|---|
| What communicates with anything inside Storage? | Inspect the diagram and its children. | Query the boundary of Storage and return the underlying relationships. |
| Why do we believe Sync writes to this database? | Find the relevant note or commit. | Select the relationship and retrieve its supporting evidence. |
| What changed in our understanding since revision 12? | Read textual diffs. | See added, withdrawn, reclassified, and newly supported claims. |

Drawing boxes does not justify a new library by itself. D2 already offers programmatic diagram construction in Go, and Structurizr already generates multiple architecture diagrams from a single model. The value worth testing is **incremental, queryable understanding with explainable relationships and history**. [D2 Documentation](https://d2lang.com/tour/api/)

My recommendation is to build that semantic layer and reuse existing rendering and layout tools.

---

## 1. Separate the model, the view, and the drawing

I would give these three layers distinct responsibilities:

| Layer | Owns | Does not own |
|---|---|---|
| **Model** | Entities, relationships, claims, evidence, revisions. | Coordinates, colours, zoom level. |
| **View** | Which part of the model to show, grouping, filters, collapsed components, aggregation rules. | New facts about the software. |
| **Drawing** | Box positions, edge routing, labels, selection, interaction. | The authoritative meaning of a relationship. |

For example, collapsing Storage should change the view, not remove its children from the model. Dragging a box should change its position, not silently change which component contains it.

This also gives humans and agents equivalent access. A human selects a relationship in the drawing; an agent inspects its ID. Both reach the same underlying record.

**The same component should be usable in many views without copying its identity or evidence.**

### Nesting needs an explicit meaning

“Inside this box” could mean several different things:

- The class belongs to this package.
- The component runs inside this process.
- The functionality belongs to this subsystem.

Those are not interchangeable.

I would support **named hierarchies**, such as `architecture`, `code`, and `deployment`. Within a selected hierarchy and interpretation, active containment should form a forest: no containment cycles and at most one parent per entity.

Overlapping categories—“uses networking”, “third-party code”, “needs investigation”—should be tags or query-defined groups, not additional containment parents.

A view may display an alias of a shared component in another location, but the alias must retain the same entity ID and be visibly a reference.

---

## 2. A small core data model

I would start with six concepts rather than a large ontology.

| Concept | Purpose | Example |
|---|---|---|
| **Entity** | Something being modelled, with a stable identity. | Application, Sync service, Message database, function. |
| **Assertion** | A statement about an entity or relationship. | “Sync writes to Message database.” |
| **Evidence reference and link** | Identifies a source and explains how it relates to an assertion. | A particular trace event supports the write relationship. |
| **Scope** | Identifies the software and conditions the assertion concerns. | Build digest, runtime session, configuration. |
| **Commit** | An atomic, attributed change to the model. | Added Storage and three relationships after inspecting a trace. |
| **View specification** | A reproducible selection and presentation of the model. | Storage boundary, build A, supported relationships only. |

### Entities need stable IDs, not meaningful paths

An entity should have an immutable ID, a human-readable label, a kind, and optional aliases or external identifiers.

Renaming `a.b.c` to `MessageStore`, or moving it from “Unknown” into “Storage”, should not invalidate its evidence or history.

Caller-supplied keys can make runtime construction convenient, but they should resolve to stable IDs independently of the containment path.

This is one reason not to use a diagram renderer’s identifiers as your canonical identities: D2 documents that moving containers can change descendant and relationship identifiers. Keep that translation inside the rendering adapter. [D2 Documentation](https://d2lang.com/tour/api/)

### Relationships should be first-class assertions

Do not store relationships as incidental strings attached to boxes.

A relationship needs its own identity, predicate, endpoints, scope, provenance, and evidence links. That allows you to inspect, challenge, supersede, or compare it independently.

Start with a small vocabulary:

`contains`, `calls`, `reads`, `writes`, `publishes`, `subscribes`, `implements`, and `depends_on`.

Allow namespaced extensions, but give each predicate a definition. For example, distinguish a static callsite from a call observed during execution rather than making both silently mean the same thing.

Important properties can also be assertions. However, I would not turn every display label and colour into a knowledge claim: ordinary presentation metadata can remain ordinary versioned fields.

---

## 3. Make relationships clear without making diagrams verbose

The central design challenge is not nesting rectangles. It is **showing enough relationships to explain the system without showing every relationship at once**.

I would prioritise the following visual features.

| Feature | Behaviour |
|---|---|
| **Expand and collapse** | Reveal one component’s internals without expanding everything else. |
| **Relationship lenses** | Show calls, data access, deployment, or another selected relationship family. |
| **Boundary view** | Show what enters or leaves a component, including relationships attached to its descendants. |
| **Relationship aggregation** | Combine similar relationships between collapsed components, with counts and drill-down. |
| **Focus mode** | Keep a selected component and relevant neighbours prominent; suppress unrelated detail. |
| **Semantic zoom** | Progress from subsystem summaries to components, symbols, and evidence—not merely larger text. |
| **Stable layout** | Preserve the user’s spatial understanding when a small part of the model changes. |
| **Linked inspector** | Keep detailed descriptions, evidence, history, and uncertainty outside the main diagram. |

### Use concise verbs, not vague arrows

A label such as `writes` is generally more useful than `uses`.

Keep protocol, payload, conditions, source locations, and longer explanations in the inspector. Let a selected lens promote one of those fields onto the diagram when relevant.

Optional **ports** could identify meaningful interaction points: a component’s public API, event input, or database interface. They should be optional; modelling an interface should not be a prerequisite for drawing a useful relationship.

### Collapsed arrows must remain explainable

Suppose five functions inside Sync write to two stores inside Storage. A collapsed view could display:

> Sync — **writes · 5 relationships** → Storage

Selecting it should reveal the five original relationship records and their evidence.

The aggregate must record—or be able to reproduce—the exact member set at that revision. It must not become a newly asserted fact merely because the renderer created it.

Counts also need precise meanings: five relationship records is not five runtime calls, five independent observations, or five pieces of evidence.

### A crucial trap: aggregation can invent apparent paths

Suppose one child of component B receives calls from A, while a completely unrelated child of B calls C. Collapsing B can make the drawing appear to show a continuous route from A through B to C.

That route may not exist in the underlying model.

**Path queries must run against the underlying relationships, never against the collapsed drawing.** The drawing should expose the concrete path when a path result is selected.

Likewise, “neighbours of Storage” must distinguish relationships directly attached to Storage from those attached to its descendants. Cytoscape.js explicitly documents this distinction: ordinary neighbourhood algorithms do not automatically incorporate compound-node descendants. [js.cytoscape.org](https://js.cytoscape.org/)

### Visual direction

For your viewer, I would use dark surfaces, subtly differentiated nested containers, restrained borders, short titles, and generous padding. The main arrangement would be a central diagram, a contextual inspector, and a thin revision strip.

Selection and uncertainty should have textual or shape cues as well as colour. A keyboard-accessible outline should provide an alternative to navigating the canvas.

---

## 4. Represent uncertainty without making everything uncertain

You want to build understanding incrementally, so the library must accommodate incomplete and mistaken interpretations.

I would distinguish three questions about an assertion:

| Dimension | Question | Example values |
|---|---|---|
| **Basis** | How was this statement obtained? | Declared, statically extracted, runtime-observed, inferred. |
| **Disposition** | How is it treated in the working model? | Proposed, accepted, rejected, superseded. |
| **Applicability** | Does it apply to the selected target and conditions? | Applicable, stale, unresolved. |

These dimensions should not become three large badges on every arrow. The view can show a compact summary and expose the details on selection.

For example, an observation can be accepted but apply only to one runtime session. An inferred relationship can have supporting evidence while still being disputed.

**Accepted should mean “admitted into this working model”, not “universally proven”.**

I would avoid numerical confidence scores initially. “Two supporting sources, one contradiction, applicable to build A” is more directly inspectable than an unexplained `0.87`.

### Preserve alternatives without duplicating the application

Your known-versus-possible-versus-observed models fit naturally here.

Use shared entities with different assertions and view filters:

**Working view:** the currently admitted interpretation.

**Evidence view:** assertions with relevant supporting observations.

**Hypothesis view:** proposed or disputed interpretations.

You do not need three disconnected copies of the software. Competing assertions can coexist while their disagreement remains explicit.

Where competing containment assertions cannot both fit a hierarchy, require the view to select an interpretation or display the ambiguity rather than silently choosing a parent.

---

## 5. Evidence should support specific statements

Attaching a file to a box is useful, but it is weaker than recording exactly what the file establishes.

Compare:

> A trace is attached to Storage.

with:

> Events 84–91 in this trace support the assertion that Sync wrote to Message database during session R.

I would support three evidence-link roles: **supports**, **contradicts**, and **context**. A convenient attachment to an entity should default to context rather than implicitly supporting every assertion about that entity.

An evidence reference should contain the artifact identity or content digest, its location, a precise selector, and its producer or capture record. Selectors might identify a source range, symbol, byte range, trace event, or screenshot region.

The W3C provenance model provides a useful foundation here: distinguish the artifact, the activity that produced it, and the person or software responsible. You can borrow that structure without adopting RDF or the entire specification. [W3C](https://www.w3.org/TR/prov-dm/)

### Keep the artifact and the interpretation separate

A digest identifies particular bytes; it does not establish that an interpretation of those bytes is correct.

Similarly, adding evidence should not automatically promote a claim to “verified”. The core can check that references resolve and scopes match. Whether the evidence genuinely supports the statement requires an explicit assessment policy.

When the target changes, retain the old evidence and mark its applicability to the new target as unresolved or stale. Do not rewrite yesterday’s observations.

For your existing evidence infrastructure, I would make artifact storage an adapter. The library should be able to reference an external content-addressed store rather than requiring you to create another one.

---

## 6. Querying should be a primary interface

I would expose a typed Go query API and a matching serialisable request format. The UI and an optional agent-tool adapter should call the same implementation.

Start with questions that directly help someone understand software:

| Question | Query behaviour |
|---|---|
| What is inside this component? | Children or descendants in a named hierarchy. |
| What communicates with anything inside it? | Incoming and outgoing boundary relationships. |
| How are these components connected? | Bounded paths over specified relationship types. |
| Why is this relationship present? | Assertion, evidence, assessment, and derivation explanation. |
| What changed since revision 12? | Semantic diff, optionally restricted to a component. |
| What is still unresolved here? | Proposed assertions, contradictions, stale evidence, open questions. |
| Who introduced or changed this interpretation? | History of the entity, assertion, or selected subgraph. |

A particularly useful operation is **Explain**. It should work on both canonical relationships and visual aggregates:

> This arrow represents seven `writes` assertions between descendants of Sync and Storage. Four have runtime evidence; three are inferred.

### Return bounded, inspectable answers

A query result should include its model revision, scope, filters, returned IDs, relevant evidence references, and whether it was truncated.

Make two kinds of completeness explicit:

**Query completeness:** did the query finish searching the selected model?

**Knowledge completeness:** how much of the actual software has been examined?

A fully completed query can still operate on a very incomplete model.

Consequently, “no relationship found” should not automatically become “these components never communicate”. An exhaustive negative claim needs its own scope and supporting method.

### Optimise agents for inspection, not screenshots

The agent-facing surface can stay small:

`Inspect`, `Query`, `Explain`, `Diff`, `Apply`, and `Watch`.

An agent should read a bounded context, submit a structured patch against a particular revision, and receive either a commit or explicit validation/conflict errors.

Natural-language querying can be an optional translation layer over this API. It should not be the only way to interrogate the model, and it should not execute arbitrary Go or SQL supplied by a model.

---

## 7. Save semantic revisions, not just snapshots of drawings

I would make every model change an atomic commit.

A commit should record its parent revision, actor, reason, resolved operations, affected IDs, and relevant source or run references.

The history should describe changes such as:

> Reclassified Parser as part of Import Pipeline.  
> Superseded the claim that Sync directly writes to Message database.  
> Added the intermediate Storage API relationship based on new evidence.

That is much more useful than a generic record saying “diagram updated”.

### Keep three timelines separate

**Model revision:** when your understanding changed.

**Target version:** which software build the statement concerns.

**Observation time or run:** when a behaviour was recorded.

Discovering something today about an old build should not look like the software changed today.

### Use a modest durable implementation

My initial implementation would be **SQLite-backed commits, indexed current state, and periodic snapshots**. SQLite supports transactions and recursive queries over trees and graphs, so these requirements do not immediately require a separate graph database. [SQLite](https://sqlite.org/lang_transaction.html)

Write the commit and update the current-state tables in the same transaction. Publish change notifications only after it succeeds.

Store resolved semantic operations in the journal—not callbacks or instructions to rerun an external analysis tool. Reconstructing an old revision must not invoke an LLM, decompiler, or live service.

For concurrency, begin with optimistic revision checks. A patch based on an outdated revision should receive a conflict rather than silently overwrite another contributor’s changes. Fine-grained rebasing can come later.

Undo should create a compensating commit. Superseded entities and assertions should remain addressable in historical revisions.

An append-only application journal provides history, but it is not automatically a tamper-proof audit system. Stronger integrity guarantees would require additional controls.

### Preserve your code-as-source-of-truth workflow

Your Go program should remain the source of the **model-building logic**, while execution records preserve what was actually recorded.

UI edits that affect meaning should produce explicit, exportable patches that the analysis program can consume. Coordinates and transient selections can remain view state.

Repeated imports should use stable keys and provenance so that rerunning a script does not duplicate components or overwrite human annotations accidentally.

---

## 8. What the Go API could look like

I would use a staged builder for convenience, backed by explicit typed operations.

The following is a **proposed API sketch**, not an existing package:

```go
func RecordStorage(
    ctx context.Context,
    m *boxmodel.Model,
    base boxmodel.RevisionID,
    target boxmodel.TargetRef,
    evidence boxmodel.EvidenceRef,
) (boxmodel.Commit, error) {
    p := boxmodel.NewPatch(boxmodel.Change{
        ExpectedRevision: base,
        Actor:            "agent:mapper",
        Reason:           "Add storage structure and a candidate writer.",
        Scope:            boxmodel.Scope{Target: target},
    })

    app := p.Entity("app", boxmodel.System, "Application")
    sync := p.Entity("sync", boxmodel.Component, "Sync")
    storage := p.Entity("storage", boxmodel.Component, "Storage")
    db := p.Entity("message-db", boxmodel.DataStore, "Message database")

    p.Contains("architecture", app, sync)
    p.Contains("architecture", app, storage)
    p.Contains("architecture", storage, db)

    writes := p.Relate(
        "sync-writes-messages",
        sync,
        boxmodel.Writes,
        db,
    )
    writes.Basis(boxmodel.Inferred)
    writes.Support(evidence)

    return m.Apply(ctx, p)
}
```

Here the builder only stages operations. `Apply` validates the whole patch and commits it atomically, returning structured errors if necessary.

The short keys resolve within a workspace; they are not containment paths. Repeated declarations with the same key and content should be idempotent. Conflicting declarations should require an explicit update rather than guessing the caller’s intent.

Recording claims does not automatically establish them as facts. An admission policy determines which assertions enter an accepted view; a working view can also display proposals with appropriate styling.

I would provide an in-memory implementation for tests and ephemeral diagrams, with persistence added through an adapter. Creating a useful model should not require starting a server.

### Keep the package boundaries small

| Package area | Responsibility |
|---|---|
| Core | Types, patches, validation, snapshots, queries, view projection. |
| SQLite adapter | Durable commits, current-state indexes, historical reconstruction. |
| Export adapter | JSON model export and D2/SVG output. |
| Optional web viewer | Interactive diagrams, inspector, revisions, change stream. |
| Optional agent adapter | Typed tool requests and responses over the core API. |

Rendering and storage adapters should depend on the core—not the reverse. Decompilers, source analysers, and agents should be external producers of patches.

---

## 9. Reuse layout tools, but own the projection rules

For an initial export path, I would use D2. Its official examples demonstrate compiling and rendering SVG from Go using `d2lib` and `d2svg`. [GitHub](https://github.com/terrastruct/d2/blob/master/docs/examples/lib/1-d2lib/d2lib.go)

For an interactive nested-box viewer, I would prototype **SVG rendering with ELK.js for layout**. ELK Layered supports ports, orthogonal routing, compound graphs, and cross-hierarchy edges; ELK.js exposes the layout algorithms to JavaScript and supports Web Workers. That makes it a candidate for the optional browser adapter, not a native Go dependency. [Eclipse Foundation](https://eclipse.dev/elk/reference/algorithms/org-eclipse-elk-layered.html)

The part you should own is the **projection engine**: the code that turns a model query into the visible boxes and relationships.

It decides which entities appear, which ancestors must remain visible, where hidden endpoints are represented, how edges are grouped, and how every drawing item maps back to the model.

A layout engine should receive that projected structure. It should not decide what the diagram means.

Treat layout stability as a benchmark requirement, not something automatically guaranteed by choosing an engine. Test whether a small update moves unrelated components, preserve previous layout hints where supported, and allow deliberate full relayout.

### Scale the model independently of the canvas

For your large-software use case, distinguish the underlying corpus from the explanatory model. Thirty thousand files may be indexed as evidence or code entities without becoming thirty thousand visible boxes.

Give views an explicit display budget and show what was omitted. Load detail on expansion, batch updates by commit, and invalidate only affected projections where possible.

This matters even with an established graph renderer: Cytoscape.js documents increasing rendering costs from large graphs, edges, labels, and compound nodes. [js.cytoscape.org](https://js.cytoscape.org/)

---

## 10. The feature set I would actually build first

I would organise implementation around three gates.

| Gate | Features | Test that earns further investment |
|---|---|---|
| **1. Explainable model** | Stable entities, typed relationships, evidence links, atomic commits, basic queries, static diagrams. | A fresh human or agent can explain a relationship and reconstruct an earlier interpretation without reading the authoring code. |
| **2. Useful interactive views** | Collapse/expand, boundary queries, relationship lenses, aggregate drill-down, inspector, semantic diffs. | Simplifying the view preserves access to the original relationships and does not produce false path answers. |
| **3. Larger concurrent investigations** | Incremental imports, bounded views, revision conflicts, change subscriptions, stale-evidence tracking. | Repeated imports do not duplicate knowledge, concurrent edits do not disappear, and small changes do not rebuild everything. |

The deterministic checks I would prioritise are identity preservation across rename/move, containment-cycle rejection, valid relationship endpoints, scope-compatible queries, exact aggregate membership, replay consistency, atomic conflict handling, and explicit reporting of truncation.

I would defer unrestricted branching and merging, a custom query language, automatic architecture discovery, numerical belief propagation, and a bespoke layout algorithm.

**The first product should let someone build a small model, ask what it means, inspect why it says that, and see how that understanding changed.** Once those operations are reliable, richer diagrams and integrations have a solid foundation.