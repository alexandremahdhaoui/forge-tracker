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

func TestGraphService_AddEdge_Valid(t *testing.T) {
	graphStore := mockadapter.NewMockGraphStore(t)
	ticketStore := mockadapter.NewMockTicketStore(t)
	svc := NewGraphService(graphStore, ticketStore)
	ctx := context.Background()

	edge := types.Edge{From: "t-1", To: "t-2", Type: "blocks"}
	graphStore.EXPECT().AddEdge(mock.Anything, "ts", edge).Return(nil)

	result, err := svc.AddEdge(ctx, "ts", edge)
	require.NoError(t, err)
	assert.Equal(t, edge, result)
}

func TestGraphService_AddEdge_InvalidType(t *testing.T) {
	graphStore := mockadapter.NewMockGraphStore(t)
	ticketStore := mockadapter.NewMockTicketStore(t)
	svc := NewGraphService(graphStore, ticketStore)
	ctx := context.Background()

	edge := types.Edge{From: "t-1", To: "t-2", Type: "invalid"}
	_, err := svc.AddEdge(ctx, "ts", edge)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid edge type")
}

func TestGraphService_AddEdge_AllValidTypes(t *testing.T) {
	for _, edgeType := range []string{"parent", "blocks", "relates-to"} {
		t.Run(edgeType, func(t *testing.T) {
			graphStore := mockadapter.NewMockGraphStore(t)
			ticketStore := mockadapter.NewMockTicketStore(t)
			svc := NewGraphService(graphStore, ticketStore)
			ctx := context.Background()

			edge := types.Edge{From: "a", To: "b", Type: edgeType}
			graphStore.EXPECT().AddEdge(mock.Anything, "ts", edge).Return(nil)

			_, err := svc.AddEdge(ctx, "ts", edge)
			require.NoError(t, err)
		})
	}
}

func TestGraphService_GetChildren(t *testing.T) {
	graphStore := mockadapter.NewMockGraphStore(t)
	ticketStore := mockadapter.NewMockTicketStore(t)
	svc := NewGraphService(graphStore, ticketStore)
	ctx := context.Background()

	graphStore.EXPECT().GetChildren(mock.Anything, "ts", "parent").Return([]string{"c-1", "c-2"}, nil)
	ticketStore.EXPECT().Get(mock.Anything, "ts", "c-1").Return(types.Ticket{ID: "c-1", Title: "Child 1"}, nil)
	ticketStore.EXPECT().Get(mock.Anything, "ts", "c-2").Return(types.Ticket{ID: "c-2", Title: "Child 2"}, nil)

	result, err := svc.GetChildren(ctx, "ts", "parent")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "c-1", result[0].ID)
	assert.Equal(t, "c-2", result[1].ID)
}

func TestGraphService_GetBlocking(t *testing.T) {
	graphStore := mockadapter.NewMockGraphStore(t)
	ticketStore := mockadapter.NewMockTicketStore(t)
	svc := NewGraphService(graphStore, ticketStore)
	ctx := context.Background()

	graphStore.EXPECT().GetBlocking(mock.Anything, "ts", "blocked").Return([]string{"b-1"}, nil)
	ticketStore.EXPECT().Get(mock.Anything, "ts", "b-1").Return(types.Ticket{ID: "b-1", Title: "Blocker"}, nil)

	result, err := svc.GetBlocking(ctx, "ts", "blocked")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "b-1", result[0].ID)
}
