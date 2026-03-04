//go:build integration

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

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter/markdown"
	"github.com/alexandremahdhaoui/forge-tracker/internal/controller"
	"github.com/alexandremahdhaoui/forge-tracker/internal/driver/rest"
	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	basePath := t.TempDir()

	store := markdown.NewStore(basePath)
	ticketSvc := controller.NewTicketService(store.TicketStore(), store.GraphStore())
	graphSvc := controller.NewGraphService(store.GraphStore(), store.TicketStore())
	tsSvc := controller.NewTrackingSetService(store.TrackingSetStore())
	planSvc := controller.NewPlanService(store.PlanStore(), store.GraphStore())
	mpSvc := controller.NewMetaPlanService(store.MetaPlanStore())

	handler := rest.NewAPIHandler(ticketSvc, graphSvc, tsSvc, planSvc, mpSvc)
	mux := http.NewServeMux()
	si := rest.NewStrictHandler(handler, nil)
	rest.HandlerFromMux(si, mux)

	return httptest.NewServer(mux)
}

const apiPrefix = "/api/v1"

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(data)
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, statusCode int) map[string]any {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		bodyReader = jsonBody(t, body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, statusCode, resp.StatusCode, "unexpected status for %s %s", method, url)

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	var result map[string]any
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if len(data) > 0 {
		require.NoError(t, json.Unmarshal(data, &result))
	}
	return result
}

func doJSONArray(t *testing.T, client *http.Client, method, url string, body any, statusCode int) []any {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		bodyReader = jsonBody(t, body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, statusCode, resp.StatusCode, "unexpected status for %s %s", method, url)

	var result []any
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if len(data) > 0 {
		require.NoError(t, json.Unmarshal(data, &result))
	}
	return result
}

func TestIntegration_FullLifecycle(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()
	client := srv.Client()
	base := srv.URL

	api := base + apiPrefix

	// 1. Create tracking set.
	doJSON(t, client, "POST", api+"/tracking-sets", map[string]string{"name": "my-ts"}, http.StatusCreated)

	// 2. Create tickets.
	doJSON(t, client, "POST", api+"/tracking-sets/my-ts/tickets",
		map[string]any{
			"id": "t-1", "title": "Ticket 1",
			"labels": []string{"kind:task", "area:api"},
		}, http.StatusCreated)
	doJSON(t, client, "POST", api+"/tracking-sets/my-ts/tickets",
		map[string]any{
			"id": "t-2", "title": "Ticket 2",
			"labels": []string{"kind:task"},
		}, http.StatusCreated)

	// 3. List tickets (no filter).
	tickets := doJSONArray(t, client, "GET", api+"/tracking-sets/my-ts/tickets", nil, http.StatusOK)
	assert.Len(t, tickets, 2)

	// 4. List tickets with label filter.
	tickets = doJSONArray(t, client, "GET", api+"/tracking-sets/my-ts/tickets?labels=area:api", nil, http.StatusOK)
	assert.Len(t, tickets, 1)

	// 5. Add parent edge: t-1 -> t-2.
	doJSON(t, client, "POST", api+"/tracking-sets/my-ts/edges",
		map[string]string{"from": "t-1", "to": "t-2", "type": "parent"}, http.StatusCreated)

	// 6. Get children of t-1.
	children := doJSONArray(t, client, "GET", api+"/tracking-sets/my-ts/tickets/t-1/children", nil, http.StatusOK)
	assert.Len(t, children, 1)

	// 7. Add blocking edge: t-2 blocks t-1.
	doJSON(t, client, "POST", api+"/tracking-sets/my-ts/edges",
		map[string]string{"from": "t-2", "to": "t-1", "type": "blocks"}, http.StatusCreated)

	// 8. Get blocking for t-1.
	blocking := doJSONArray(t, client, "GET", api+"/tracking-sets/my-ts/tickets/t-1/blocking", nil, http.StatusOK)
	assert.Len(t, blocking, 1)

	// 9. Add comment.
	doJSON(t, client, "POST", api+"/tracking-sets/my-ts/tickets/t-1/comments",
		map[string]string{"author": "user", "text": "A comment."}, http.StatusCreated)

	// 10. Get ticket includes comment.
	ticket := doJSON(t, client, "GET", api+"/tracking-sets/my-ts/tickets/t-1", nil, http.StatusOK)
	comments, ok := ticket["comments"].([]any)
	require.True(t, ok, "comments should be an array")
	assert.Len(t, comments, 1)

	// 11. Delete ticket t-2.
	doJSON(t, client, "DELETE", api+"/tracking-sets/my-ts/tickets/t-2", nil, http.StatusNoContent)

	// 12. Verify t-2 returns 404 after deletion.
	resp, err := client.Get(api + "/tracking-sets/my-ts/tickets/t-2")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestIntegration_PlanWithTasks(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()
	client := srv.Client()
	base := srv.URL

	api := base + apiPrefix

	// Create tracking set.
	doJSON(t, client, "POST", api+"/tracking-sets", map[string]string{"name": "plan-ts"}, http.StatusCreated)

	// Create all tickets first (they need to be graph nodes for parent edges).
	// The plan itself must also be a ticket/node for edges to work.
	doJSON(t, client, "POST", api+"/tracking-sets/plan-ts/tickets",
		map[string]any{"id": "plan-1", "title": "My Plan", "labels": []string{"kind:plan"}}, http.StatusCreated)
	doJSON(t, client, "POST", api+"/tracking-sets/plan-ts/tickets",
		map[string]any{"id": "task-1", "title": "Task 1"}, http.StatusCreated)
	doJSON(t, client, "POST", api+"/tracking-sets/plan-ts/tickets",
		map[string]any{"id": "task-2", "title": "Task 2"}, http.StatusCreated)

	// Create plan with tasks (overwrites the plan ticket file and adds parent edges).
	doJSON(t, client, "POST", api+"/tracking-sets/plan-ts/plans",
		map[string]any{
			"id": "plan-1", "title": "My Plan",
			"labels": []string{"kind:plan"},
			"tasks":  []string{"task-1", "task-2"},
		}, http.StatusCreated)

	// Get plan -- tasks should be populated via graph edges.
	plan := doJSON(t, client, "GET", api+"/tracking-sets/plan-ts/plans/plan-1", nil, http.StatusOK)
	tasks, ok := plan["tasks"].([]any)
	require.True(t, ok, "tasks should be an array")
	assert.Len(t, tasks, 2)
}

func TestIntegration_MetaPlan(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()
	client := srv.Client()
	base := srv.URL

	api := base + apiPrefix

	doJSON(t, client, "POST", api+"/tracking-sets", map[string]string{"name": "mp-ts"}, http.StatusCreated)

	stages := []types.MetaPlanStage{
		{Name: "stage-1", Status: "pending", Repos: []types.StageRepoRef{{Name: "repo-a", PlanID: "p-1"}}},
	}
	checkpoints := []types.MetaCheckpoint{
		{Name: "gate-1", Stage: "stage-1", Condition: "all done"},
	}

	doJSON(t, client, "POST", api+"/tracking-sets/mp-ts/metaplans",
		map[string]any{
			"id": "mp-1", "title": "Meta Plan",
			"stages": stages, "checkpoints": checkpoints,
		}, http.StatusCreated)

	mp := doJSON(t, client, "GET", api+"/tracking-sets/mp-ts/metaplans/mp-1", nil, http.StatusOK)
	assert.Equal(t, "mp-1", mp["id"])
	assert.Equal(t, "Meta Plan", mp["title"])
}
