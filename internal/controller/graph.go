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
	"fmt"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// GraphService defines business logic for graph edge operations.
type GraphService interface {
	GetGraph(ctx context.Context, trackingSet string) (types.Graph, error)
	AddEdge(ctx context.Context, trackingSet string, edge types.Edge) (types.Edge, error)
	RemoveEdge(ctx context.Context, trackingSet string, edge types.Edge) error
	ListEdges(ctx context.Context, trackingSet string, ticketID string) ([]types.Edge, error)
	GetChildren(ctx context.Context, trackingSet string, parentID string) ([]types.Ticket, error)
	GetBlocking(ctx context.Context, trackingSet string, ticketID string) ([]types.Ticket, error)
}

var _ GraphService = (*graphService)(nil)

type graphService struct {
	graphs  adapter.GraphStore
	tickets adapter.TicketStore
}

// NewGraphService creates a GraphService with the given stores.
func NewGraphService(gs adapter.GraphStore, ts adapter.TicketStore) GraphService {
	return &graphService{
		graphs:  gs,
		tickets: ts,
	}
}

var validEdgeTypes = map[string]struct{}{
	"parent":     {},
	"blocks":     {},
	"relates-to": {},
}

func (s *graphService) GetGraph(ctx context.Context, trackingSet string) (types.Graph, error) {
	return s.graphs.Get(ctx, trackingSet)
}

func (s *graphService) AddEdge(ctx context.Context, trackingSet string, edge types.Edge) (types.Edge, error) {
	if _, ok := validEdgeTypes[edge.Type]; !ok {
		return types.Edge{}, fmt.Errorf("invalid edge type %q: must be one of parent, blocks, relates-to: %w", edge.Type, adapter.ErrValidation)
	}
	if err := s.graphs.AddEdge(ctx, trackingSet, edge); err != nil {
		return types.Edge{}, fmt.Errorf("adding edge: %w", err)
	}
	return edge, nil
}

func (s *graphService) RemoveEdge(ctx context.Context, trackingSet string, edge types.Edge) error {
	return s.graphs.RemoveEdge(ctx, trackingSet, edge)
}

func (s *graphService) ListEdges(ctx context.Context, trackingSet string, ticketID string) ([]types.Edge, error) {
	return s.graphs.GetEdges(ctx, trackingSet, ticketID)
}

func (s *graphService) GetChildren(ctx context.Context, trackingSet string, parentID string) ([]types.Ticket, error) {
	ids, err := s.graphs.GetChildren(ctx, trackingSet, parentID)
	if err != nil {
		return nil, fmt.Errorf("getting children IDs: %w", err)
	}
	return s.loadTickets(ctx, trackingSet, ids)
}

func (s *graphService) GetBlocking(ctx context.Context, trackingSet string, ticketID string) ([]types.Ticket, error) {
	ids, err := s.graphs.GetBlocking(ctx, trackingSet, ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting blocking IDs: %w", err)
	}
	return s.loadTickets(ctx, trackingSet, ids)
}

func (s *graphService) loadTickets(ctx context.Context, trackingSet string, ids []string) ([]types.Ticket, error) {
	tickets := make([]types.Ticket, 0, len(ids))
	for _, id := range ids {
		t, err := s.tickets.Get(ctx, trackingSet, id)
		if err != nil {
			return nil, fmt.Errorf("loading ticket %q: %w", id, err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}
