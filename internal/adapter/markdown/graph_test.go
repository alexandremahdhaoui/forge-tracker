//go:build unit

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package markdown

import (
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

func setupGraphTest(t *testing.T) (adapter.GraphStore, context.Context) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()

	if err := store.TrackingSetStore().Create(ctx, types.TrackingSet{Name: "test-ts"}); err != nil {
		t.Fatalf("Create tracking set: %v", err)
	}
	return store.GraphStore(), ctx
}

// mustAddNode is a test helper that adds a node and fails on error.
func mustAddNode(t *testing.T, gs adapter.GraphStore, ctx context.Context, ts, id string) {
	t.Helper()
	if err := gs.AddNode(ctx, ts, id); err != nil {
		t.Fatalf("AddNode(%q): %v", id, err)
	}
}

// mustAddEdge is a test helper that adds an edge and fails on error.
func mustAddEdge(t *testing.T, gs adapter.GraphStore, ctx context.Context, ts string, edge types.Edge) {
	t.Helper()
	if err := gs.AddEdge(ctx, ts, edge); err != nil {
		t.Fatalf("AddEdge(%+v): %v", edge, err)
	}
}

func TestGraphStore_AddNode(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")

	graph, err := gs.Get(ctx, "test-ts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(graph.Nodes) != 1 || graph.Nodes[0] != "task-001" {
		t.Errorf("Nodes = %v, want [task-001]", graph.Nodes)
	}
}

func TestGraphStore_AddNode_Idempotent(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")
	mustAddNode(t, gs, ctx, "test-ts", "task-001") // duplicate

	graph, err := gs.Get(ctx, "test-ts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Errorf("Nodes len = %d, want 1 (idempotent)", len(graph.Nodes))
	}
}

func TestGraphStore_RemoveNode(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")
	mustAddNode(t, gs, ctx, "test-ts", "task-002")
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "task-001", To: "task-002", Type: "blocks"})

	if err := gs.RemoveNode(ctx, "test-ts", "task-001"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}

	graph, err := gs.Get(ctx, "test-ts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Errorf("Nodes len = %d, want 1", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("Edges len = %d, want 0 (edges should be removed)", len(graph.Edges))
	}
}

func TestGraphStore_AddEdge(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")
	mustAddNode(t, gs, ctx, "test-ts", "task-002")

	edge := types.Edge{From: "task-001", To: "task-002", Type: "blocks"}
	mustAddEdge(t, gs, ctx, "test-ts", edge)

	graph, err := gs.Get(ctx, "test-ts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("Edges len = %d, want 1", len(graph.Edges))
	}
	if graph.Edges[0] != edge {
		t.Errorf("Edge = %+v, want %+v", graph.Edges[0], edge)
	}
}

func TestGraphStore_AddEdge_InvalidNode(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")

	err := gs.AddEdge(ctx, "test-ts", types.Edge{From: "task-001", To: "missing", Type: "blocks"})
	if err == nil {
		t.Fatal("expected error for missing To node")
	}

	err = gs.AddEdge(ctx, "test-ts", types.Edge{From: "missing", To: "task-001", Type: "blocks"})
	if err == nil {
		t.Fatal("expected error for missing From node")
	}
}

func TestGraphStore_RemoveEdge(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "task-001")
	mustAddNode(t, gs, ctx, "test-ts", "task-002")

	edge := types.Edge{From: "task-001", To: "task-002", Type: "blocks"}
	mustAddEdge(t, gs, ctx, "test-ts", edge)

	if err := gs.RemoveEdge(ctx, "test-ts", edge); err != nil {
		t.Fatalf("RemoveEdge: %v", err)
	}

	graph, err := gs.Get(ctx, "test-ts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(graph.Edges) != 0 {
		t.Errorf("Edges len = %d, want 0", len(graph.Edges))
	}
}

func TestGraphStore_GetEdges(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "a")
	mustAddNode(t, gs, ctx, "test-ts", "b")
	mustAddNode(t, gs, ctx, "test-ts", "c")
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "a", To: "b", Type: "blocks"})
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "b", To: "c", Type: "parent"})

	edges, err := gs.GetEdges(ctx, "test-ts", "b")
	if err != nil {
		t.Fatalf("GetEdges: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("GetEdges len = %d, want 2", len(edges))
	}
}

func TestGraphStore_GetChildren(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "parent")
	mustAddNode(t, gs, ctx, "test-ts", "child-1")
	mustAddNode(t, gs, ctx, "test-ts", "child-2")
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "parent", To: "child-1", Type: "parent"})
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "parent", To: "child-2", Type: "parent"})
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "parent", To: "child-1", Type: "blocks"}) // not parent type

	children, err := gs.GetChildren(ctx, "test-ts", "parent")
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("GetChildren len = %d, want 2", len(children))
	}
}

func TestGraphStore_GetBlocking(t *testing.T) {
	gs, ctx := setupGraphTest(t)

	mustAddNode(t, gs, ctx, "test-ts", "blocker-1")
	mustAddNode(t, gs, ctx, "test-ts", "blocker-2")
	mustAddNode(t, gs, ctx, "test-ts", "blocked")
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "blocker-1", To: "blocked", Type: "blocks"})
	mustAddEdge(t, gs, ctx, "test-ts", types.Edge{From: "blocker-2", To: "blocked", Type: "blocks"})

	blocking, err := gs.GetBlocking(ctx, "test-ts", "blocked")
	if err != nil {
		t.Fatalf("GetBlocking: %v", err)
	}
	if len(blocking) != 2 {
		t.Errorf("GetBlocking len = %d, want 2", len(blocking))
	}
}
