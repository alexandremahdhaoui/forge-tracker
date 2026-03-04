# forge-tracker Design

**A file-based ticket tracker with graph relationships, exposed over REST.**

## Problem Statement

AI agents and humans need to share a ticket tracker during development. Hosted
trackers (Jira, Linear) require API keys, network access, and cannot be
version-controlled alongside code. A local, file-based tracker with a REST API
solves this. Tickets must support directed relationships (parent, blocks,
relates-to) to model task hierarchies and dependencies. The storage format
must be human-readable and diffable.

forge-tracker stores tickets as markdown files with YAML frontmatter on the
local filesystem. It exposes 22 REST endpoints over HTTP. A directed graph per
TrackingSet tracks 3 edge types between tickets.

## Tenets

1. **File-first.** Data lives on disk as markdown and YAML. No database.
2. **Human-readable.** A developer can read and edit tickets in any text editor.
3. **Graph-aware.** Relationships between tickets are first-class, not metadata.
4. **API-driven.** All mutations go through the REST API for consistency.
5. **Simple deployment.** Single binary, zero external dependencies.

When tenets conflict, higher-ranked wins. File-first beats API-driven: if the
API is down, files remain accessible.

## Requirements

- Store tickets with id, title, status, priority (0-3), labels, annotations,
  assignee, timestamps, description, and comments.
- Support 3 directed edge types: parent, blocks, relates-to.
- Group tickets into TrackingSets (tenancy boundary).
- Support Plans (tickets with child task references via parent edges).
- Support MetaPlans (cross-repo coordination with stages and checkpoints).
- Expose all operations over a RESTful JSON API.
- Filter tickets by status, assignee, labels (AND semantics), and priority.
- Persist all data as human-readable files (markdown + YAML).
- Use atomic writes to prevent corruption on crash.

## Out of Scope

- Authentication and authorization.
- Multi-user concurrency control (file-level locking).
- Real-time notifications or webhooks.
- Full-text search beyond label index.
- Remote storage backends (S3, databases).
- Ticket state machine (status transitions are unconstrained strings).

## Success Criteria

- All 22 REST endpoints pass integration tests.
- Ticket CRUD completes in under 10ms for a TrackingSet with 1000 tickets.
- Label-filtered list queries use O(1) index lookup per label, not full scan.
- No partial writes on crash (atomic rename).
- Zero external runtime dependencies (single static binary).

## Proposed Design

### Architecture

```
+------------------------------------------------------------------+
|                         forge-tracker                             |
|                                                                  |
|  +------------------+    +-------------------+                   |
|  |  cmd/            |    |  api/             |                   |
|  |  forge-tracker/  |--->|  forge-tracker.   |                   |
|  |  main.go         |    |  v1.yaml          |                   |
|  +--------+---------+    +--------+----------+                   |
|           |                       |  (codegen)                   |
|  +--------v-----------------------v----------+                   |
|  |              internal/driver/rest/        |   Layer 1: Driver |
|  |  handler.go  zz_generated.oapi-codegen.go |                   |
|  +-------------------+----------------------+                    |
|                      |                                           |
|  +-------------------v----------------------+                    |
|  |           internal/controller/           |   Layer 2: Logic   |
|  |  ticket.go  graph.go  trackingset.go     |                   |
|  |  plan.go    metaplan.go                  |                   |
|  +-------------------+---------------------+                    |
|                      |                                           |
|  +-------------------v---------------------+                    |
|  |           internal/adapter/             |   Layer 3: Port     |
|  |  adapter.go (5 interfaces)              |                    |
|  +-------------------+---------------------+                    |
|                      |                                           |
|  +-------------------v---------------------+                    |
|  |      internal/adapter/markdown/         |   Layer 4: Impl    |
|  |  store.go  reader.go  writer.go         |                    |
|  |  graph.go  trackingset.go  plan.go      |                    |
|  |  metaplan.go  index.go                  |                    |
|  +-------------------+---------------------+                    |
|                      |                                           |
|              +-------v--------+                                  |
|              |   Filesystem   |                                  |
|              +----------------+                                  |
+------------------------------------------------------------------+
```

Dependencies flow inward. The driver depends on controllers. Controllers
depend on adapter interfaces. The markdown package implements those interfaces.

### Request Flow

```
Client                REST Handler          Controller           Adapter
  |                       |                     |                   |
  |  POST /tickets        |                     |                   |
  |---------------------->|                     |                   |
  |                       |  Create(ts, ticket) |                   |
  |                       |-------------------->|                   |
  |                       |                     |  Validate fields  |
  |                       |                     |  Set timestamps   |
  |                       |                     |                   |
  |                       |                     |  tickets.Create() |
  |                       |                     |------------------>|
  |                       |                     |                   | WriteTicket()
  |                       |                     |                   | atomicWrite()
  |                       |                     |                   | index.Add()
  |                       |                     |  graphs.AddNode() |
  |                       |                     |------------------>|
  |                       |                     |                   | readTrackingSet()
  |                       |                     |                   | append node
  |                       |                     |                   | writeTrackingSet()
  |                       |  201 + Ticket JSON  |                   |
  |<----------------------|---------------------|                   |
  |                       |                     |                   |
```

Creating a ticket involves 2 store operations: write the markdown file and add
the ticket ID as a node in the TrackingSet graph.

### Storage Layout

```
{storage-path}/
  tracking-sets/
    {ts-name}/
      tracking-set.yaml            <-- YAML: name, graph (nodes + edges)
      tickets/
        {ticket-id}.md             <-- YAML frontmatter + markdown body
        {ticket-id}.metaplan.yaml  <-- YAML: stages + checkpoints
```

The `tracking-set.yaml` file holds the full graph (nodes and edges) for one
TrackingSet. Each ticket is a single `.md` file. MetaPlan data splits across
2 files: the ticket markdown and a `.metaplan.yaml` sidecar.

### Graph Relationships

```
    t-1 (Plan)
     |
     | parent         t-3
     |                 |
     v                 | blocks
    t-2 ---------------+
         relates-to    |
    t-4 <--------------+
```

- `parent`: t-1 is the parent of t-2 (plan-to-task hierarchy).
- `blocks`: t-3 blocks t-2 (t-2 cannot proceed until t-3 completes).
- `relates-to`: t-3 relates to t-4 (informational link).

Edges are stored in `tracking-set.yaml` as a list of `{from, to, type}` tuples.
Graph queries (GetChildren, GetBlocking) filter this list by edge type and
direction.

## Technical Design

### Data Model

```go
type Ticket struct {
    ID          string
    Title       string
    Status      string            // unconstrained (e.g. "pending", "done")
    Priority    int               // 0=critical, 1=high, 2=medium, 3=low
    Labels      []string
    Annotations map[string]string
    Assignee    string
    Created     time.Time
    Updated     time.Time
    Description string            // parsed from markdown body
    Comments    []Comment         // parsed from ### headers
}

type Comment struct {
    Timestamp time.Time
    Author    string
    Text      string
}

type Edge struct {
    From string                   // source ticket ID
    To   string                   // target ticket ID
    Type string                   // "parent", "blocks", "relates-to"
}

type Graph struct {
    Nodes []string                // ticket IDs
    Edges []Edge
}

type TrackingSet struct {
    Name  string
    Graph Graph
}

type Plan struct {
    Ticket                        // embeds Ticket
    Tasks  []string               // child IDs (derived from parent edges)
}

type MetaPlan struct {
    Ticket                        // embeds Ticket
    Stages      []MetaPlanStage
    Checkpoints []MetaCheckpoint
}

type MetaPlanStage struct {
    Name   string
    Status string
    Repos  []StageRepoRef        // {Name, PlanID, TasksTotal, TasksDone}
}

type MetaCheckpoint struct {
    Name      string
    Stage     string
    Condition string
    Met       bool
}
```

### REST API Catalog

22 endpoints under `/api/v1`:

| Resource | Endpoints | Operations |
|----------|-----------|------------|
| TrackingSet | 4 | create, list, get, delete |
| Ticket | 5 | create, list, get, update, delete |
| Comment | 1 | add |
| Edge | 3 | list, add, remove |
| Children | 1 | get |
| Blocking | 1 | get |
| Plan | 5 | create, list, get, update, delete |
| MetaPlan | 5 | create, list, get, update, delete |

Plus 1 health check endpoint (`/healthz`). Total: 23 routes.

Error responses use `{"error": "message"}` with appropriate HTTP status codes
(400 for validation, 404 for not found, 500 for internal errors).

### Adapter Interfaces

5 interfaces defined in `internal/adapter/adapter.go`:

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `TicketStore` | 6 | CRUD + List + AddComment |
| `GraphStore` | 8 | Get, AddNode, RemoveNode, AddEdge, RemoveEdge, GetEdges, GetChildren, GetBlocking |
| `TrackingSetStore` | 4 | CRUD |
| `PlanStore` | 5 | CRUD + List |
| `MetaPlanStore` | 5 | CRUD + List |

2 sentinel errors: `ErrNotFound`, `ErrValidation`.

### Controller Services

5 services defined in `internal/controller/`:

| Service | Dependencies | Key Logic |
|---------|-------------|-----------|
| `TicketService` | TicketStore, GraphStore | Creates ticket + graph node atomically. Sets default status "pending". |
| `GraphService` | GraphStore, TicketStore | Validates edge types (parent, blocks, relates-to). Loads full Ticket objects for children/blocking queries. |
| `TrackingSetService` | TrackingSetStore | Validates name format: `^[a-zA-Z0-9_-]+$`. |
| `PlanService` | PlanStore, GraphStore | Creates parent edges for each task in the plan. |
| `MetaPlanService` | MetaPlanStore | Validates non-empty ID. |

### Package Catalog

| Package | Path | Purpose |
|---------|------|---------|
| `main` | `cmd/forge-tracker/` | Binary entry point. Wires stores, controllers, handler. |
| `rest` | `internal/driver/rest/` | HTTP handler (OpenAPI codegen) + request mapping. |
| `controller` | `internal/controller/` | 5 service interfaces + implementations. Business logic. |
| `adapter` | `internal/adapter/` | 5 store interfaces + error sentinels. |
| `markdown` | `internal/adapter/markdown/` | File-based store: 8 files (store, reader, writer, graph, trackingset, plan, metaplan, index). |
| `types` | `internal/types/` | Domain types: Ticket, Edge, Graph, TrackingSet, Plan, MetaPlan. |
| `integration` | `test/integration/` | HTTP integration tests (3 test functions). |
| `mockcontroller` | `internal/util/mocks/mockcontroller/` | Generated mocks for controller interfaces. |
| `mockadapter` | `internal/util/mocks/mockadapter/` | Generated mocks for adapter interfaces. |

## Design Patterns

**Atomic writes.** Every file mutation creates a temp file in the same
directory, writes content, then renames over the target. This guarantees no
partial writes survive a crash.

**Inverted label index.** On startup, the markdown store scans all tickets and
builds an in-memory map: `trackingSet -> label -> set(ticketID)`. Queries
intersect sets for AND semantics. The index updates incrementally on
create/update/delete.

**Compile-time interface checks.** Each adapter implementation includes
`var _ Interface = (*impl)(nil)` to catch interface mismatches at compile time
rather than runtime.

**Strict handler pattern.** OpenAPI codegen produces a `StrictServerInterface`.
The `APIHandler` implements it, receiving typed request objects and returning
typed response objects. No manual HTTP parsing.

**Graph-derived plan tasks.** A Plan does not store its task list. The
`PlanStore.Get` method queries the graph for `parent` edges where the plan is
the source, then returns child IDs. This keeps the graph as the single source
of truth for relationships.

## Alternatives Considered

**Do nothing (use an external tracker).**
Rejected. External trackers require network access and cannot be
version-controlled. AI agents need local, file-based access.

**SQLite backend.**
Rejected. SQLite adds a binary dependency and makes files non-human-readable.
Markdown files are diffable and editable in any text editor.

**Embed graph in each ticket file.**
Rejected. Distributing graph edges across ticket files makes graph queries
require scanning all files. A central `tracking-set.yaml` keeps graph queries
to 1 file read.

**gRPC instead of REST.**
Rejected. REST with JSON is simpler to call from shell scripts and AI agents.
No protobuf compilation step needed.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| File contention under concurrent access | Data loss | Atomic writes prevent corruption. No concurrent write safety beyond OS-level rename atomicity. Document single-writer assumption. |
| Large TrackingSets (10,000+ tickets) slow list queries | High latency | Label index reduces scan scope. Plan: add pagination to list endpoints. |
| `tracking-set.yaml` becomes bottleneck | Write contention | Every graph mutation reads and rewrites the full file. Acceptable for target scale (100s of tickets). Plan: split into per-node edge files if needed. |
| OpenAPI codegen drift | Build failures | Generated code is committed (`zz_generated.oapi-codegen.go`). `forge build generate-rest-api` regenerates from spec. |

## Testing Strategy

4 test stages run via `forge test-all`:

| Stage | Tool | Scope |
|-------|------|-------|
| `lint-tags` | custom | Verify build tags on test files |
| `lint-licenses` | custom | Verify Apache 2.0 headers |
| `lint` | golangci-lint v2 | Static analysis |
| `unit` | `go test -tags unit` | 13 test files: controller tests (4), adapter tests (8), integration tests (1) |

Integration tests (`test/integration/api_test.go`) start an `httptest.Server`
with the full stack (markdown store, controllers, REST handler) and exercise
the API end-to-end. 3 test functions cover: full ticket lifecycle with edges
and comments, plan creation with task references, and meta-plan CRUD.

Controller tests use generated mocks (`mockadapter/`, `mockcontroller/`) from
mockery.

## FAQ

**Why store the graph in tracking-set.yaml instead of a separate file?**
The TrackingSet already acts as the tenancy manifest. Keeping nodes and edges
together means creating a TrackingSet initializes the graph in 1 file write.

**Why not enforce a ticket status state machine?**
Status is a free-form string. Different teams use different workflows. The
tracker stores state; the workflow engine (or human) enforces transitions.

**How does the Plan task list stay in sync with the graph?**
It does not store tasks independently. `PlanStore.Get` queries graph edges at
read time. The graph is always the source of truth.

**Why split MetaPlan data across 2 files?**
The ticket markdown file holds the same fields as any ticket. The
`.metaplan.yaml` sidecar holds structured data (stages, checkpoints) that does
not fit the markdown format. This avoids overloading frontmatter.

## Appendix: forge.yaml

```yaml
name: forge-tracker

build:
  - name: generate-rest-api
    src: ./api/forge-tracker.v1.yaml
    dest: ./internal/driver/rest
    engine: alias://go-gen-openapi-with-license

  - name: generate-mocks
    src: .
    engine: alias://generate-mocks

  - name: forge-tracker
    src: ./cmd/forge-tracker
    dest: ./build/bin
    engine: go://go-build

test:
  - name: lint-tags
    runner: go://go-lint-tags

  - name: lint-licenses
    runner: go://go-lint-licenses

  - name: lint
    runner: go://go-lint

  - name: unit
    runner: go://go-test
```
