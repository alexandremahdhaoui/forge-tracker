
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

import (
	"fmt"
	"log"
	"net/http"
)

// RunTestCase executes a single test case against the service at serviceURL.
//
//  1. Initializes TemplateData with ServiceURL and an empty Steps map.
//  2. Executes setup steps (fail-fast on error).
//  3. Executes test steps (fail-fast on error, runs assertions from expectedBody).
//  4. Executes teardown steps (always runs, logs errors but does not fail).
//  5. Returns the first error encountered in setup/steps, or nil.
func RunTestCase(client *http.Client, serviceURL string, tc TestCase) error {
	data := &TemplateData{
		ServiceURL: serviceURL,
		Steps:      make(map[string]map[string]interface{}),
	}

	// 2. Setup steps — fail fast on error.
	for i, step := range tc.Setup {
		desc := stepDesc("setup", i, step)
		if _, err := ExecuteStep(client, serviceURL, step, data); err != nil {
			return fmt.Errorf("%s: %w", desc, err)
		}
	}

	// 3. Test steps — fail fast on error, run assertions.
	var testErr error
	for i, step := range tc.Steps {
		desc := stepDesc("step", i, step)

		result, err := ExecuteStep(client, serviceURL, step, data)
		if err != nil {
			testErr = fmt.Errorf("%s: %w", desc, err)
			break
		}

		if step.ExpectedBody != nil {
			if err := AssertResponse(result, step.ExpectedBody); err != nil {
				testErr = fmt.Errorf("%s: assertion failed: %w", desc, err)
				break
			}
		}
	}

	// 4. Teardown steps — always run, log errors but do not fail.
	for i, step := range tc.Teardown {
		desc := stepDesc("teardown", i, step)
		if _, err := ExecuteStep(client, serviceURL, step, data); err != nil {
			log.Printf("WARNING: %s: %v", desc, err)
		}
	}

	return testErr
}

// stepDesc builds a human-readable description for error messages.
func stepDesc(phase string, index int, step Step) string {
	if step.ID != "" {
		return fmt.Sprintf("%s[%d] (%s)", phase, index, step.ID)
	}
	if step.Description != "" {
		return fmt.Sprintf("%s[%d] (%s)", phase, index, step.Description)
	}
	return fmt.Sprintf("%s[%d] %s %s", phase, index, step.Method, step.Path)
}
