//go:build e2e

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
)

const (
	testNamespace = "test"
	serviceURL    = "http://forge-tracker.default:8081"
	jobTimeout    = 5 * time.Minute
)

var (
	k8sClient *kubernetes.Clientset
	lcrFQDN   string
)

// testResult matches the JSON output from the test runner container.
type testResult struct {
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
	Error string `json:"error,omitempty"`
}

func TestMain(m *testing.M) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		fmt.Fprintln(os.Stderr, "KUBECONFIG environment variable is required")
		os.Exit(1)
	}

	lcrFQDN = os.Getenv("TESTENV_LCR_FQDN")
	if lcrFQDN == "" {
		fmt.Fprintln(os.Stderr, "TESTENV_LCR_FQDN environment variable is required")
		os.Exit(1)
	}

	var err error
	k8sClient, err = createK8sClient(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create k8s client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := ensureNamespace(ctx, k8sClient, testNamespace); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure namespace %q: %v\n", testNamespace, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	testdataDir := filepath.Join("testdata")

	var yamlFiles []string
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext == ".yaml" || ext == ".yml" {
			// Store relative path from testdataDir for display and loading.
			relPath, relErr := filepath.Rel(testdataDir, path)
			if relErr != nil {
				return relErr
			}
			yamlFiles = append(yamlFiles, relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk testdata directory: %v", err)
	}

	if len(yamlFiles) == 0 {
		t.Fatal("no YAML test files found in testdata/")
	}

	for _, relPath := range yamlFiles {
		t.Run(relPath, func(t *testing.T) {
			runTestFile(t, testdataDir, relPath)
		})
	}
}

// runTestFile creates a ConfigMap and Job for a single YAML test file, waits
// for the Job to complete, reads the pod logs, and reports results via t.Run.
func runTestFile(t *testing.T, testdataDir, filename string) {
	t.Helper()

	ctx := context.Background()
	resourceName := sanitizeK8sName(filename)
	image := fmt.Sprintf("%s/forge-tracker-test-runner-image:latest", lcrFQDN)

	// Read the YAML file content.
	yamlPath := filepath.Join(testdataDir, filename)
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to read %q: %v", yamlPath, err)
	}

	// Schedule cleanup of k8s resources.
	t.Cleanup(func() {
		cleanupResources(context.Background(), k8sClient, testNamespace, []string{resourceName})
	})

	// Create a ConfigMap with the YAML content.
	if err := createConfigMap(ctx, k8sClient, testNamespace, resourceName, string(yamlContent)); err != nil {
		t.Fatalf("failed to create configmap: %v", err)
	}

	// Create the test runner Job.
	if err := createTestRunnerJob(ctx, k8sClient, testNamespace, resourceName, resourceName, image, serviceURL); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Wait for the Job to complete.
	jobErr := waitForJobCompletion(ctx, k8sClient, testNamespace, resourceName, jobTimeout)

	// Read the pod logs regardless of Job success/failure, because the runner
	// may have produced partial results before exiting with a non-zero code.
	logs, err := getJobPodLogs(ctx, k8sClient, testNamespace, resourceName)
	if err != nil {
		t.Fatalf("failed to get pod logs: %v", err)
	}

	// Parse JSON lines from the logs.
	results := parseResults(t, logs)

	if len(results) == 0 {
		if jobErr != nil {
			t.Fatalf("job failed with no test results: %v\nlogs:\n%s", jobErr, logs)
		}
		t.Fatalf("no test results found in pod logs:\n%s", logs)
	}

	// Report each test case result.
	for _, r := range results {
		t.Run(r.Name, func(t *testing.T) {
			if !r.Pass {
				t.Errorf("test case failed: %s", r.Error)
			}
		})
	}
}

// parseResults extracts testResult entries from JSON lines in the log output.
// Non-JSON lines are silently skipped (the test runner may emit log lines).
func parseResults(t *testing.T, logs string) []testResult {
	t.Helper()

	var results []testResult
	for _, line := range strings.Split(logs, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var r testResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			// Skip non-JSON lines (e.g., log output from the runner).
			continue
		}

		// Only include lines that look like valid test results.
		if r.Name != "" {
			results = append(results, r)
		}
	}

	return results
}
