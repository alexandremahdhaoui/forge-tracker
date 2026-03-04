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

package rest

import (
	"context"
	"errors"
	"strings"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter"
	"github.com/alexandremahdhaoui/forge-tracker/internal/controller"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

// APIHandler implements the generated StrictServerInterface, delegating to
// controller services and returning typed JSON responses.
type APIHandler struct {
	TicketService      controller.TicketService
	GraphService       controller.GraphService
	TrackingSetService controller.TrackingSetService
	PlanService        controller.PlanService
	MetaPlanService    controller.MetaPlanService
}

// NewAPIHandler creates an APIHandler wired to the given controller services.
func NewAPIHandler(
	ts controller.TicketService,
	gs controller.GraphService,
	tss controller.TrackingSetService,
	ps controller.PlanService,
	mps controller.MetaPlanService,
) *APIHandler {
	return &APIHandler{
		TicketService:      ts,
		GraphService:       gs,
		TrackingSetService: tss,
		PlanService:        ps,
		MetaPlanService:    mps,
	}
}

// Compile-time check that APIHandler satisfies StrictServerInterface.
var _ StrictServerInterface = (*APIHandler)(nil)

// --- Tracking Sets ---

func (h *APIHandler) ListTrackingSets(ctx context.Context, _ ListTrackingSetsRequestObject) (ListTrackingSetsResponseObject, error) {
	result, err := h.TrackingSetService.List(ctx)
	if err != nil {
		return ListTrackingSets500JSONResponse{Error: err.Error()}, nil
	}
	return ListTrackingSets200JSONResponse(result), nil
}

func (h *APIHandler) CreateTrackingSet(ctx context.Context, request CreateTrackingSetRequestObject) (CreateTrackingSetResponseObject, error) {
	ts := types.TrackingSet{Name: request.Body.Name}
	result, err := h.TrackingSetService.Create(ctx, ts)
	if err != nil {
		if errors.Is(err, adapter.ErrValidation) {
			return CreateTrackingSet400JSONResponse{Error: err.Error()}, nil
		}
		return CreateTrackingSet500JSONResponse{Error: err.Error()}, nil
	}
	return CreateTrackingSet201JSONResponse(result), nil
}

func (h *APIHandler) GetTrackingSet(ctx context.Context, request GetTrackingSetRequestObject) (GetTrackingSetResponseObject, error) {
	result, err := h.TrackingSetService.Get(ctx, request.Ts)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return GetTrackingSet404JSONResponse{Error: err.Error()}, nil
		}
		return GetTrackingSet500JSONResponse{Error: err.Error()}, nil
	}
	return GetTrackingSet200JSONResponse(result), nil
}

func (h *APIHandler) DeleteTrackingSet(ctx context.Context, request DeleteTrackingSetRequestObject) (DeleteTrackingSetResponseObject, error) {
	if err := h.TrackingSetService.Delete(ctx, request.Ts); err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return DeleteTrackingSet404JSONResponse{Error: err.Error()}, nil
		}
		return DeleteTrackingSet500JSONResponse{Error: err.Error()}, nil
	}
	return DeleteTrackingSet204Response{}, nil
}

// --- Tickets ---

func (h *APIHandler) CreateTicket(ctx context.Context, request CreateTicketRequestObject) (CreateTicketResponseObject, error) {
	ticket := ticketFromCreateRequest(request.Body)
	result, err := h.TicketService.Create(ctx, request.Ts, ticket)
	if err != nil {
		if errors.Is(err, adapter.ErrValidation) {
			return CreateTicket400JSONResponse{Error: err.Error()}, nil
		}
		return CreateTicket500JSONResponse{Error: err.Error()}, nil
	}
	return CreateTicket201JSONResponse(result), nil
}

func (h *APIHandler) ListTickets(ctx context.Context, request ListTicketsRequestObject) (ListTicketsResponseObject, error) {
	filter := adapter.TicketFilter{
		Priority: request.Params.Priority,
	}
	if request.Params.Status != nil {
		filter.Status = *request.Params.Status
	}
	if request.Params.Assignee != nil {
		filter.Assignee = *request.Params.Assignee
	}
	if request.Params.Labels != nil && *request.Params.Labels != "" {
		filter.Labels = strings.Split(*request.Params.Labels, ",")
	}

	result, err := h.TicketService.List(ctx, request.Ts, filter)
	if err != nil {
		return ListTickets500JSONResponse{Error: err.Error()}, nil
	}
	return ListTickets200JSONResponse(result), nil
}

func (h *APIHandler) GetTicket(ctx context.Context, request GetTicketRequestObject) (GetTicketResponseObject, error) {
	result, err := h.TicketService.Get(ctx, request.Ts, request.Id)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return GetTicket404JSONResponse{Error: err.Error()}, nil
		}
		return GetTicket500JSONResponse{Error: err.Error()}, nil
	}
	return GetTicket200JSONResponse(result), nil
}

func (h *APIHandler) UpdateTicket(ctx context.Context, request UpdateTicketRequestObject) (UpdateTicketResponseObject, error) {
	ticket := ticketFromUpdateRequest(request.Id, request.Body)
	result, err := h.TicketService.Update(ctx, request.Ts, ticket)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return UpdateTicket404JSONResponse{Error: err.Error()}, nil
		}
		return UpdateTicket500JSONResponse{Error: err.Error()}, nil
	}
	return UpdateTicket200JSONResponse(result), nil
}

func (h *APIHandler) DeleteTicket(ctx context.Context, request DeleteTicketRequestObject) (DeleteTicketResponseObject, error) {
	if err := h.TicketService.Delete(ctx, request.Ts, request.Id); err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return DeleteTicket404JSONResponse{Error: err.Error()}, nil
		}
		return DeleteTicket500JSONResponse{Error: err.Error()}, nil
	}
	return DeleteTicket204Response{}, nil
}

func (h *APIHandler) AddComment(ctx context.Context, request AddCommentRequestObject) (AddCommentResponseObject, error) {
	comment := types.Comment{
		Author: request.Body.Author,
		Text:   request.Body.Text,
	}
	result, err := h.TicketService.AddComment(ctx, request.Ts, request.Id, comment)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return AddComment404JSONResponse{Error: err.Error()}, nil
		}
		return AddComment500JSONResponse{Error: err.Error()}, nil
	}
	return AddComment201JSONResponse(result), nil
}

// --- Edges ---

func (h *APIHandler) ListEdges(ctx context.Context, request ListEdgesRequestObject) (ListEdgesResponseObject, error) {
	// If ticket filter is provided, get edges for that ticket.
	if request.Params.Ticket != nil && *request.Params.Ticket != "" {
		edges, err := h.GraphService.ListEdges(ctx, request.Ts, *request.Params.Ticket)
		if err != nil {
			return ListEdges500JSONResponse{Error: err.Error()}, nil
		}
		// Apply type filter if provided.
		if request.Params.Type != nil && *request.Params.Type != "" {
			edges = filterEdgesByType(edges, *request.Params.Type)
		}
		return ListEdges200JSONResponse(edges), nil
	}

	// Get the full graph and return all edges (with optional type filter).
	graph, err := h.GraphService.GetGraph(ctx, request.Ts)
	if err != nil {
		return ListEdges500JSONResponse{Error: err.Error()}, nil
	}
	edges := graph.Edges
	if request.Params.Type != nil && *request.Params.Type != "" {
		edges = filterEdgesByType(edges, *request.Params.Type)
	}
	return ListEdges200JSONResponse(edges), nil
}

func (h *APIHandler) AddEdge(ctx context.Context, request AddEdgeRequestObject) (AddEdgeResponseObject, error) {
	edge := types.Edge{
		From: request.Body.From,
		To:   request.Body.To,
		Type: request.Body.Type,
	}
	result, err := h.GraphService.AddEdge(ctx, request.Ts, edge)
	if err != nil {
		if errors.Is(err, adapter.ErrValidation) || errors.Is(err, adapter.ErrNotFound) {
			return AddEdge400JSONResponse{Error: err.Error()}, nil
		}
		return AddEdge500JSONResponse{Error: err.Error()}, nil
	}
	return AddEdge201JSONResponse(result), nil
}

func (h *APIHandler) RemoveEdge(ctx context.Context, request RemoveEdgeRequestObject) (RemoveEdgeResponseObject, error) {
	edge := types.Edge{
		From: request.Body.From,
		To:   request.Body.To,
		Type: request.Body.Type,
	}
	if err := h.GraphService.RemoveEdge(ctx, request.Ts, edge); err != nil {
		return RemoveEdge500JSONResponse{Error: err.Error()}, nil
	}
	return RemoveEdge204Response{}, nil
}

// --- Children / Blocking ---

func (h *APIHandler) GetChildren(ctx context.Context, request GetChildrenRequestObject) (GetChildrenResponseObject, error) {
	result, err := h.GraphService.GetChildren(ctx, request.Ts, request.Id)
	if err != nil {
		return GetChildren500JSONResponse{Error: err.Error()}, nil
	}
	return GetChildren200JSONResponse(result), nil
}

func (h *APIHandler) GetBlocking(ctx context.Context, request GetBlockingRequestObject) (GetBlockingResponseObject, error) {
	result, err := h.GraphService.GetBlocking(ctx, request.Ts, request.Id)
	if err != nil {
		return GetBlocking500JSONResponse{Error: err.Error()}, nil
	}
	return GetBlocking200JSONResponse(result), nil
}

// --- Plans ---

func (h *APIHandler) CreatePlan(ctx context.Context, request CreatePlanRequestObject) (CreatePlanResponseObject, error) {
	plan := planFromCreateRequest(request.Body)
	result, err := h.PlanService.Create(ctx, request.Ts, plan)
	if err != nil {
		if errors.Is(err, adapter.ErrValidation) {
			return CreatePlan400JSONResponse{Error: err.Error()}, nil
		}
		return CreatePlan500JSONResponse{Error: err.Error()}, nil
	}
	return CreatePlan201JSONResponse(result), nil
}

func (h *APIHandler) ListPlans(ctx context.Context, request ListPlansRequestObject) (ListPlansResponseObject, error) {
	result, err := h.PlanService.List(ctx, request.Ts)
	if err != nil {
		return ListPlans500JSONResponse{Error: err.Error()}, nil
	}
	return ListPlans200JSONResponse(result), nil
}

func (h *APIHandler) GetPlan(ctx context.Context, request GetPlanRequestObject) (GetPlanResponseObject, error) {
	result, err := h.PlanService.Get(ctx, request.Ts, request.Id)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return GetPlan404JSONResponse{Error: err.Error()}, nil
		}
		return GetPlan500JSONResponse{Error: err.Error()}, nil
	}
	return GetPlan200JSONResponse(result), nil
}

func (h *APIHandler) UpdatePlan(ctx context.Context, request UpdatePlanRequestObject) (UpdatePlanResponseObject, error) {
	plan := planFromUpdateRequest(request.Id, request.Body)
	result, err := h.PlanService.Update(ctx, request.Ts, plan)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return UpdatePlan404JSONResponse{Error: err.Error()}, nil
		}
		return UpdatePlan500JSONResponse{Error: err.Error()}, nil
	}
	return UpdatePlan200JSONResponse(result), nil
}

func (h *APIHandler) DeletePlan(ctx context.Context, request DeletePlanRequestObject) (DeletePlanResponseObject, error) {
	if err := h.PlanService.Delete(ctx, request.Ts, request.Id); err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return DeletePlan404JSONResponse{Error: err.Error()}, nil
		}
		return DeletePlan500JSONResponse{Error: err.Error()}, nil
	}
	return DeletePlan204Response{}, nil
}

// --- Meta-Plans ---

func (h *APIHandler) CreateMetaPlan(ctx context.Context, request CreateMetaPlanRequestObject) (CreateMetaPlanResponseObject, error) {
	mp := metaPlanFromCreateRequest(request.Body)
	result, err := h.MetaPlanService.Create(ctx, request.Ts, mp)
	if err != nil {
		if errors.Is(err, adapter.ErrValidation) {
			return CreateMetaPlan400JSONResponse{Error: err.Error()}, nil
		}
		return CreateMetaPlan500JSONResponse{Error: err.Error()}, nil
	}
	return CreateMetaPlan201JSONResponse(result), nil
}

func (h *APIHandler) ListMetaPlans(ctx context.Context, request ListMetaPlansRequestObject) (ListMetaPlansResponseObject, error) {
	result, err := h.MetaPlanService.List(ctx, request.Ts)
	if err != nil {
		return ListMetaPlans500JSONResponse{Error: err.Error()}, nil
	}
	return ListMetaPlans200JSONResponse(result), nil
}

func (h *APIHandler) GetMetaPlan(ctx context.Context, request GetMetaPlanRequestObject) (GetMetaPlanResponseObject, error) {
	result, err := h.MetaPlanService.Get(ctx, request.Ts, request.Id)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return GetMetaPlan404JSONResponse{Error: err.Error()}, nil
		}
		return GetMetaPlan500JSONResponse{Error: err.Error()}, nil
	}
	return GetMetaPlan200JSONResponse(result), nil
}

func (h *APIHandler) UpdateMetaPlan(ctx context.Context, request UpdateMetaPlanRequestObject) (UpdateMetaPlanResponseObject, error) {
	mp := metaPlanFromUpdateRequest(request.Id, request.Body)
	result, err := h.MetaPlanService.Update(ctx, request.Ts, mp)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return UpdateMetaPlan404JSONResponse{Error: err.Error()}, nil
		}
		return UpdateMetaPlan500JSONResponse{Error: err.Error()}, nil
	}
	return UpdateMetaPlan200JSONResponse(result), nil
}

func (h *APIHandler) DeleteMetaPlan(ctx context.Context, request DeleteMetaPlanRequestObject) (DeleteMetaPlanResponseObject, error) {
	if err := h.MetaPlanService.Delete(ctx, request.Ts, request.Id); err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return DeleteMetaPlan404JSONResponse{Error: err.Error()}, nil
		}
		return DeleteMetaPlan500JSONResponse{Error: err.Error()}, nil
	}
	return DeleteMetaPlan204Response{}, nil
}

// --- Helper functions ---

func ticketFromCreateRequest(req *CreateTicketRequest) types.Ticket {
	t := types.Ticket{
		ID:    req.Id,
		Title: req.Title,
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.Labels != nil {
		t.Labels = *req.Labels
	}
	if req.Annotations != nil {
		t.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		t.Assignee = *req.Assignee
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	return t
}

func ticketFromUpdateRequest(id string, req *UpdateTicketRequest) types.Ticket {
	t := types.Ticket{
		ID:    id,
		Title: req.Title,
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.Labels != nil {
		t.Labels = *req.Labels
	}
	if req.Annotations != nil {
		t.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		t.Assignee = *req.Assignee
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	return t
}

func planFromCreateRequest(req *CreatePlanRequest) types.Plan {
	p := types.Plan{
		Ticket: types.Ticket{
			ID:    req.Id,
			Title: req.Title,
		},
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.Labels != nil {
		p.Labels = *req.Labels
	}
	if req.Annotations != nil {
		p.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		p.Assignee = *req.Assignee
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Tasks != nil {
		p.Tasks = *req.Tasks
	}
	return p
}

func planFromUpdateRequest(id string, req *UpdatePlanRequest) types.Plan {
	p := types.Plan{
		Ticket: types.Ticket{
			ID:    id,
			Title: req.Title,
		},
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.Labels != nil {
		p.Labels = *req.Labels
	}
	if req.Annotations != nil {
		p.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		p.Assignee = *req.Assignee
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Tasks != nil {
		p.Tasks = *req.Tasks
	}
	return p
}

func metaPlanFromCreateRequest(req *CreateMetaPlanRequest) types.MetaPlan {
	mp := types.MetaPlan{
		Ticket: types.Ticket{
			ID:    req.Id,
			Title: req.Title,
		},
	}
	if req.Status != nil {
		mp.Status = *req.Status
	}
	if req.Priority != nil {
		mp.Priority = *req.Priority
	}
	if req.Labels != nil {
		mp.Labels = *req.Labels
	}
	if req.Annotations != nil {
		mp.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		mp.Assignee = *req.Assignee
	}
	if req.Description != nil {
		mp.Description = *req.Description
	}
	if req.Stages != nil {
		mp.Stages = *req.Stages
	}
	if req.Checkpoints != nil {
		mp.Checkpoints = *req.Checkpoints
	}
	return mp
}

func metaPlanFromUpdateRequest(id string, req *UpdateMetaPlanRequest) types.MetaPlan {
	mp := types.MetaPlan{
		Ticket: types.Ticket{
			ID:    id,
			Title: req.Title,
		},
	}
	if req.Status != nil {
		mp.Status = *req.Status
	}
	if req.Priority != nil {
		mp.Priority = *req.Priority
	}
	if req.Labels != nil {
		mp.Labels = *req.Labels
	}
	if req.Annotations != nil {
		mp.Annotations = *req.Annotations
	}
	if req.Assignee != nil {
		mp.Assignee = *req.Assignee
	}
	if req.Description != nil {
		mp.Description = *req.Description
	}
	if req.Stages != nil {
		mp.Stages = *req.Stages
	}
	if req.Checkpoints != nil {
		mp.Checkpoints = *req.Checkpoints
	}
	return mp
}

func filterEdgesByType(edges []types.Edge, edgeType string) []types.Edge {
	var filtered []types.Edge
	for _, e := range edges {
		if e.Type == edgeType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
