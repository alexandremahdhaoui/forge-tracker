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
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
)

func TestWriteTicket(t *testing.T) {
	ticket := types.Ticket{
		ID:       "task-001",
		Title:    "Test ticket",
		Status:   "open",
		Priority: 2,
		Labels:   []string{"kind:task"},
		Annotations: map[string]string{
			"note": "test",
		},
		Assignee:    "user-1",
		Created:     time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
		Updated:     time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		Description: "A test ticket.",
		Comments: []types.Comment{
			{
				Timestamp: time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC),
				Author:    "user-1",
				Text:      "First comment.",
			},
		},
	}

	data, err := WriteTicket(ticket)
	if err != nil {
		t.Fatalf("WriteTicket: %v", err)
	}

	s := string(data)

	if !strings.HasPrefix(s, "---\n") {
		t.Error("expected --- prefix")
	}
	if !strings.Contains(s, "id: task-001") {
		t.Errorf("missing id in frontmatter: %s", s)
	}
	if !strings.Contains(s, "## Description") {
		t.Error("missing ## Description")
	}
	if !strings.Contains(s, "A test ticket.") {
		t.Error("missing description text")
	}
	if !strings.Contains(s, "## Comments") {
		t.Error("missing ## Comments")
	}
	if !strings.Contains(s, "### 2026-03-03T11:00:00Z - user-1") {
		t.Error("missing comment header")
	}
	if !strings.Contains(s, "First comment.") {
		t.Error("missing comment text")
	}
}

func TestWriteTicket_RoundTrip(t *testing.T) {
	ticket := types.Ticket{
		ID:       "task-002",
		Title:    "Round trip",
		Status:   "closed",
		Priority: 0,
		Labels:   []string{"a", "b"},
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Description: "Multi-line\ndescription.",
		Comments: []types.Comment{
			{
				Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
				Author:    "bot",
				Text:      "Hello.",
			},
			{
				Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				Author:    "human",
				Text:      "Goodbye.",
			},
		},
	}

	data, err := WriteTicket(ticket)
	if err != nil {
		t.Fatalf("WriteTicket: %v", err)
	}

	parsed, err := ParseTicket(data)
	if err != nil {
		t.Fatalf("ParseTicket: %v", err)
	}

	if parsed.ID != ticket.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, ticket.ID)
	}
	if parsed.Title != ticket.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, ticket.Title)
	}
	if parsed.Description != ticket.Description {
		t.Errorf("Description = %q, want %q", parsed.Description, ticket.Description)
	}
	if len(parsed.Comments) != len(ticket.Comments) {
		t.Fatalf("Comments len = %d, want %d", len(parsed.Comments), len(ticket.Comments))
	}
	for i, c := range parsed.Comments {
		if c.Author != ticket.Comments[i].Author {
			t.Errorf("Comment[%d].Author = %q, want %q", i, c.Author, ticket.Comments[i].Author)
		}
		if c.Text != ticket.Comments[i].Text {
			t.Errorf("Comment[%d].Text = %q, want %q", i, c.Text, ticket.Comments[i].Text)
		}
	}
}

func TestAppendComment(t *testing.T) {
	existing := []byte("---\nid: test\n---\n\n## Description\n\nHello.\n\n## Comments\n")

	comment := types.Comment{
		Timestamp: time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
		Author:    "user",
		Text:      "New comment.",
	}

	result, err := AppendComment(existing, comment)
	if err != nil {
		t.Fatalf("AppendComment: %v", err)
	}

	s := string(result)
	if !strings.Contains(s, "### 2026-03-03T10:00:00Z - user") {
		t.Error("missing appended comment header")
	}
	if !strings.Contains(s, "New comment.") {
		t.Error("missing appended comment text")
	}
}
