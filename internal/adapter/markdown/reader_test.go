//go:build unit

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
	"testing"
	"time"
)

func TestParseTicket(t *testing.T) {
	input := `---
id: "task-001"
title: "Implement workspace reconciler"
status: "in_progress"
priority: 1
labels:
  - "repo:forge-workspace"
  - "kind:task"
annotations:
  design-doc: ".forge-ai/design"
assignee: "claude-agent-1"
created: "2026-03-03T10:00:00Z"
updated: "2026-03-03T12:00:00Z"
---

## Description

Implement the Workspace reconciler.

## Comments

### 2026-03-03T11:00:00Z - claude-agent-1

Started working on reconciler.

### 2026-03-03T12:00:00Z - claude-agent-1

Completed. Unit tests pass.
`

	ticket, err := ParseTicket([]byte(input))
	if err != nil {
		t.Fatalf("ParseTicket: %v", err)
	}

	if ticket.ID != "task-001" {
		t.Errorf("ID = %q, want %q", ticket.ID, "task-001")
	}
	if ticket.Title != "Implement workspace reconciler" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Implement workspace reconciler")
	}
	if ticket.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", ticket.Status, "in_progress")
	}
	if ticket.Priority != 1 {
		t.Errorf("Priority = %d, want 1", ticket.Priority)
	}
	if len(ticket.Labels) != 2 {
		t.Fatalf("Labels len = %d, want 2", len(ticket.Labels))
	}
	if ticket.Labels[0] != "repo:forge-workspace" {
		t.Errorf("Labels[0] = %q, want %q", ticket.Labels[0], "repo:forge-workspace")
	}
	if ticket.Annotations["design-doc"] != ".forge-ai/design" {
		t.Errorf("Annotations[design-doc] = %q, want %q", ticket.Annotations["design-doc"], ".forge-ai/design")
	}
	if ticket.Assignee != "claude-agent-1" {
		t.Errorf("Assignee = %q, want %q", ticket.Assignee, "claude-agent-1")
	}

	wantCreated := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	if !ticket.Created.Equal(wantCreated) {
		t.Errorf("Created = %v, want %v", ticket.Created, wantCreated)
	}

	if ticket.Description != "Implement the Workspace reconciler." {
		t.Errorf("Description = %q, want %q", ticket.Description, "Implement the Workspace reconciler.")
	}

	if len(ticket.Comments) != 2 {
		t.Fatalf("Comments len = %d, want 2", len(ticket.Comments))
	}

	c0 := ticket.Comments[0]
	if c0.Author != "claude-agent-1" {
		t.Errorf("Comment[0].Author = %q, want %q", c0.Author, "claude-agent-1")
	}
	wantTS := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)
	if !c0.Timestamp.Equal(wantTS) {
		t.Errorf("Comment[0].Timestamp = %v, want %v", c0.Timestamp, wantTS)
	}
	if c0.Text != "Started working on reconciler." {
		t.Errorf("Comment[0].Text = %q, want %q", c0.Text, "Started working on reconciler.")
	}

	c1 := ticket.Comments[1]
	if c1.Text != "Completed. Unit tests pass." {
		t.Errorf("Comment[1].Text = %q, want %q", c1.Text, "Completed. Unit tests pass.")
	}
}

func TestParseTicket_NoComments(t *testing.T) {
	input := `---
id: "task-002"
title: "Simple task"
status: "open"
priority: 2
created: "2026-03-03T10:00:00Z"
updated: "2026-03-03T10:00:00Z"
---

## Description

A task with no comments section.
`

	ticket, err := ParseTicket([]byte(input))
	if err != nil {
		t.Fatalf("ParseTicket: %v", err)
	}

	if ticket.Description != "A task with no comments section." {
		t.Errorf("Description = %q, want %q", ticket.Description, "A task with no comments section.")
	}
	if len(ticket.Comments) != 0 {
		t.Errorf("Comments len = %d, want 0", len(ticket.Comments))
	}
}

func TestParseTicket_MissingFrontmatter(t *testing.T) {
	_, err := ParseTicket([]byte("no frontmatter here"))
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}
