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
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// Compile-time interface check.
var _ adapter.PlanStore = (*planStore)(nil)

// planStore implements adapter.PlanStore using markdown files and the graph.
type planStore struct {
	*markdownStore
	graph adapter.GraphStore
}

func (s *planStore) Create(_ context.Context, trackingSet string, plan types.Plan) error {
	now := time.Now()
	if plan.Created.IsZero() {
		plan.Created = now
	}
	if plan.Updated.IsZero() {
		plan.Updated = now
	}

	data, err := WriteTicket(plan.Ticket)
	if err != nil {
		return fmt.Errorf("write plan ticket: %w", err)
	}

	path := ticketPath(s.basePath, trackingSet, plan.ID)
	return atomicWrite(path, data)
}

func (s *planStore) Get(ctx context.Context, trackingSet string, id string) (types.Plan, error) {
	path := ticketPath(s.basePath, trackingSet, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.Plan{}, fmt.Errorf("plan ticket %q: %w", id, adapter.ErrNotFound)
		}
		return types.Plan{}, fmt.Errorf("read plan ticket %q: %w", id, err)
	}

	ticket, err := ParseTicket(data)
	if err != nil {
		return types.Plan{}, fmt.Errorf("parse plan ticket %q: %w", id, err)
	}

	tasks, err := s.graph.GetChildren(ctx, trackingSet, id)
	if err != nil {
		return types.Plan{}, fmt.Errorf("get plan tasks: %w", err)
	}

	return types.Plan{Ticket: ticket, Tasks: tasks}, nil
}

func (s *planStore) List(ctx context.Context, trackingSet string) ([]types.Plan, error) {
	dir := filepath.Join(s.basePath, "tracking-sets", trackingSet, "tickets")

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tickets directory: %w", err)
	}

	var result []types.Plan
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read ticket %q: %w", entry.Name(), err)
		}

		ticket, err := ParseTicket(data)
		if err != nil {
			return nil, fmt.Errorf("parse ticket %q: %w", entry.Name(), err)
		}

		// Filter by "kind:plan" label.
		if !hasLabel(ticket.Labels, "kind:plan") {
			continue
		}

		tasks, err := s.graph.GetChildren(ctx, trackingSet, ticket.ID)
		if err != nil {
			return nil, fmt.Errorf("get plan tasks for %q: %w", ticket.ID, err)
		}

		result = append(result, types.Plan{Ticket: ticket, Tasks: tasks})
	}
	return result, nil
}

func (s *planStore) Update(_ context.Context, trackingSet string, plan types.Plan) error {
	path := ticketPath(s.basePath, trackingSet, plan.ID)

	existingData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan ticket %q: %w", plan.ID, adapter.ErrNotFound)
		}
		return fmt.Errorf("read existing plan ticket %q: %w", plan.ID, err)
	}

	existing, err := ParseTicket(existingData)
	if err != nil {
		return fmt.Errorf("parse existing plan ticket %q: %w", plan.ID, err)
	}

	plan.Created = existing.Created
	plan.Updated = time.Now()

	data, err := WriteTicket(plan.Ticket)
	if err != nil {
		return fmt.Errorf("write plan ticket: %w", err)
	}

	return atomicWrite(path, data)
}

func (s *planStore) Delete(_ context.Context, trackingSet string, id string) error {
	if err := os.Remove(ticketPath(s.basePath, trackingSet, id)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan ticket %q: %w", id, adapter.ErrNotFound)
		}
		return err
	}
	return nil
}

// hasLabel checks if a label exists in a slice of labels.
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}
