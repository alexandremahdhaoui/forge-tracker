# Contributing to forge-tracker

**Build, test, and submit changes to forge-tracker.**

## Quick Start

```bash
# Clone and enter the repo.
git clone https://github.com/alexandremahdhaoui/forge-tracker.git
cd forge-tracker

# Generate code (OpenAPI server + mocks).
forge build generate-rest-api
forge build generate-mocks

# Build the binary.
forge build forge-tracker

# Run all tests.
forge test-all
```

## How do I structure commits?

Each commit uses an emoji prefix and a structured body.

| Emoji | Meaning |
|-------|---------|
| `✨` | New feature |
| `🐛` | Bug fix |
| `📖` | Documentation |
| `🌱` | Misc (chore, test, refactor) |
| `⚠` | Breaking change (requires maintainer approval) |

Commit body format:

```
✨ Short imperative summary (50 chars or less)

Why: Explain the motivation.

How: Describe the approach.

What:

- internal/controller/ticket.go: added priority validation
- internal/adapter/markdown/store.go: updated atomic write path

How changes were verified:

- forge test-all: all stages passed

Signed-off-by: Your Name <your@email.com>
```

Every commit requires `Signed-off-by`. Use `git commit -s` to add it.

## How do I submit a pull request?

1. Create a feature branch from `main`.
2. Make changes. Run `forge test-all`. All 4 stages must pass.
3. Commit with the format above.
4. Open a PR. Include a summary and test plan.

PR title: under 70 characters, imperative mood.

## How do I run tests?

```bash
# Run all stages (lint-tags, lint-licenses, lint, unit).
forge test-all

# Run a specific stage.
forge test run lint
forge test run unit

# List all test stages.
forge list test
```

| Stage | What it checks |
|-------|---------------|
| `lint-tags` | Build tags on test files |
| `lint-licenses` | Apache 2.0 license headers |
| `lint` | golangci-lint v2 static analysis |
| `unit` | Go tests with `-tags unit` (13 test files) |

Integration tests live in `test/integration/api_test.go`. They start a full
HTTP test server and exercise all API endpoints. They run as part of the
`unit` stage (build tag: `unit`).

## How is the project structured?

```
forge-tracker/
  api/
    forge-tracker.v1.yaml            # OpenAPI 3.0 spec (1325 lines)
  cmd/
    forge-tracker/
      main.go                        # Binary entry point
  internal/
    driver/
      rest/
        handler.go                   # API handler (maps HTTP to controllers)
        zz_generated.oapi-codegen.go # Generated server code
    controller/
      ticket.go                      # TicketService
      graph.go                       # GraphService
      trackingset.go                 # TrackingSetService
      plan.go                        # PlanService
      metaplan.go                    # MetaPlanService
      *_test.go                      # Controller unit tests (4 files)
    adapter/
      adapter.go                     # 5 store interfaces + error sentinels
      markdown/
        store.go                     # Store factory + TicketStore impl
        reader.go                    # ParseTicket (YAML frontmatter parser)
        writer.go                    # WriteTicket + AppendComment
        graph.go                     # GraphStore + TrackingSet I/O helpers
        trackingset.go               # TrackingSetStore
        plan.go                      # PlanStore
        metaplan.go                  # MetaPlanStore
        index.go                     # In-memory inverted label index
        *_test.go                    # Adapter unit tests (8 files)
    types/
      types.go                       # Domain types (8 types)
    util/
      mocks/                         # Generated mocks (mockery)
  test/
    integration/
      api_test.go                    # Full HTTP integration tests
  forge.yaml                         # Build + test configuration
  .golangci.yml                      # Linter config
  .mockery.yml                       # Mock generation config
```

## What does each package do?

### Public packages

| Package | Path | Purpose |
|---------|------|---------|
| `main` | `cmd/forge-tracker/` | Wires stores, controllers, and handler. Starts HTTP server. |

### Internal packages

| Package | Path | Files | Purpose |
|---------|------|-------|---------|
| `rest` | `internal/driver/rest/` | 2 | HTTP handler + generated OpenAPI server code |
| `controller` | `internal/controller/` | 5 | Business logic: validation, timestamps, graph consistency |
| `adapter` | `internal/adapter/` | 1 | Store interfaces: TicketStore, GraphStore, TrackingSetStore, PlanStore, MetaPlanStore |
| `markdown` | `internal/adapter/markdown/` | 8 | File-based store: atomic writes, YAML frontmatter, label index |
| `types` | `internal/types/` | 1 | Domain types: Ticket, Edge, Graph, TrackingSet, Plan, MetaPlan, MetaPlanStage, MetaCheckpoint |
| `mockcontroller` | `internal/util/mocks/mockcontroller/` | 5 | Generated controller mocks |
| `mockadapter` | `internal/util/mocks/mockadapter/` | 5 | Generated adapter mocks |

## What conventions must I follow?

**Build tags.** All test files require the `unit` build tag:
```go
//go:build unit
```

**License headers.** Every `.go` file starts with the Apache 2.0 license
header. The `lint-licenses` stage enforces this. Run
`./hack/add-license-headers.sh` to add missing headers.

**Generated files.** Files prefixed with `zz_generated.` are generated. Do
not edit them by hand. Regenerate with:
```bash
forge build generate-rest-api    # OpenAPI server code
forge build generate-mocks       # Mockery mocks
```

**Code style.** golangci-lint v2 enforces style. Run `forge test run lint`
before committing.

**Interface compliance.** Every adapter implementation must include a
compile-time check:
```go
var _ adapter.TicketStore = (*ticketStore)(nil)
```

**Atomic writes.** All file mutations must use the `atomicWrite` helper
(temp file + rename). Direct `os.WriteFile` calls are not allowed.
