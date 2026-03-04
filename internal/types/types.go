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

package types

import "time"

// Ticket represents a work item with queryable labels and semantic annotations.
type Ticket struct {
	ID          string            `json:"id"          yaml:"id"`
	Title       string            `json:"title"       yaml:"title"`
	Status      string            `json:"status"      yaml:"status"`
	Priority    int               `json:"priority"    yaml:"priority"`
	Labels      []string          `json:"labels"      yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
	Assignee    string            `json:"assignee"    yaml:"assignee"`
	Created     time.Time         `json:"created"     yaml:"created"`
	Updated     time.Time         `json:"updated"     yaml:"updated"`
	Description string            `json:"description" yaml:"-"`
	Comments    []Comment         `json:"comments"    yaml:"-"`
}

// Priority scale:
//   0 = critical  (drop everything, fix now)
//   1 = high      (next up, blocks progress)
//   2 = medium    (planned work, normal priority)
//   3 = low       (nice to have, backlog)

// Comment represents a timestamped entry by a human or AI agent.
type Comment struct {
	Timestamp time.Time `json:"timestamp"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
}

// Edge represents a directed relationship between two tickets.
type Edge struct {
	From string `json:"from" yaml:"from"`
	To   string `json:"to"   yaml:"to"`
	Type string `json:"type" yaml:"type"`
}

// Graph holds the set of ticket IDs (nodes) and their relationships (edges)
// within a single TrackingSet.
type Graph struct {
	Nodes []string `json:"nodes" yaml:"nodes"`
	Edges []Edge   `json:"edges" yaml:"edges"`
}

// TrackingSet is the tenancy boundary. Each ticket belongs to exactly one
// TrackingSet.
type TrackingSet struct {
	Name  string `json:"name"  yaml:"name"`
	Graph Graph  `json:"graph" yaml:"graph"`
}

// Plan embeds a Ticket and adds child task references derived from graph
// edges of type "parent" where this plan ticket is the parent.
type Plan struct {
	Ticket `json:",inline" yaml:",inline"`
	Tasks  []string `json:"tasks"`
}

// MetaPlan coordinates work across repos.
type MetaPlan struct {
	Ticket      `json:",inline"  yaml:",inline"`
	Stages      []MetaPlanStage  `json:"stages"`
	Checkpoints []MetaCheckpoint `json:"checkpoints"`
}

// MetaPlanStage groups repo-scoped plan references under a named stage.
type MetaPlanStage struct {
	Name   string         `json:"name"`
	Status string         `json:"status"`
	Repos  []StageRepoRef `json:"repos"`
}

// StageRepoRef links a repository name to its plan within a stage.
type StageRepoRef struct {
	Name       string `json:"name"`
	PlanID     string `json:"planId"`
	TasksTotal int    `json:"tasksTotal"`
	TasksDone  int    `json:"tasksDone"`
}

// MetaCheckpoint defines a gate condition that must be met before
// proceeding past a stage.
type MetaCheckpoint struct {
	Name      string `json:"name"`
	Stage     string `json:"stage"`
	Condition string `json:"condition"`
	Met       bool   `json:"met"`
}
