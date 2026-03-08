
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
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadTestFiles finds *.yaml files in the given directory and its immediate
// subdirectories, then parses each into a TestFile. Go's filepath.Glob does
// not support recursive ** globbing, so this function searches at depth 0 and
// depth 1 only. Returns the parsed test files or an error.
func LoadTestFiles(dir string) ([]TestFile, error) {
	pattern := filepath.Join(dir, "**", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing %q: %w", pattern, err)
	}

	// filepath.Glob with ** does not recurse into subdirectories on all
	// platforms. Also glob the top-level directory to catch files that are
	// not nested.
	topLevel := filepath.Join(dir, "*.yaml")
	topMatches, err := filepath.Glob(topLevel)
	if err != nil {
		return nil, fmt.Errorf("globbing %q: %w", topLevel, err)
	}

	// Deduplicate matches.
	seen := make(map[string]struct{}, len(matches)+len(topMatches))
	var allPaths []string
	for _, m := range append(topMatches, matches...) {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		allPaths = append(allPaths, m)
	}

	if len(allPaths) == 0 {
		return nil, fmt.Errorf("no YAML files found in %q", dir)
	}

	var testFiles []TestFile
	for _, path := range allPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", path, err)
		}

		var tf TestFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			return nil, fmt.Errorf("parsing %q: %w", path, err)
		}

		testFiles = append(testFiles, tf)
	}

	return testFiles, nil
}
