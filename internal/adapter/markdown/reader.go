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
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"gopkg.in/yaml.v3"
)

// ParseTicket parses a markdown file with YAML frontmatter into a Ticket.
func ParseTicket(data []byte) (types.Ticket, error) {
	s := string(data)

	// Extract YAML frontmatter between --- delimiters.
	frontmatter, body, err := splitFrontmatter(s)
	if err != nil {
		return types.Ticket{}, err
	}

	var ticket types.Ticket
	if err := yaml.Unmarshal([]byte(frontmatter), &ticket); err != nil {
		return types.Ticket{}, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	ticket.Description = parseDescription(body)
	ticket.Comments = parseComments(body)

	return ticket, nil
}

// splitFrontmatter splits content on --- delimiters. Returns frontmatter YAML
// and the remaining body after the closing ---.
func splitFrontmatter(s string) (string, string, error) {
	const delim = "---"

	if !strings.HasPrefix(s, delim+"\n") {
		return "", "", fmt.Errorf("missing opening frontmatter delimiter")
	}

	rest := s[len(delim)+1:]
	idx := strings.Index(rest, "\n"+delim+"\n")
	if idx < 0 {
		return "", "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	frontmatter := rest[:idx]
	body := rest[idx+len("\n"+delim+"\n"):]
	return frontmatter, body, nil
}

// parseDescription extracts text between "## Description" and "## Comments"
// (or EOF).
func parseDescription(body string) string {
	const descHeader = "## Description"
	const commentsHeader = "## Comments"

	idx := strings.Index(body, descHeader)
	if idx < 0 {
		return ""
	}

	after := body[idx+len(descHeader):]

	// Find end: either ## Comments or EOF.
	endIdx := strings.Index(after, commentsHeader)
	var desc string
	if endIdx < 0 {
		desc = after
	} else {
		desc = after[:endIdx]
	}

	// Trim the leading newlines after the header and trailing whitespace.
	desc = strings.TrimLeft(desc, "\n")
	desc = strings.TrimRight(desc, "\n ")

	return desc
}

// parseComments extracts comments from the ## Comments section. Each comment
// starts with ### {RFC3339} - {author}.
func parseComments(body string) []Comment {
	const commentsHeader = "## Comments"
	const commentPrefix = "### "

	idx := strings.Index(body, commentsHeader)
	if idx < 0 {
		return nil
	}

	section := body[idx+len(commentsHeader):]
	lines := strings.Split(section, "\n")

	var comments []Comment
	var current *commentBuilder

	for _, line := range lines {
		if strings.HasPrefix(line, commentPrefix) {
			// Flush previous comment.
			if current != nil {
				comments = append(comments, current.build())
			}
			current = parseCommentHeader(line[len(commentPrefix):])
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		}
	}

	if current != nil {
		comments = append(comments, current.build())
	}

	return comments
}

type Comment = types.Comment

type commentBuilder struct {
	timestamp time.Time
	author    string
	lines     []string
}

func parseCommentHeader(header string) *commentBuilder {
	// Format: {RFC3339} - {author}
	parts := strings.SplitN(header, " - ", 2)
	if len(parts) != 2 {
		return &commentBuilder{}
	}

	ts, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
	return &commentBuilder{
		timestamp: ts,
		author:    strings.TrimSpace(parts[1]),
	}
}

func (b *commentBuilder) build() Comment {
	text := strings.Join(b.lines, "\n")
	text = strings.TrimLeft(text, "\n")
	text = strings.TrimRight(text, "\n ")
	return Comment{
		Timestamp: b.timestamp,
		Author:    b.author,
		Text:      text,
	}
}
