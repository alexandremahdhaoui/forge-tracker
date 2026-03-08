
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

package testrunner

// TestFile represents a parsed YAML test case file.
type TestFile struct {
	TestCases []TestCase `yaml:"testCases"`
}

// TestCase represents a single test case with setup, steps, and teardown.
type TestCase struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Setup       []Step   `yaml:"setup,omitempty"`
	Steps       []Step   `yaml:"steps"`
	Teardown    []Step   `yaml:"teardown,omitempty"`
}

// Step represents a single HTTP API call in a test case.
type Step struct {
	ID             string                 `yaml:"id,omitempty"`
	Description    string                 `yaml:"description,omitempty"`
	Method         string                 `yaml:"method"`
	Path           string                 `yaml:"path"`
	Body           map[string]interface{} `yaml:"body,omitempty"`
	ExpectedStatus int                    `yaml:"expectedStatus"`
	ExpectedBody   map[string]interface{} `yaml:"expectedBody,omitempty"`
	Capture        map[string]string      `yaml:"capture,omitempty"`
}

// TemplateData holds the data available for Go template rendering in step fields.
type TemplateData struct {
	ServiceURL string
	Steps      map[string]map[string]interface{}
}
