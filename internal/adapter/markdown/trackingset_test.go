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
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

func TestTrackingSetStore_Create(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	tss := store.TrackingSetStore()

	ts := types.TrackingSet{Name: "my-project"}
	if err := tss.Create(ctx, ts); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify directory structure.
	ticketsDir := filepath.Join(dir, "tracking-sets", "my-project", "tickets")
	if _, err := os.Stat(ticketsDir); err != nil {
		t.Errorf("tickets directory not created: %v", err)
	}

	// Verify file exists and is readable.
	got, err := tss.Get(ctx, "my-project")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "my-project" {
		t.Errorf("Name = %q, want %q", got.Name, "my-project")
	}
	if got.Graph.Nodes == nil {
		t.Error("Graph.Nodes should be initialized, not nil")
	}
}

func TestTrackingSetStore_List(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	tss := store.TrackingSetStore()

	if err := tss.Create(ctx, types.TrackingSet{Name: "alpha"}); err != nil {
		t.Fatalf("Create alpha: %v", err)
	}
	if err := tss.Create(ctx, types.TrackingSet{Name: "beta"}); err != nil {
		t.Fatalf("Create beta: %v", err)
	}

	list, err := tss.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List len = %d, want 2", len(list))
	}
}

func TestTrackingSetStore_List_Empty(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	tss := store.TrackingSetStore()

	list, err := tss.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list != nil {
		t.Errorf("List = %v, want nil for empty", list)
	}
}

func TestTrackingSetStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	tss := store.TrackingSetStore()

	if err := tss.Create(ctx, types.TrackingSet{Name: "to-delete"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := tss.Delete(ctx, "to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	tsDir := filepath.Join(dir, "tracking-sets", "to-delete")
	if _, err := os.Stat(tsDir); !os.IsNotExist(err) {
		t.Errorf("tracking set directory should be deleted")
	}
}

func TestTrackingSetStore_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	tss := store.TrackingSetStore()

	_, err := tss.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tracking set")
	}
}
