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
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketTest(t *testing.T) (adapter.TicketStore, context.Context) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()

	err := store.TrackingSetStore().Create(ctx, types.TrackingSet{Name: "ts"})
	require.NoError(t, err)
	return store.TicketStore(), ctx
}

func TestTicketStore_CreateGet_RoundTrip(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	ticket := types.Ticket{
		ID:          "t-1",
		Title:       "First ticket",
		Status:      "open",
		Priority:    2,
		Labels:      []string{"kind:task"},
		Assignee:    "user-1",
		Description: "A description.",
	}

	err := ts.Create(ctx, "ts", ticket)
	require.NoError(t, err)

	got, err := ts.Get(ctx, "ts", "t-1")
	require.NoError(t, err)

	assert.Equal(t, "t-1", got.ID)
	assert.Equal(t, "First ticket", got.Title)
	assert.Equal(t, "open", got.Status)
	assert.Equal(t, 2, got.Priority)
	assert.Equal(t, []string{"kind:task"}, got.Labels)
	assert.Equal(t, "user-1", got.Assignee)
	assert.Equal(t, "A description.", got.Description)
}

func TestTicketStore_Create_EmptyID(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	err := ts.Create(ctx, "ts", types.Ticket{Title: "No ID"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID must not be empty")
}

func TestTicketStore_Create_EmptyTitle(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	err := ts.Create(ctx, "ts", types.Ticket{ID: "t-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title must not be empty")
}

func TestTicketStore_Create_SetsTimestamps(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	before := time.Now()
	err := ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "Test"})
	require.NoError(t, err)
	after := time.Now()

	got, err := ts.Get(ctx, "ts", "t-1")
	require.NoError(t, err)

	assert.False(t, got.Created.IsZero(), "Created should be set")
	assert.False(t, got.Updated.IsZero(), "Updated should be set")
	assert.True(t, !got.Created.Before(before) && !got.Created.After(after),
		"Created should be between before and after")
}

func TestTicketStore_List_NoFilter(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A"}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B"}))

	list, err := ts.List(ctx, "ts", adapter.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestTicketStore_List_StatusFilter(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A", Status: "open"}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B", Status: "closed"}))

	list, err := ts.List(ctx, "ts", adapter.TicketFilter{Status: "open"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "t-1", list[0].ID)
}

func TestTicketStore_List_AssigneeFilter(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A", Assignee: "alice"}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B", Assignee: "bob"}))

	list, err := ts.List(ctx, "ts", adapter.TicketFilter{Assignee: "alice"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "t-1", list[0].ID)
}

func TestTicketStore_List_LabelFilter_AND(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A", Labels: []string{"a", "b"}}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B", Labels: []string{"a"}}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-3", Title: "C", Labels: []string{"b"}}))

	list, err := ts.List(ctx, "ts", adapter.TicketFilter{Labels: []string{"a", "b"}})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "t-1", list[0].ID)
}

func TestTicketStore_List_PriorityFilter(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A", Priority: 1}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B", Priority: 2}))

	prio := 1
	list, err := ts.List(ctx, "ts", adapter.TicketFilter{Priority: &prio})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "t-1", list[0].ID)
}

func TestTicketStore_Update_PreservesCreated(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{
		ID: "t-1", Title: "Original", Created: created, Updated: created,
	}))

	err := ts.Update(ctx, "ts", types.Ticket{
		ID: "t-1", Title: "Updated", Status: "closed",
	})
	require.NoError(t, err)

	got, err := ts.Get(ctx, "ts", "t-1")
	require.NoError(t, err)
	assert.Equal(t, created, got.Created, "Created timestamp must be preserved")
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, "closed", got.Status)
}

func TestTicketStore_Delete(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "Delete me"}))

	err := ts.Delete(ctx, "ts", "t-1")
	require.NoError(t, err)

	_, err = ts.Get(ctx, "ts", "t-1")
	assert.Error(t, err)
}

func TestTicketStore_AddComment(t *testing.T) {
	ts, ctx := setupTicketTest(t)

	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "Ticket"}))

	comment := types.Comment{
		Timestamp: time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
		Author:    "user",
		Text:      "A comment.",
	}
	err := ts.AddComment(ctx, "ts", "t-1", comment)
	require.NoError(t, err)

	got, err := ts.Get(ctx, "ts", "t-1")
	require.NoError(t, err)
	require.Len(t, got.Comments, 1)
	assert.Equal(t, "user", got.Comments[0].Author)
	assert.Equal(t, "A comment.", got.Comments[0].Text)
}
