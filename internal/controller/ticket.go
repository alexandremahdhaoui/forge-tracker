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
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// TicketService defines business logic for ticket operations.
type TicketService interface {
	Create(ctx context.Context, trackingSet string, ticket types.Ticket) (types.Ticket, error)
	Get(ctx context.Context, trackingSet string, id string) (types.Ticket, error)
	List(ctx context.Context, trackingSet string, filter adapter.TicketFilter) ([]types.Ticket, error)
	Update(ctx context.Context, trackingSet string, ticket types.Ticket) (types.Ticket, error)
	Delete(ctx context.Context, trackingSet string, id string) error
	AddComment(ctx context.Context, trackingSet string, id string, comment types.Comment) (types.Comment, error)
}

var _ TicketService = (*ticketService)(nil)

type ticketService struct {
	tickets adapter.TicketStore
	graphs  adapter.GraphStore
}

// NewTicketService creates a TicketService with the given stores.
func NewTicketService(ts adapter.TicketStore, gs adapter.GraphStore) TicketService {
	return &ticketService{
		tickets: ts,
		graphs:  gs,
	}
}

func (s *ticketService) Create(ctx context.Context, trackingSet string, ticket types.Ticket) (types.Ticket, error) {
	if ticket.ID == "" {
		return types.Ticket{}, fmt.Errorf("ticket ID must not be empty: %w", adapter.ErrValidation)
	}
	if ticket.Title == "" {
		return types.Ticket{}, fmt.Errorf("ticket title must not be empty: %w", adapter.ErrValidation)
	}

	now := time.Now()
	if ticket.Created.IsZero() {
		ticket.Created = now
	}
	if ticket.Updated.IsZero() {
		ticket.Updated = now
	}
	if ticket.Status == "" {
		ticket.Status = "pending"
	}

	if err := s.tickets.Create(ctx, trackingSet, ticket); err != nil {
		return types.Ticket{}, fmt.Errorf("creating ticket: %w", err)
	}
	if err := s.graphs.AddNode(ctx, trackingSet, ticket.ID); err != nil {
		return types.Ticket{}, fmt.Errorf("adding graph node: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) Get(ctx context.Context, trackingSet string, id string) (types.Ticket, error) {
	return s.tickets.Get(ctx, trackingSet, id)
}

func (s *ticketService) List(ctx context.Context, trackingSet string, filter adapter.TicketFilter) ([]types.Ticket, error) {
	return s.tickets.List(ctx, trackingSet, filter)
}

func (s *ticketService) Update(ctx context.Context, trackingSet string, ticket types.Ticket) (types.Ticket, error) {
	if err := s.tickets.Update(ctx, trackingSet, ticket); err != nil {
		return types.Ticket{}, fmt.Errorf("updating ticket: %w", err)
	}
	return s.tickets.Get(ctx, trackingSet, ticket.ID)
}

func (s *ticketService) Delete(ctx context.Context, trackingSet string, id string) error {
	if err := s.graphs.RemoveNode(ctx, trackingSet, id); err != nil {
		return fmt.Errorf("removing graph node: %w", err)
	}
	if err := s.tickets.Delete(ctx, trackingSet, id); err != nil {
		return fmt.Errorf("deleting ticket: %w", err)
	}
	return nil
}

func (s *ticketService) AddComment(ctx context.Context, trackingSet string, id string, comment types.Comment) (types.Comment, error) {
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	if err := s.tickets.AddComment(ctx, trackingSet, id, comment); err != nil {
		return types.Comment{}, fmt.Errorf("adding comment: %w", err)
	}
	return comment, nil
}
