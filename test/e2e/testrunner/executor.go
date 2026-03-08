
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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ExecuteStep executes a single HTTP step against the service.
//
//  1. Renders step.Path and step.Body values through Go templates.
//  2. Builds an HTTP request (method, URL = serviceURL + rendered path, JSON body).
//  3. Sends the request.
//  4. Parses the JSON response body into map[string]interface{}.
//  5. Asserts the status code matches step.ExpectedStatus.
//  6. If step.Capture is defined, extracts values from the response and stores
//     them in data.Steps[step.ID].
//  7. Returns the parsed response body.
func ExecuteStep(client *http.Client, serviceURL string, step Step, data *TemplateData) (map[string]interface{}, error) {
	// 1. Render the path template.
	renderedPath, err := RenderTemplate(step.Path, data)
	if err != nil {
		return nil, fmt.Errorf("rendering path: %w", err)
	}

	// 2. Build the request body.
	var bodyReader io.Reader
	if step.Body != nil {
		renderedBody, err := renderBodyMap(step.Body, data)
		if err != nil {
			return nil, fmt.Errorf("rendering body: %w", err)
		}
		bodyBytes, err := json.Marshal(renderedBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 3. Create and send the HTTP request.
	url := serviceURL + renderedPath
	req, err := http.NewRequest(step.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if step.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request %s %s: %w", step.Method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 4. Parse JSON response body.
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var result map[string]interface{}
	if len(respBytes) > 0 {
		// Try parsing as a JSON object first.
		if err := json.Unmarshal(respBytes, &result); err != nil {
			// If parsing as object fails, try parsing as a JSON array.
			// List endpoints return JSON arrays, which we wrap in a map
			// with "items" and "length" keys so assertions can reference them.
			var arrResult []interface{}
			if arrErr := json.Unmarshal(respBytes, &arrResult); arrErr != nil {
				return nil, fmt.Errorf("parsing response body: %w (body: %s)", err, string(respBytes))
			}
			result = map[string]interface{}{
				"items":  arrResult,
				"length": float64(len(arrResult)),
			}
		}
	}

	// 5. Assert status code.
	if resp.StatusCode != step.ExpectedStatus {
		return result, fmt.Errorf(
			"expected status %d, got %d (body: %s)",
			step.ExpectedStatus, resp.StatusCode, string(respBytes),
		)
	}

	// 6. Capture values from response.
	if step.ID != "" && step.Capture != nil {
		if data.Steps == nil {
			data.Steps = make(map[string]map[string]interface{})
		}
		captured := make(map[string]interface{}, len(step.Capture))
		for varName, fieldName := range step.Capture {
			val, ok := result[fieldName]
			if !ok {
				return result, fmt.Errorf(
					"capture: field %q not found in response for variable %q",
					fieldName, varName,
				)
			}
			captured[varName] = val
		}
		data.Steps[step.ID] = captured
	}

	return result, nil
}
