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

package controller

import (
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/alexandremahdhaoui/forge-tracker/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanService_Create(t *testing.T) {
	planStore := mockadapter.NewMockPlanStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewPlanService(planStore, graphStore)
	ctx := context.Background()

	plan := types.Plan{
		Ticket: types.Ticket{
			ID:    "plan-1",
			Title: "My plan",
		},
		Tasks: []string{"task-1", "task-2"},
	}

	planStore.EXPECT().Create(mock.Anything, "ts", plan).Return(nil)
	graphStore.EXPECT().AddEdge(mock.Anything, "ts", types.Edge{From: "plan-1", To: "task-1", Type: "parent"}).Return(nil)
	graphStore.EXPECT().AddEdge(mock.Anything, "ts", types.Edge{From: "plan-1", To: "task-2", Type: "parent"}).Return(nil)

	result, err := svc.Create(ctx, "ts", plan)
	require.NoError(t, err)
	assert.Equal(t, "plan-1", result.ID)
	assert.Equal(t, []string{"task-1", "task-2"}, result.Tasks)
}

func TestPlanService_Create_NoTasks(t *testing.T) {
	planStore := mockadapter.NewMockPlanStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewPlanService(planStore, graphStore)
	ctx := context.Background()

	plan := types.Plan{
		Ticket: types.Ticket{ID: "plan-1", Title: "Empty plan"},
	}

	planStore.EXPECT().Create(mock.Anything, "ts", plan).Return(nil)

	result, err := svc.Create(ctx, "ts", plan)
	require.NoError(t, err)
	assert.Equal(t, "plan-1", result.ID)
}
