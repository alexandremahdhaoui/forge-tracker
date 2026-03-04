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
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/alexandremahdhaoui/forge-tracker/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTicketService_Create(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	ticketStore.EXPECT().Create(mock.Anything, "ts", mock.AnythingOfType("types.Ticket")).Return(nil)
	graphStore.EXPECT().AddNode(mock.Anything, "ts", "t-1").Return(nil)

	result, err := svc.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "Test"})
	require.NoError(t, err)
	assert.Equal(t, "t-1", result.ID)
	assert.Equal(t, "Test", result.Title)
	assert.False(t, result.Created.IsZero(), "Created should be set")
	assert.False(t, result.Updated.IsZero(), "Updated should be set")
	assert.Equal(t, "pending", result.Status, "default status should be pending")
}

func TestTicketService_Create_EmptyID(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	_, err := svc.Create(ctx, "ts", types.Ticket{Title: "No ID"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID must not be empty")
}

func TestTicketService_Create_EmptyTitle(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	_, err := svc.Create(ctx, "ts", types.Ticket{ID: "t-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title must not be empty")
}

func TestTicketService_Delete(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	graphStore.EXPECT().RemoveNode(mock.Anything, "ts", "t-1").Return(nil)
	ticketStore.EXPECT().Delete(mock.Anything, "ts", "t-1").Return(nil)

	err := svc.Delete(ctx, "ts", "t-1")
	require.NoError(t, err)
}

func TestTicketService_AddComment_SetsTimestamp(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	ticketStore.EXPECT().AddComment(mock.Anything, "ts", "t-1", mock.AnythingOfType("types.Comment")).Return(nil)

	before := time.Now()
	result, err := svc.AddComment(ctx, "ts", "t-1", types.Comment{Author: "user", Text: "Hello"})
	require.NoError(t, err)
	assert.Equal(t, "user", result.Author)
	assert.True(t, !result.Timestamp.Before(before), "Timestamp should be set")
}

func TestTicketService_Get(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	expected := types.Ticket{ID: "t-1", Title: "Test"}
	ticketStore.EXPECT().Get(mock.Anything, "ts", "t-1").Return(expected, nil)

	result, err := svc.Get(ctx, "ts", "t-1")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTicketService_List(t *testing.T) {
	ticketStore := mockadapter.NewMockTicketStore(t)
	graphStore := mockadapter.NewMockGraphStore(t)
	svc := NewTicketService(ticketStore, graphStore)
	ctx := context.Background()

	expected := []types.Ticket{{ID: "t-1"}, {ID: "t-2"}}
	filter := adapter.TicketFilter{Status: "open"}
	ticketStore.EXPECT().List(mock.Anything, "ts", filter).Return(expected, nil)

	result, err := svc.List(ctx, "ts", filter)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
