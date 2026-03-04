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
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// Compile-time interface check.
var _ adapter.TrackingSetStore = (*trackingSetStore)(nil)

// trackingSetStore implements adapter.TrackingSetStore.
type trackingSetStore struct {
	*markdownStore
}

// --- TrackingSetStore implementation ---

func (s *trackingSetStore) Create(_ context.Context, ts types.TrackingSet) error {
	dir := filepath.Join(s.basePath, "tracking-sets", ts.Name, "tickets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create tracking set directory: %w", err)
	}

	// Initialize empty graph if zero-value.
	if ts.Graph.Nodes == nil {
		ts.Graph.Nodes = []string{}
	}
	if ts.Graph.Edges == nil {
		ts.Graph.Edges = []types.Edge{}
	}

	path := trackingSetPath(s.basePath, ts.Name)
	return writeTrackingSet(path, ts)
}

func (s *trackingSetStore) Get(_ context.Context, name string) (types.TrackingSet, error) {
	return readTrackingSet(trackingSetPath(s.basePath, name))
}

func (s *trackingSetStore) List(_ context.Context) ([]types.TrackingSet, error) {
	dir := filepath.Join(s.basePath, "tracking-sets")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tracking-sets directory: %w", err)
	}

	var result []types.TrackingSet
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ts, err := readTrackingSet(trackingSetPath(s.basePath, entry.Name()))
		if err != nil {
			return nil, err
		}
		result = append(result, ts)
	}
	return result, nil
}

func (s *trackingSetStore) Delete(_ context.Context, name string) error {
	dir := filepath.Join(s.basePath, "tracking-sets", name)
	return os.RemoveAll(dir)
}
