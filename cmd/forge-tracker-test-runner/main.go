
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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alexandremahdhaoui/forge-tracker/test/e2e/testrunner"
)

type result struct {
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
	Error string `json:"error,omitempty"`
}

func main() {
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		log.Fatal("SERVICE_URL environment variable is required")
	}

	testdataDir := "/testdata"
	if dir := os.Getenv("TESTDATA_DIR"); dir != "" {
		testdataDir = dir
	}

	testFiles, err := testrunner.LoadTestFiles(testdataDir)
	if err != nil {
		log.Fatalf("failed to load test files: %v", err)
	}

	client := &http.Client{}
	var failed bool

	for _, tf := range testFiles {
		for _, tc := range tf.TestCases {
			err := testrunner.RunTestCase(client, serviceURL, tc)
			r := result{Name: tc.Name, Pass: err == nil}
			if err != nil {
				r.Error = err.Error()
				failed = true
			}
			data, _ := json.Marshal(r)
			fmt.Println(string(data))
		}
	}

	if failed {
		os.Exit(1)
	}
}
