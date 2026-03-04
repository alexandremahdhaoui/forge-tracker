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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPlanTest(t *testing.T) (adapter.PlanStore, adapter.GraphStore, context.Context) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()

	require.NoError(t, store.TrackingSetStore().Create(ctx, types.TrackingSet{Name: "ts"}))
	return store.PlanStore(), store.GraphStore(), ctx
}

func TestPlanStore_CreateGet_RoundTrip(t *testing.T) {
	ps, gs, ctx := setupPlanTest(t)

	plan := types.Plan{
		Ticket: types.Ticket{
			ID:          "plan-1",
			Title:       "My plan",
			Labels:      []string{"kind:plan"},
			Description: "Plan description.",
		},
	}

	require.NoError(t, ps.Create(ctx, "ts", plan))

	// Add graph nodes so GetChildren works.
	require.NoError(t, gs.AddNode(ctx, "ts", "plan-1"))
	require.NoError(t, gs.AddNode(ctx, "ts", "task-1"))
	require.NoError(t, gs.AddEdge(ctx, "ts", types.Edge{From: "plan-1", To: "task-1", Type: "parent"}))

	got, err := ps.Get(ctx, "ts", "plan-1")
	require.NoError(t, err)
	assert.Equal(t, "plan-1", got.ID)
	assert.Equal(t, "My plan", got.Title)
	assert.Equal(t, "Plan description.", got.Description)
	assert.Equal(t, []string{"task-1"}, got.Tasks)
}

func TestPlanStore_Get_PopulatesTasks(t *testing.T) {
	ps, gs, ctx := setupPlanTest(t)

	plan := types.Plan{
		Ticket: types.Ticket{
			ID:     "plan-1",
			Title:  "Plan",
			Labels: []string{"kind:plan"},
		},
	}
	require.NoError(t, ps.Create(ctx, "ts", plan))
	require.NoError(t, gs.AddNode(ctx, "ts", "plan-1"))
	require.NoError(t, gs.AddNode(ctx, "ts", "child-a"))
	require.NoError(t, gs.AddNode(ctx, "ts", "child-b"))
	require.NoError(t, gs.AddEdge(ctx, "ts", types.Edge{From: "plan-1", To: "child-a", Type: "parent"}))
	require.NoError(t, gs.AddEdge(ctx, "ts", types.Edge{From: "plan-1", To: "child-b", Type: "parent"}))

	got, err := ps.Get(ctx, "ts", "plan-1")
	require.NoError(t, err)
	assert.Len(t, got.Tasks, 2)
	assert.Contains(t, got.Tasks, "child-a")
	assert.Contains(t, got.Tasks, "child-b")
}

func TestPlanStore_List_FiltersByKindPlan(t *testing.T) {
	ps, gs, ctx := setupPlanTest(t)

	// Create a plan ticket with kind:plan label.
	planTicket := types.Plan{
		Ticket: types.Ticket{
			ID:     "plan-1",
			Title:  "A plan",
			Labels: []string{"kind:plan"},
		},
	}
	require.NoError(t, ps.Create(ctx, "ts", planTicket))
	require.NoError(t, gs.AddNode(ctx, "ts", "plan-1"))

	// Create a regular ticket (no kind:plan label) via the ticket store path.
	dir := t.TempDir()
	store2 := NewStore(dir)
	ctx2 := context.Background()
	require.NoError(t, store2.TrackingSetStore().Create(ctx2, types.TrackingSet{Name: "ts2"}))
	ts2 := store2.TicketStore()
	require.NoError(t, ts2.Create(ctx2, "ts2", types.Ticket{ID: "not-plan", Title: "Regular", Labels: []string{"kind:task"}}))

	// List plans for the tracking set with the plan.
	plans, err := ps.List(ctx, "ts")
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "plan-1", plans[0].ID)
}
