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

// MetaPlanService defines business logic for meta-plan operations.
type MetaPlanService interface {
	Create(ctx context.Context, trackingSet string, mp types.MetaPlan) (types.MetaPlan, error)
	Get(ctx context.Context, trackingSet string, id string) (types.MetaPlan, error)
	List(ctx context.Context, trackingSet string) ([]types.MetaPlan, error)
	Update(ctx context.Context, trackingSet string, mp types.MetaPlan) (types.MetaPlan, error)
	Delete(ctx context.Context, trackingSet string, id string) error
}

var _ MetaPlanService = (*metaPlanService)(nil)

type metaPlanService struct {
	store adapter.MetaPlanStore
}

// NewMetaPlanService creates a MetaPlanService with the given store.
func NewMetaPlanService(mps adapter.MetaPlanStore) MetaPlanService {
	return &metaPlanService{store: mps}
}

func (s *metaPlanService) Create(ctx context.Context, trackingSet string, mp types.MetaPlan) (types.MetaPlan, error) {
	if mp.ID == "" {
		return types.MetaPlan{}, fmt.Errorf("meta-plan ID must not be empty: %w", adapter.ErrValidation)
	}
	if err := s.store.Create(ctx, trackingSet, mp); err != nil {
		return types.MetaPlan{}, fmt.Errorf("creating meta-plan: %w", err)
	}
	return mp, nil
}

func (s *metaPlanService) Get(ctx context.Context, trackingSet string, id string) (types.MetaPlan, error) {
	return s.store.Get(ctx, trackingSet, id)
}

func (s *metaPlanService) List(ctx context.Context, trackingSet string) ([]types.MetaPlan, error) {
	return s.store.List(ctx, trackingSet)
}

func (s *metaPlanService) Update(ctx context.Context, trackingSet string, mp types.MetaPlan) (types.MetaPlan, error) {
	if err := s.store.Update(ctx, trackingSet, mp); err != nil {
		return types.MetaPlan{}, fmt.Errorf("updating meta-plan: %w", err)
	}
	return s.store.Get(ctx, trackingSet, mp.ID)
}

func (s *metaPlanService) Delete(ctx context.Context, trackingSet string, id string) error {
	return s.store.Delete(ctx, trackingSet, id)
}
