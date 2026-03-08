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

// PlanService defines business logic for plan operations.
type PlanService interface {
	Create(ctx context.Context, trackingSet string, plan types.Plan) (types.Plan, error)
	Get(ctx context.Context, trackingSet string, id string) (types.Plan, error)
	List(ctx context.Context, trackingSet string) ([]types.Plan, error)
	Update(ctx context.Context, trackingSet string, plan types.Plan) (types.Plan, error)
	Delete(ctx context.Context, trackingSet string, id string) error
}

var _ PlanService = (*planService)(nil)

type planService struct {
	plans  adapter.PlanStore
	graphs adapter.GraphStore
}

// NewPlanService creates a PlanService with the given stores.
func NewPlanService(ps adapter.PlanStore, gs adapter.GraphStore) PlanService {
	return &planService{
		plans:  ps,
		graphs: gs,
	}
}

func (s *planService) Create(ctx context.Context, trackingSet string, plan types.Plan) (types.Plan, error) {
	if err := s.plans.Create(ctx, trackingSet, plan); err != nil {
		return types.Plan{}, fmt.Errorf("creating plan: %w", err)
	}
	if err := s.graphs.AddNode(ctx, trackingSet, plan.ID); err != nil {
		return types.Plan{}, fmt.Errorf("adding plan node to graph: %w", err)
	}
	for _, taskID := range plan.Tasks {
		edge := types.Edge{
			From: plan.ID,
			To:   taskID,
			Type: "parent",
		}
		if err := s.graphs.AddEdge(ctx, trackingSet, edge); err != nil {
			return types.Plan{}, fmt.Errorf("adding parent edge from %q to %q: %w", plan.ID, taskID, err)
		}
	}
	return plan, nil
}

func (s *planService) Get(ctx context.Context, trackingSet string, id string) (types.Plan, error) {
	return s.plans.Get(ctx, trackingSet, id)
}

func (s *planService) List(ctx context.Context, trackingSet string) ([]types.Plan, error) {
	return s.plans.List(ctx, trackingSet)
}

func (s *planService) Update(ctx context.Context, trackingSet string, plan types.Plan) (types.Plan, error) {
	if err := s.plans.Update(ctx, trackingSet, plan); err != nil {
		return types.Plan{}, fmt.Errorf("updating plan: %w", err)
	}
	return s.plans.Get(ctx, trackingSet, plan.ID)
}

func (s *planService) Delete(ctx context.Context, trackingSet string, id string) error {
	return s.plans.Delete(ctx, trackingSet, id)
}
