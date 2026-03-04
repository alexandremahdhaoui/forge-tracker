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
var _ adapter.TicketStore = (*ticketStore)(nil)

// ticketStore implements adapter.TicketStore using markdown files.
type ticketStore struct {
	*markdownStore
	index *labelIndex
}

func ticketPath(basePath, trackingSet, id string) string {
	return filepath.Join(basePath, "tracking-sets", trackingSet, "tickets", id+".md")
}

func (s *ticketStore) Create(_ context.Context, trackingSet string, ticket types.Ticket) error {
	if ticket.ID == "" {
		return fmt.Errorf("ticket ID must not be empty")
	}
	if ticket.Title == "" {
		return fmt.Errorf("ticket title must not be empty")
	}

	now := time.Now()
	if ticket.Created.IsZero() {
		ticket.Created = now
	}
	if ticket.Updated.IsZero() {
		ticket.Updated = now
	}

	data, err := WriteTicket(ticket)
	if err != nil {
		return fmt.Errorf("write ticket: %w", err)
	}

	path := ticketPath(s.basePath, trackingSet, ticket.ID)
	if err := atomicWrite(path, data); err != nil {
		return err
	}

	if s.index != nil {
		s.index.Add(trackingSet, ticket.ID, ticket.Labels)
	}
	return nil
}

func (s *ticketStore) Get(_ context.Context, trackingSet string, id string) (types.Ticket, error) {
	path := ticketPath(s.basePath, trackingSet, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.Ticket{}, fmt.Errorf("ticket %q: %w", id, adapter.ErrNotFound)
		}
		return types.Ticket{}, fmt.Errorf("read ticket %q: %w", id, err)
	}
	return ParseTicket(data)
}

func (s *ticketStore) List(_ context.Context, trackingSet string, filter adapter.TicketFilter) ([]types.Ticket, error) {
	dir := filepath.Join(s.basePath, "tracking-sets", trackingSet, "tickets")

	// If filter has labels and we have an index, use it to narrow candidates.
	var candidateIDs map[string]struct{}
	if len(filter.Labels) > 0 && s.index != nil {
		candidateIDs = s.index.MatchLabels(trackingSet, filter.Labels)
		if candidateIDs != nil && len(candidateIDs) == 0 {
			return nil, nil
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tickets directory: %w", err)
	}

	var result []types.Ticket
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".md")

		// Skip if index filtering is active and this ticket is not a candidate.
		if candidateIDs != nil {
			if _, ok := candidateIDs[id]; !ok {
				continue
			}
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read ticket %q: %w", entry.Name(), err)
		}

		ticket, err := ParseTicket(data)
		if err != nil {
			return nil, fmt.Errorf("parse ticket %q: %w", entry.Name(), err)
		}

		if !matchesFilter(ticket, filter) {
			continue
		}

		result = append(result, ticket)
	}
	return result, nil
}

func (s *ticketStore) Update(_ context.Context, trackingSet string, ticket types.Ticket) error {
	path := ticketPath(s.basePath, trackingSet, ticket.ID)

	// Read existing to preserve Created timestamp and get old labels.
	existingData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ticket %q: %w", ticket.ID, adapter.ErrNotFound)
		}
		return fmt.Errorf("read existing ticket %q: %w", ticket.ID, err)
	}

	existing, err := ParseTicket(existingData)
	if err != nil {
		return fmt.Errorf("parse existing ticket %q: %w", ticket.ID, err)
	}

	ticket.Created = existing.Created
	ticket.Updated = time.Now()

	data, err := WriteTicket(ticket)
	if err != nil {
		return fmt.Errorf("write ticket: %w", err)
	}

	if err := atomicWrite(path, data); err != nil {
		return err
	}

	if s.index != nil {
		s.index.Update(trackingSet, ticket.ID, existing.Labels, ticket.Labels)
	}
	return nil
}

func (s *ticketStore) Delete(_ context.Context, trackingSet string, id string) error {
	path := ticketPath(s.basePath, trackingSet, id)

	// Read ticket to get labels before deleting.
	if s.index != nil {
		data, err := os.ReadFile(path)
		if err == nil {
			if ticket, err := ParseTicket(data); err == nil {
				s.index.Remove(trackingSet, id, ticket.Labels)
			}
		}
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ticket %q: %w", id, adapter.ErrNotFound)
		}
		return err
	}
	return nil
}

func (s *ticketStore) AddComment(_ context.Context, trackingSet string, id string, comment types.Comment) error {
	path := ticketPath(s.basePath, trackingSet, id)

	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ticket %q: %w", id, adapter.ErrNotFound)
		}
		return fmt.Errorf("read ticket %q: %w", id, err)
	}

	data, err := AppendComment(existing, comment)
	if err != nil {
		return fmt.Errorf("append comment: %w", err)
	}

	return atomicWrite(path, data)
}

// matchesFilter checks if a ticket matches the given filter criteria.
func matchesFilter(ticket types.Ticket, filter adapter.TicketFilter) bool {
	if filter.Status != "" && ticket.Status != filter.Status {
		return false
	}
	if filter.Assignee != "" && ticket.Assignee != filter.Assignee {
		return false
	}
	if filter.Priority != nil && ticket.Priority != *filter.Priority {
		return false
	}
	// AND semantics: ticket must have ALL filter labels.
	if len(filter.Labels) > 0 {
		labelSet := make(map[string]struct{}, len(ticket.Labels))
		for _, l := range ticket.Labels {
			labelSet[l] = struct{}{}
		}
		for _, l := range filter.Labels {
			if _, ok := labelSet[l]; !ok {
				return false
			}
		}
	}
	return true
}

// atomicWrite writes data to path using a temp file + rename.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
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
