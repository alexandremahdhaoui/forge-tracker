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
	"gopkg.in/yaml.v3"
)

// Compile-time interface check.
var _ adapter.MetaPlanStore = (*metaPlanStore)(nil)

// metaPlanStore implements adapter.MetaPlanStore using markdown + YAML files.
type metaPlanStore struct {
	*markdownStore
}

// metaPlanExtra holds the YAML-serializable parts of a MetaPlan that are
// stored in a separate .metaplan.yaml file.
type metaPlanExtra struct {
	Stages      []types.MetaPlanStage  `yaml:"stages"`
	Checkpoints []types.MetaCheckpoint `yaml:"checkpoints"`
}

func metaPlanPath(basePath, trackingSet, id string) string {
	return filepath.Join(basePath, "tracking-sets", trackingSet, "tickets", id+".metaplan.yaml")
}

func (s *metaPlanStore) Create(_ context.Context, trackingSet string, mp types.MetaPlan) error {
	now := time.Now()
	if mp.Created.IsZero() {
		mp.Created = now
	}
	if mp.Updated.IsZero() {
		mp.Updated = now
	}

	// Write the ticket markdown file.
	ticketData, err := WriteTicket(mp.Ticket)
	if err != nil {
		return fmt.Errorf("write metaplan ticket: %w", err)
	}

	if err := atomicWrite(ticketPath(s.basePath, trackingSet, mp.ID), ticketData); err != nil {
		return fmt.Errorf("write metaplan ticket file: %w", err)
	}

	// Write the metaplan YAML file.
	extra := metaPlanExtra{
		Stages:      mp.Stages,
		Checkpoints: mp.Checkpoints,
	}

	yamlData, err := yaml.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal metaplan extra: %w", err)
	}

	if err := atomicWrite(metaPlanPath(s.basePath, trackingSet, mp.ID), yamlData); err != nil {
		return fmt.Errorf("write metaplan yaml file: %w", err)
	}

	return nil
}

func (s *metaPlanStore) Get(_ context.Context, trackingSet string, id string) (types.MetaPlan, error) {
	// Read ticket file.
	ticketData, err := os.ReadFile(ticketPath(s.basePath, trackingSet, id))
	if err != nil {
		if os.IsNotExist(err) {
			return types.MetaPlan{}, fmt.Errorf("metaplan ticket %q: %w", id, adapter.ErrNotFound)
		}
		return types.MetaPlan{}, fmt.Errorf("read metaplan ticket %q: %w", id, err)
	}

	ticket, err := ParseTicket(ticketData)
	if err != nil {
		return types.MetaPlan{}, fmt.Errorf("parse metaplan ticket %q: %w", id, err)
	}

	// Read metaplan YAML file.
	yamlData, err := os.ReadFile(metaPlanPath(s.basePath, trackingSet, id))
	if err != nil {
		if os.IsNotExist(err) {
			return types.MetaPlan{}, fmt.Errorf("metaplan yaml %q: %w", id, adapter.ErrNotFound)
		}
		return types.MetaPlan{}, fmt.Errorf("read metaplan yaml %q: %w", id, err)
	}

	var extra metaPlanExtra
	if err := yaml.Unmarshal(yamlData, &extra); err != nil {
		return types.MetaPlan{}, fmt.Errorf("unmarshal metaplan yaml %q: %w", id, err)
	}

	return types.MetaPlan{
		Ticket:      ticket,
		Stages:      extra.Stages,
		Checkpoints: extra.Checkpoints,
	}, nil
}

func (s *metaPlanStore) List(_ context.Context, trackingSet string) ([]types.MetaPlan, error) {
	dir := filepath.Join(s.basePath, "tracking-sets", trackingSet, "tickets")

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tickets directory: %w", err)
	}

	var result []types.MetaPlan
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".metaplan.yaml") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".metaplan.yaml")

		// Read ticket file.
		ticketData, err := os.ReadFile(ticketPath(s.basePath, trackingSet, id))
		if err != nil {
			return nil, fmt.Errorf("read metaplan ticket %q: %w", id, err)
		}

		ticket, err := ParseTicket(ticketData)
		if err != nil {
			return nil, fmt.Errorf("parse metaplan ticket %q: %w", id, err)
		}

		// Read metaplan YAML file.
		yamlData, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read metaplan yaml %q: %w", id, err)
		}

		var extra metaPlanExtra
		if err := yaml.Unmarshal(yamlData, &extra); err != nil {
			return nil, fmt.Errorf("unmarshal metaplan yaml %q: %w", id, err)
		}

		result = append(result, types.MetaPlan{
			Ticket:      ticket,
			Stages:      extra.Stages,
			Checkpoints: extra.Checkpoints,
		})
	}
	return result, nil
}

func (s *metaPlanStore) Update(_ context.Context, trackingSet string, mp types.MetaPlan) error {
	tPath := ticketPath(s.basePath, trackingSet, mp.ID)

	// Read existing to preserve Created.
	existingData, err := os.ReadFile(tPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("metaplan ticket %q: %w", mp.ID, adapter.ErrNotFound)
		}
		return fmt.Errorf("read existing metaplan ticket %q: %w", mp.ID, err)
	}

	existing, err := ParseTicket(existingData)
	if err != nil {
		return fmt.Errorf("parse existing metaplan ticket %q: %w", mp.ID, err)
	}

	mp.Created = existing.Created
	mp.Updated = time.Now()

	// Write ticket file.
	ticketData, err := WriteTicket(mp.Ticket)
	if err != nil {
		return fmt.Errorf("write metaplan ticket: %w", err)
	}

	if err := atomicWrite(tPath, ticketData); err != nil {
		return fmt.Errorf("write metaplan ticket file: %w", err)
	}

	// Write metaplan YAML file.
	extra := metaPlanExtra{
		Stages:      mp.Stages,
		Checkpoints: mp.Checkpoints,
	}

	yamlData, err := yaml.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal metaplan extra: %w", err)
	}

	if err := atomicWrite(metaPlanPath(s.basePath, trackingSet, mp.ID), yamlData); err != nil {
		return fmt.Errorf("write metaplan yaml file: %w", err)
	}

	return nil
}

func (s *metaPlanStore) Delete(_ context.Context, trackingSet string, id string) error {
	tPath := ticketPath(s.basePath, trackingSet, id)
	mpPath := metaPlanPath(s.basePath, trackingSet, id)

	// Remove both files. If the ticket file fails, still try to remove the yaml.
	tErr := os.Remove(tPath)
	mpErr := os.Remove(mpPath)

	if tErr != nil {
		if os.IsNotExist(tErr) {
			return fmt.Errorf("metaplan ticket %q: %w", id, adapter.ErrNotFound)
		}
		return fmt.Errorf("remove metaplan ticket file: %w", tErr)
	}
	if mpErr != nil {
		if os.IsNotExist(mpErr) {
			return fmt.Errorf("metaplan yaml %q: %w", id, adapter.ErrNotFound)
		}
		return fmt.Errorf("remove metaplan yaml file: %w", mpErr)
	}
	return nil
}
