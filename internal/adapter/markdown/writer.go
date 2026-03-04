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
	"time"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"gopkg.in/yaml.v3"
)

// WriteTicket serializes a Ticket to markdown with YAML frontmatter.
func WriteTicket(ticket types.Ticket) ([]byte, error) {
	fm, err := yaml.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	buf := []byte("---\n")
	buf = append(buf, fm...)
	buf = append(buf, []byte("---\n")...)

	buf = append(buf, []byte("\n## Description\n\n")...)
	buf = append(buf, []byte(ticket.Description)...)

	buf = append(buf, []byte("\n\n## Comments\n")...)
	for _, c := range ticket.Comments {
		buf = append(buf, formatComment(c)...)
	}

	return buf, nil
}

// AppendComment appends a comment to existing markdown file content without
// rewriting the entire file.
func AppendComment(existing []byte, comment types.Comment) ([]byte, error) {
	buf := make([]byte, len(existing))
	copy(buf, existing)
	buf = append(buf, formatComment(comment)...)
	return buf, nil
}

func formatComment(c types.Comment) []byte {
	return []byte(fmt.Sprintf(
		"\n### %s - %s\n\n%s\n",
		c.Timestamp.Format(time.RFC3339),
		c.Author,
		c.Text,
	))
}
