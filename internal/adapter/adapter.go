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

package adapter

import (
	"context"
	"errors"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// ErrNotFound indicates the requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrValidation indicates a validation failure in the request.
var ErrValidation = errors.New("validation error")

// TicketStore provides CRUD operations for tickets within a TrackingSet.
type TicketStore interface {
	Create(ctx context.Context, trackingSet string, ticket types.Ticket) error
	Get(ctx context.Context, trackingSet string, id string) (types.Ticket, error)
	List(ctx context.Context, trackingSet string, filter TicketFilter) ([]types.Ticket, error)
	Update(ctx context.Context, trackingSet string, ticket types.Ticket) error
	Delete(ctx context.Context, trackingSet string, id string) error
	AddComment(ctx context.Context, trackingSet string, id string, comment types.Comment) error
}

// TicketFilter defines query parameters for listing tickets.
type TicketFilter struct {
	Status   string
	Assignee string
	Labels   []string
	Priority *int
}

// GraphStore provides CRUD operations for the ticket relationship graph.
type GraphStore interface {
	Get(ctx context.Context, trackingSet string) (types.Graph, error)
	AddNode(ctx context.Context, trackingSet string, ticketID string) error
	RemoveNode(ctx context.Context, trackingSet string, ticketID string) error
	AddEdge(ctx context.Context, trackingSet string, edge types.Edge) error
	RemoveEdge(ctx context.Context, trackingSet string, edge types.Edge) error
	GetEdges(ctx context.Context, trackingSet string, ticketID string) ([]types.Edge, error)
	GetChildren(ctx context.Context, trackingSet string, parentID string) ([]string, error)
	GetBlocking(ctx context.Context, trackingSet string, ticketID string) ([]string, error)
}

// TrackingSetStore provides CRUD operations for TrackingSets.
type TrackingSetStore interface {
	Create(ctx context.Context, ts types.TrackingSet) error
	Get(ctx context.Context, name string) (types.TrackingSet, error)
	List(ctx context.Context) ([]types.TrackingSet, error)
	Delete(ctx context.Context, name string) error
}

// PlanStore provides CRUD operations for Plan tickets.
type PlanStore interface {
	Create(ctx context.Context, trackingSet string, plan types.Plan) error
	Get(ctx context.Context, trackingSet string, id string) (types.Plan, error)
	List(ctx context.Context, trackingSet string) ([]types.Plan, error)
	Update(ctx context.Context, trackingSet string, plan types.Plan) error
	Delete(ctx context.Context, trackingSet string, id string) error
}

// MetaPlanStore provides CRUD operations for MetaPlan tickets.
type MetaPlanStore interface {
	Create(ctx context.Context, trackingSet string, mp types.MetaPlan) error
	Get(ctx context.Context, trackingSet string, id string) (types.MetaPlan, error)
	List(ctx context.Context, trackingSet string) ([]types.MetaPlan, error)
	Update(ctx context.Context, trackingSet string, mp types.MetaPlan) error
	Delete(ctx context.Context, trackingSet string, id string) error
}
