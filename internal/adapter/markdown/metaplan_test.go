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

func setupMetaPlanTest(t *testing.T) (adapter.MetaPlanStore, context.Context) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()

	require.NoError(t, store.TrackingSetStore().Create(ctx, types.TrackingSet{Name: "ts"}))
	return store.MetaPlanStore(), ctx
}

func TestMetaPlanStore_CreateGet_RoundTrip(t *testing.T) {
	mps, ctx := setupMetaPlanTest(t)

	mp := types.MetaPlan{
		Ticket: types.Ticket{
			ID:          "mp-1",
			Title:       "Cross-repo plan",
			Description: "Coordinates work.",
		},
		Stages: []types.MetaPlanStage{
			{
				Name:   "stage-1",
				Status: "pending",
				Repos: []types.StageRepoRef{
					{Name: "repo-a", PlanID: "plan-a-1", TasksTotal: 5, TasksDone: 2},
				},
			},
		},
		Checkpoints: []types.MetaCheckpoint{
			{Name: "gate-1", Stage: "stage-1", Condition: "all tasks done", Met: false},
		},
	}

	require.NoError(t, mps.Create(ctx, "ts", mp))

	got, err := mps.Get(ctx, "ts", "mp-1")
	require.NoError(t, err)
	assert.Equal(t, "mp-1", got.ID)
	assert.Equal(t, "Cross-repo plan", got.Title)
	assert.Equal(t, "Coordinates work.", got.Description)

	require.Len(t, got.Stages, 1)
	assert.Equal(t, "stage-1", got.Stages[0].Name)
	assert.Equal(t, "pending", got.Stages[0].Status)
	require.Len(t, got.Stages[0].Repos, 1)
	assert.Equal(t, "repo-a", got.Stages[0].Repos[0].Name)
	assert.Equal(t, 5, got.Stages[0].Repos[0].TasksTotal)
	assert.Equal(t, 2, got.Stages[0].Repos[0].TasksDone)

	require.Len(t, got.Checkpoints, 1)
	assert.Equal(t, "gate-1", got.Checkpoints[0].Name)
	assert.Equal(t, false, got.Checkpoints[0].Met)
}

func TestMetaPlanStore_List(t *testing.T) {
	mps, ctx := setupMetaPlanTest(t)

	require.NoError(t, mps.Create(ctx, "ts", types.MetaPlan{
		Ticket: types.Ticket{ID: "mp-1", Title: "Plan A"},
	}))
	require.NoError(t, mps.Create(ctx, "ts", types.MetaPlan{
		Ticket: types.Ticket{ID: "mp-2", Title: "Plan B"},
	}))

	list, err := mps.List(ctx, "ts")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestMetaPlanStore_Delete(t *testing.T) {
	mps, ctx := setupMetaPlanTest(t)

	require.NoError(t, mps.Create(ctx, "ts", types.MetaPlan{
		Ticket: types.Ticket{ID: "mp-1", Title: "To delete"},
	}))

	require.NoError(t, mps.Delete(ctx, "ts", "mp-1"))

	_, err := mps.Get(ctx, "ts", "mp-1")
	assert.Error(t, err)
}
