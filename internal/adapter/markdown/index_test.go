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

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelIndex_BuildIndex(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()

	require.NoError(t, store.TrackingSetStore().Create(ctx, types.TrackingSet{Name: "ts"}))
	ts := store.TicketStore()
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-1", Title: "A", Labels: []string{"x", "y"}}))
	require.NoError(t, ts.Create(ctx, "ts", types.Ticket{ID: "t-2", Title: "B", Labels: []string{"y", "z"}}))

	// Build a fresh index from disk.
	idx := newLabelIndex()
	require.NoError(t, idx.BuildIndex(dir))

	result := idx.MatchLabels("ts", []string{"y"})
	assert.Len(t, result, 2)
	assert.Contains(t, result, "t-1")
	assert.Contains(t, result, "t-2")
}

func TestLabelIndex_MatchLabels_SingleLabel(t *testing.T) {
	idx := newLabelIndex()
	idx.Add("ts", "t-1", []string{"a", "b"})
	idx.Add("ts", "t-2", []string{"a"})

	result := idx.MatchLabels("ts", []string{"b"})
	assert.Len(t, result, 1)
	assert.Contains(t, result, "t-1")
}

func TestLabelIndex_MatchLabels_MultipleLabels_AND(t *testing.T) {
	idx := newLabelIndex()
	idx.Add("ts", "t-1", []string{"a", "b"})
	idx.Add("ts", "t-2", []string{"a"})
	idx.Add("ts", "t-3", []string{"b"})

	result := idx.MatchLabels("ts", []string{"a", "b"})
	assert.Len(t, result, 1)
	assert.Contains(t, result, "t-1")
}

func TestLabelIndex_MatchLabels_NonExistentLabel(t *testing.T) {
	idx := newLabelIndex()
	idx.Add("ts", "t-1", []string{"a"})

	result := idx.MatchLabels("ts", []string{"missing"})
	assert.Empty(t, result)
}

func TestLabelIndex_MatchLabels_Empty(t *testing.T) {
	idx := newLabelIndex()

	result := idx.MatchLabels("ts", []string{})
	assert.Nil(t, result, "empty labels should return nil (no filtering)")
}

func TestLabelIndex_AddRemove(t *testing.T) {
	idx := newLabelIndex()
	idx.Add("ts", "t-1", []string{"a", "b"})

	result := idx.MatchLabels("ts", []string{"a"})
	assert.Len(t, result, 1)

	idx.Remove("ts", "t-1", []string{"a"})

	result = idx.MatchLabels("ts", []string{"a"})
	assert.Empty(t, result)

	// Label "b" should still be present.
	result = idx.MatchLabels("ts", []string{"b"})
	assert.Len(t, result, 1)
}

func TestLabelIndex_Update(t *testing.T) {
	idx := newLabelIndex()
	idx.Add("ts", "t-1", []string{"a", "b"})

	idx.Update("ts", "t-1", []string{"a", "b"}, []string{"b", "c"})

	// "a" removed, "c" added, "b" unchanged.
	assert.Empty(t, idx.MatchLabels("ts", []string{"a"}))

	result := idx.MatchLabels("ts", []string{"b"})
	assert.Len(t, result, 1)
	assert.Contains(t, result, "t-1")

	result = idx.MatchLabels("ts", []string{"c"})
	assert.Len(t, result, 1)
	assert.Contains(t, result, "t-1")
}
