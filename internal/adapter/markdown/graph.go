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
	"gopkg.in/yaml.v3"
)

// Compile-time interface check.
var _ adapter.GraphStore = (*graphStore)(nil)

// markdownStore holds the shared base path for all markdown-backed stores.
type markdownStore struct {
	basePath string
}

// graphStore implements adapter.GraphStore.
type graphStore struct {
	*markdownStore
}

// Store groups all markdown-backed store implementations.
type Store struct {
	store *markdownStore
	index *labelIndex
}

// NewStore returns a Store rooted at basePath. Use GraphStore(),
// TrackingSetStore(), TicketStore(), PlanStore(), and MetaPlanStore()
// to obtain the individual interface implementations.
func NewStore(basePath string) *Store {
	return &Store{
		store: &markdownStore{basePath: basePath},
		index: newLabelIndex(),
	}
}

// GraphStore returns the adapter.GraphStore implementation.
func (s *Store) GraphStore() adapter.GraphStore {
	return &graphStore{s.store}
}

// TrackingSetStore returns the adapter.TrackingSetStore implementation.
func (s *Store) TrackingSetStore() adapter.TrackingSetStore {
	return &trackingSetStore{s.store}
}

// TicketStore returns the adapter.TicketStore implementation.
func (s *Store) TicketStore() adapter.TicketStore {
	return &ticketStore{markdownStore: s.store, index: s.index}
}

// PlanStore returns the adapter.PlanStore implementation.
func (s *Store) PlanStore() adapter.PlanStore {
	return &planStore{markdownStore: s.store, graph: s.GraphStore()}
}

// MetaPlanStore returns the adapter.MetaPlanStore implementation.
func (s *Store) MetaPlanStore() adapter.MetaPlanStore {
	return &metaPlanStore{markdownStore: s.store}
}

// BuildIndex scans all tracking sets and populates the label index.
func (s *Store) BuildIndex() error {
	return s.index.BuildIndex(s.store.basePath)
}

// --- GraphStore implementation ---

func (s *graphStore) Get(_ context.Context, trackingSet string) (types.Graph, error) {
	ts, err := readTrackingSet(trackingSetPath(s.basePath, trackingSet))
	if err != nil {
		return types.Graph{}, err
	}
	return ts.Graph, nil
}

func (s *graphStore) AddNode(_ context.Context, trackingSet string, ticketID string) error {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return err
	}

	for _, n := range ts.Graph.Nodes {
		if n == ticketID {
			return nil // already exists
		}
	}

	ts.Graph.Nodes = append(ts.Graph.Nodes, ticketID)
	return writeTrackingSet(path, ts)
}

func (s *graphStore) RemoveNode(_ context.Context, trackingSet string, ticketID string) error {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return err
	}

	// Remove from nodes.
	nodes := make([]string, 0, len(ts.Graph.Nodes))
	for _, n := range ts.Graph.Nodes {
		if n != ticketID {
			nodes = append(nodes, n)
		}
	}
	ts.Graph.Nodes = nodes

	// Remove edges referencing this node.
	edges := make([]types.Edge, 0, len(ts.Graph.Edges))
	for _, e := range ts.Graph.Edges {
		if e.From != ticketID && e.To != ticketID {
			edges = append(edges, e)
		}
	}
	ts.Graph.Edges = edges

	return writeTrackingSet(path, ts)
}

func (s *graphStore) AddEdge(_ context.Context, trackingSet string, edge types.Edge) error {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return err
	}

	// Validate both endpoints exist.
	fromExists, toExists := false, false
	for _, n := range ts.Graph.Nodes {
		if n == edge.From {
			fromExists = true
		}
		if n == edge.To {
			toExists = true
		}
	}
	if !fromExists {
		return fmt.Errorf("node %q not found in graph", edge.From)
	}
	if !toExists {
		return fmt.Errorf("node %q not found in graph", edge.To)
	}

	ts.Graph.Edges = append(ts.Graph.Edges, edge)
	return writeTrackingSet(path, ts)
}

func (s *graphStore) RemoveEdge(_ context.Context, trackingSet string, edge types.Edge) error {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return err
	}

	edges := make([]types.Edge, 0, len(ts.Graph.Edges))
	for _, e := range ts.Graph.Edges {
		if e.From != edge.From || e.To != edge.To || e.Type != edge.Type {
			edges = append(edges, e)
		}
	}
	ts.Graph.Edges = edges

	return writeTrackingSet(path, ts)
}

func (s *graphStore) GetEdges(_ context.Context, trackingSet string, ticketID string) ([]types.Edge, error) {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return nil, err
	}

	var result []types.Edge
	for _, e := range ts.Graph.Edges {
		if e.From == ticketID || e.To == ticketID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *graphStore) GetChildren(_ context.Context, trackingSet string, parentID string) ([]string, error) {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return nil, err
	}

	var children []string
	for _, e := range ts.Graph.Edges {
		if e.From == parentID && e.Type == "parent" {
			children = append(children, e.To)
		}
	}
	return children, nil
}

func (s *graphStore) GetBlocking(_ context.Context, trackingSet string, ticketID string) ([]string, error) {
	path := trackingSetPath(s.basePath, trackingSet)
	ts, err := readTrackingSet(path)
	if err != nil {
		return nil, err
	}

	var blocking []string
	for _, e := range ts.Graph.Edges {
		if e.To == ticketID && e.Type == "blocks" {
			blocking = append(blocking, e.From)
		}
	}
	return blocking, nil
}

// --- Helper functions ---

func trackingSetPath(basePath, tsName string) string {
	return filepath.Join(basePath, "tracking-sets", tsName, "tracking-set.yaml")
}

func readTrackingSet(path string) (types.TrackingSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.TrackingSet{}, fmt.Errorf("tracking set: %w", adapter.ErrNotFound)
		}
		return types.TrackingSet{}, fmt.Errorf("read tracking set: %w", err)
	}

	var ts types.TrackingSet
	if err := yaml.Unmarshal(data, &ts); err != nil {
		return types.TrackingSet{}, fmt.Errorf("unmarshal tracking set: %w", err)
	}
	return ts, nil
}

func writeTrackingSet(path string, ts types.TrackingSet) error {
	data, err := yaml.Marshal(ts)
	if err != nil {
		return fmt.Errorf("marshal tracking set: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "tracking-set-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
