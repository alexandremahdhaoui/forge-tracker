# forge-tracker

**File-based ticket tracking with a RESTful API, graph relationships, and markdown storage.**

> "I need a tracker that AI agents and humans share without a database.
> forge-tracker stores tickets as markdown files with YAML frontmatter,
> exposes a REST API, and keeps a relationship graph -- all on disk."

## What problem does forge-tracker solve?

Teams running AI-assisted development need a ticket tracker that both agents
and humans can read, edit, and version-control. Traditional trackers lock data
in databases, blocking git-native workflows. forge-tracker stores tickets as
markdown files on the local filesystem. It exposes a JSON REST API on port
8081. Tickets link to each other through a directed graph with 3 edge types:
parent, blocks, and relates-to.

## Quick Start

```bash
# Build the binary.
forge build forge-tracker

# Start the server (data stored in ./data/).
./build/bin/forge-tracker --storage-path ./data --addr :8081

# Create a tracking set.
curl -X POST http://localhost:8081/api/v1/tracking-sets \
  -H 'Content-Type: application/json' \
  -d '{"name": "my-project"}'

# Create a ticket.
curl -X POST http://localhost:8081/api/v1/tracking-sets/my-project/tickets \
  -H 'Content-Type: application/json' \
  -d '{"id": "t-1", "title": "Implement feature X", "priority": 1}'

# List tickets.
curl http://localhost:8081/api/v1/tracking-sets/my-project/tickets
```

## How does it work?

```
                      +------------------+
                      |   HTTP Client    |
                      +--------+---------+
                               |
                      +--------v---------+
                      |   REST Handler   |  internal/driver/rest/
                      | (OpenAPI codegen)|
                      +--------+---------+
                               |
              +----------------+----------------+
              |                |                |
     +--------v-----+ +-------v------+ +-------v--------+
     |TicketService  | |GraphService  | |TrackingSetSvc  |
     |PlanService    | |              | |MetaPlanService  |
     +---------+-----+ +------+------+ +-------+--------+
              |                |                |
              +----------------+----------------+
                               |
                      +--------v---------+
                      | Markdown Adapter |  internal/adapter/markdown/
                      |  (file I/O)      |
                      +--------+---------+
                               |
                      +--------v---------+
                      |   Filesystem     |
                      |  tracking-sets/  |
                      |    {name}/       |
                      |      tickets/    |
                      +------------------+
```

forge-tracker follows a 4-layer architecture. The REST handler receives HTTP
requests and delegates to 5 controller services. Controllers enforce business
rules (validation, timestamp management, graph consistency). The markdown
adapter reads and writes tickets as `.md` files with YAML frontmatter. All
writes use atomic temp-file-plus-rename for durability. See [DESIGN.md](DESIGN.md)
for full technical details.

## Table of Contents

- [How do I use the API?](#how-do-i-use-the-api)
- [How do I configure the server?](#how-do-i-configure-the-server)
- [How do I build and test?](#how-do-i-build-and-test)
- [What does the storage format look like?](#what-does-the-storage-format-look-like)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing and License](#contributing-and-license)

## How do I use the API?

All endpoints live under `/api/v1`. The parameter `{ts}` is a TrackingSet
name (alphanumeric, hyphens, underscores). `{id}` is a ticket ID.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/tracking-sets` | Create a tracking set |
| GET | `/tracking-sets` | List tracking sets |
| GET | `/tracking-sets/{ts}` | Get a tracking set |
| DELETE | `/tracking-sets/{ts}` | Delete a tracking set |
| POST | `/tracking-sets/{ts}/tickets` | Create a ticket |
| GET | `/tracking-sets/{ts}/tickets` | List tickets (filter: status, assignee, labels, priority) |
| GET | `/tracking-sets/{ts}/tickets/{id}` | Get a ticket |
| PUT | `/tracking-sets/{ts}/tickets/{id}` | Update a ticket |
| DELETE | `/tracking-sets/{ts}/tickets/{id}` | Delete a ticket |
| POST | `/tracking-sets/{ts}/tickets/{id}/comments` | Add a comment |
| GET | `/tracking-sets/{ts}/edges` | List edges (filter: ticket, type) |
| POST | `/tracking-sets/{ts}/edges` | Add an edge (parent, blocks, relates-to) |
| DELETE | `/tracking-sets/{ts}/edges` | Remove an edge |
| GET | `/tracking-sets/{ts}/tickets/{id}/children` | Get child tickets |
| GET | `/tracking-sets/{ts}/tickets/{id}/blocking` | Get blocking tickets |
| POST | `/tracking-sets/{ts}/plans` | Create a plan |
| GET | `/tracking-sets/{ts}/plans` | List plans (filtered by `kind:plan` label) |
| GET | `/tracking-sets/{ts}/plans/{id}` | Get a plan (includes tasks from graph) |
| PUT | `/tracking-sets/{ts}/plans/{id}` | Update a plan |
| DELETE | `/tracking-sets/{ts}/plans/{id}` | Delete a plan |
| POST | `/tracking-sets/{ts}/metaplans` | Create a meta-plan |
| GET | `/tracking-sets/{ts}/metaplans` | List meta-plans |
| GET | `/tracking-sets/{ts}/metaplans/{id}` | Get a meta-plan |
| PUT | `/tracking-sets/{ts}/metaplans/{id}` | Update a meta-plan |
| DELETE | `/tracking-sets/{ts}/metaplans/{id}` | Delete a meta-plan |
| GET | `/healthz` | Health check (returns `ok`) |

## How do I configure the server?

forge-tracker accepts 2 command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--storage-path` | (required) | Root directory for all tracking data |
| `--addr` | `:8081` | HTTP listen address |

The server enables CORS for all origins.

## How do I build and test?

```bash
# Generate OpenAPI server code and mocks, then build.
forge build

# Run all test stages (lint-tags, lint-licenses, lint, unit).
forge test-all

# Run a specific test stage.
forge test run unit
```

Build targets: `generate-rest-api`, `generate-mocks`, `forge-tracker`.
Test stages: `lint-tags`, `lint-licenses`, `lint`, `unit`.

## What does the storage format look like?

```
{storage-path}/
  tracking-sets/
    {name}/
      tracking-set.yaml          # graph: nodes + edges
      tickets/
        {id}.md                  # YAML frontmatter + description + comments
        {id}.metaplan.yaml       # stages + checkpoints (MetaPlan only)
```

A ticket file looks like this:

```markdown
---
id: t-1
title: Implement feature X
status: pending
priority: 1
labels:
  - kind:task
  - area:api
assignee: alice
created: 2025-01-15T10:00:00Z
updated: 2025-01-15T10:00:00Z
---

## Description

Implement the new feature X for the API layer.

## Comments

### 2025-01-15T11:00:00Z - bob

Looks good, starting review.
```

Priority values: 0 = critical, 1 = high, 2 = medium, 3 = low.

## FAQ

**Can I edit ticket files directly on disk?**
Yes. Ticket files are plain markdown. Edit them with any text editor or script.
Restart the server to rebuild the label index after direct edits.

**How does label filtering work?**
forge-tracker builds an in-memory inverted index at startup. Label queries use
AND semantics: a ticket must match all specified labels. The index provides
O(1) candidate lookup per label.

**What are the 3 edge types?**
`parent` -- hierarchical containment (plan contains tasks). `blocks` --
dependency (ticket A blocks ticket B). `relates-to` -- informational link.

**How do Plans relate to Tickets?**
A Plan is a Ticket with the `kind:plan` label. The Plan's task list comes from
`parent` edges in the graph. Creating a plan with tasks adds those edges.

**What is a MetaPlan?**
A MetaPlan coordinates work across repositories. It stores stages (with
repo-scoped plan references) and checkpoints (gate conditions) in a separate
`.metaplan.yaml` file alongside the ticket markdown.

**Is there authentication?**
No. forge-tracker runs as a local service. Add a reverse proxy for auth.

**What happens if the server crashes mid-write?**
All writes use atomic temp-file-plus-rename. A crash leaves either the old
file or the new file, never a partial write.

## Documentation

| Document | Audience |
|----------|----------|
| [DESIGN.md](DESIGN.md) | Developers -- architecture and technical design |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributors -- build, test, commit conventions |
| [api/forge-tracker.v1.yaml](api/forge-tracker.v1.yaml) | API consumers -- OpenAPI 3.0 spec |

## Contributing and License

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, test commands,
and commit conventions.

Licensed under the Apache License 2.0. See the license headers in source files.
