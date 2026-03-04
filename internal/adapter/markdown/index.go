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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// labelIndex is an in-memory inverted index mapping labels to ticket IDs.
// Structure: trackingSet -> label -> set of ticket IDs.
type labelIndex struct {
	mu    sync.RWMutex
	index map[string]map[string]map[string]struct{}
}

func newLabelIndex() *labelIndex {
	return &labelIndex{
		index: make(map[string]map[string]map[string]struct{}),
	}
}

// BuildIndex scans all tracking sets under basePath and populates the index
// by parsing frontmatter from each ticket file.
func (li *labelIndex) BuildIndex(basePath string) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	li.index = make(map[string]map[string]map[string]struct{})

	tsDir := filepath.Join(basePath, "tracking-sets")
	tsDirs, err := os.ReadDir(tsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read tracking-sets directory: %w", err)
	}

	for _, tsEntry := range tsDirs {
		if !tsEntry.IsDir() {
			continue
		}
		tsName := tsEntry.Name()
		ticketsDir := filepath.Join(tsDir, tsName, "tickets")

		entries, err := os.ReadDir(ticketsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read tickets directory for %q: %w", tsName, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			id := strings.TrimSuffix(entry.Name(), ".md")
			data, err := os.ReadFile(filepath.Join(ticketsDir, entry.Name()))
			if err != nil {
				return fmt.Errorf("read ticket %q: %w", entry.Name(), err)
			}

			ticket, err := ParseTicket(data)
			if err != nil {
				return fmt.Errorf("parse ticket %q: %w", entry.Name(), err)
			}

			li.addLocked(tsName, id, ticket.Labels)
		}
	}

	return nil
}

// Add adds all labels for a ticket to the index.
func (li *labelIndex) Add(trackingSet, ticketID string, labels []string) {
	li.mu.Lock()
	defer li.mu.Unlock()
	li.addLocked(trackingSet, ticketID, labels)
}

func (li *labelIndex) addLocked(trackingSet, ticketID string, labels []string) {
	tsIndex, ok := li.index[trackingSet]
	if !ok {
		tsIndex = make(map[string]map[string]struct{})
		li.index[trackingSet] = tsIndex
	}

	for _, label := range labels {
		ids, ok := tsIndex[label]
		if !ok {
			ids = make(map[string]struct{})
			tsIndex[label] = ids
		}
		ids[ticketID] = struct{}{}
	}
}

// Remove removes label entries for a ticket from the index.
func (li *labelIndex) Remove(trackingSet, ticketID string, labels []string) {
	li.mu.Lock()
	defer li.mu.Unlock()

	tsIndex, ok := li.index[trackingSet]
	if !ok {
		return
	}

	for _, label := range labels {
		ids, ok := tsIndex[label]
		if !ok {
			continue
		}
		delete(ids, ticketID)
		if len(ids) == 0 {
			delete(tsIndex, label)
		}
	}
}

// Update diffs old vs new labels and updates the index accordingly.
func (li *labelIndex) Update(trackingSet, ticketID string, oldLabels, newLabels []string) {
	li.mu.Lock()
	defer li.mu.Unlock()

	oldSet := make(map[string]struct{}, len(oldLabels))
	for _, l := range oldLabels {
		oldSet[l] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newLabels))
	for _, l := range newLabels {
		newSet[l] = struct{}{}
	}

	// Remove labels that are in old but not in new.
	var toRemove []string
	for l := range oldSet {
		if _, ok := newSet[l]; !ok {
			toRemove = append(toRemove, l)
		}
	}

	// Add labels that are in new but not in old.
	var toAdd []string
	for l := range newSet {
		if _, ok := oldSet[l]; !ok {
			toAdd = append(toAdd, l)
		}
	}

	tsIndex, ok := li.index[trackingSet]
	if !ok {
		tsIndex = make(map[string]map[string]struct{})
		li.index[trackingSet] = tsIndex
	}

	for _, label := range toRemove {
		ids, ok := tsIndex[label]
		if !ok {
			continue
		}
		delete(ids, ticketID)
		if len(ids) == 0 {
			delete(tsIndex, label)
		}
	}

	for _, label := range toAdd {
		ids, ok := tsIndex[label]
		if !ok {
			ids = make(map[string]struct{})
			tsIndex[label] = ids
		}
		ids[ticketID] = struct{}{}
	}
}

// MatchLabels returns ticket IDs that match ALL provided labels (set intersection).
// Returns nil if labels is empty (no filtering).
func (li *labelIndex) MatchLabels(trackingSet string, labels []string) map[string]struct{} {
	if len(labels) == 0 {
		return nil
	}

	li.mu.RLock()
	defer li.mu.RUnlock()

	tsIndex, ok := li.index[trackingSet]
	if !ok {
		return make(map[string]struct{})
	}

	// Start with the set of IDs for the first label, then intersect.
	var result map[string]struct{}
	for i, label := range labels {
		ids, ok := tsIndex[label]
		if !ok {
			return make(map[string]struct{})
		}

		if i == 0 {
			result = make(map[string]struct{}, len(ids))
			for id := range ids {
				result[id] = struct{}{}
			}
			continue
		}

		// Intersect with current result.
		for id := range result {
			if _, ok := ids[id]; !ok {
				delete(result, id)
			}
		}

		if len(result) == 0 {
			return result
		}
	}

	return result
}
