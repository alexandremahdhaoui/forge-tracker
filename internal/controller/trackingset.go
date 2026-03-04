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
	"regexp"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// TrackingSetService defines business logic for tracking set operations.
type TrackingSetService interface {
	Create(ctx context.Context, ts types.TrackingSet) (types.TrackingSet, error)
	Get(ctx context.Context, name string) (types.TrackingSet, error)
	List(ctx context.Context) ([]types.TrackingSet, error)
	Delete(ctx context.Context, name string) error
}

var _ TrackingSetService = (*trackingSetService)(nil)

var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type trackingSetService struct {
	store adapter.TrackingSetStore
}

// NewTrackingSetService creates a TrackingSetService with the given store.
func NewTrackingSetService(store adapter.TrackingSetStore) TrackingSetService {
	return &trackingSetService{store: store}
}

func (s *trackingSetService) Create(ctx context.Context, ts types.TrackingSet) (types.TrackingSet, error) {
	if ts.Name == "" {
		return types.TrackingSet{}, fmt.Errorf("tracking set name must not be empty: %w", adapter.ErrValidation)
	}
	if !validNamePattern.MatchString(ts.Name) {
		return types.TrackingSet{}, fmt.Errorf("tracking set name %q must contain only alphanumeric characters, hyphens, and underscores: %w", ts.Name, adapter.ErrValidation)
	}
	if err := s.store.Create(ctx, ts); err != nil {
		return types.TrackingSet{}, fmt.Errorf("creating tracking set: %w", err)
	}
	return ts, nil
}

func (s *trackingSetService) Get(ctx context.Context, name string) (types.TrackingSet, error) {
	return s.store.Get(ctx, name)
}

func (s *trackingSetService) List(ctx context.Context) ([]types.TrackingSet, error) {
	return s.store.List(ctx)
}

func (s *trackingSetService) Delete(ctx context.Context, name string) error {
	return s.store.Delete(ctx, name)
}
