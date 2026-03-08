
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
	"regexp"
	"strings"
)

// AssertResponse checks that the actual response body matches the expected
// assertions. It supports:
//   - Exact match: field: "value"
//   - Nested object: field: {subfield: "value"}
//   - Array length: field: {length: 2}
//   - Array contains: field: {contains: ["value1"]}
//   - Not empty: field: {notEmpty: true}
//   - Regex: field: {matches: "^[a-z]+-[0-9]+$"}
func AssertResponse(actual map[string]interface{}, expected map[string]interface{}) error {
	return assertMap("", actual, expected)
}

func assertMap(prefix string, actual, expected map[string]interface{}) error {
	for key, expectedVal := range expected {
		fieldPath := joinPath(prefix, key)
		actualVal, ok := actual[key]
		if !ok {
			return fmt.Errorf("field %q: not found in response", fieldPath)
		}

		if err := assertValue(fieldPath, actualVal, expectedVal); err != nil {
			return err
		}
	}
	return nil
}

func assertValue(fieldPath string, actual, expected interface{}) error {
	// If expected is a map, it might be a nested object assertion or a
	// special assertion (length, contains, notEmpty, matches).
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		if isSpecialAssertion(expectedMap) {
			return assertSpecial(fieldPath, actual, expectedMap)
		}

		// Regular nested object comparison.
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Errorf("field %q: expected object, got %T", fieldPath, actual)
		}
		return assertMap(fieldPath, actualMap, expectedMap)
	}

	// Exact match comparison.
	if !valuesEqual(actual, expected) {
		return fmt.Errorf("field %q: expected %v (%T), got %v (%T)",
			fieldPath, expected, expected, actual, actual)
	}

	return nil
}

// isSpecialAssertion returns true if the map contains any of the recognized
// assertion keys.
func isSpecialAssertion(m map[string]interface{}) bool {
	for key := range m {
		switch key {
		case "length", "contains", "notEmpty", "matches":
			return true
		}
	}
	return false
}

func assertSpecial(fieldPath string, actual interface{}, assertions map[string]interface{}) error {
	for key, assertVal := range assertions {
		switch key {
		case "length":
			if err := assertLength(fieldPath, actual, assertVal); err != nil {
				return err
			}
		case "contains":
			if err := assertContains(fieldPath, actual, assertVal); err != nil {
				return err
			}
		case "notEmpty":
			if err := assertNotEmpty(fieldPath, actual, assertVal); err != nil {
				return err
			}
		case "matches":
			if err := assertMatches(fieldPath, actual, assertVal); err != nil {
				return err
			}
		default:
			return fmt.Errorf("field %q: unknown assertion %q", fieldPath, key)
		}
	}
	return nil
}

func assertLength(fieldPath string, actual, expected interface{}) error {
	arr, ok := actual.([]interface{})
	if !ok {
		return fmt.Errorf("field %q (length): expected array, got %T", fieldPath, actual)
	}

	expectedLen, err := toInt(expected)
	if err != nil {
		return fmt.Errorf("field %q (length): %w", fieldPath, err)
	}

	if len(arr) != expectedLen {
		return fmt.Errorf("field %q (length): expected %d, got %d", fieldPath, expectedLen, len(arr))
	}

	return nil
}

func assertContains(fieldPath string, actual, expected interface{}) error {
	arr, ok := actual.([]interface{})
	if !ok {
		return fmt.Errorf("field %q (contains): expected array, got %T", fieldPath, actual)
	}

	expectedItems, ok := expected.([]interface{})
	if !ok {
		return fmt.Errorf("field %q (contains): expected array of values, got %T", fieldPath, expected)
	}

	for _, item := range expectedItems {
		found := false
		for _, elem := range arr {
			if valuesEqual(elem, item) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %q (contains): value %v not found in array", fieldPath, item)
		}
	}

	return nil
}

func assertNotEmpty(fieldPath string, actual, expected interface{}) error {
	shouldBeNotEmpty, ok := expected.(bool)
	if !ok {
		return fmt.Errorf("field %q (notEmpty): expected bool, got %T", fieldPath, expected)
	}

	if !shouldBeNotEmpty {
		return nil
	}

	if actual == nil {
		return fmt.Errorf("field %q (notEmpty): value is nil", fieldPath)
	}

	if str, ok := actual.(string); ok && str == "" {
		return fmt.Errorf("field %q (notEmpty): value is empty string", fieldPath)
	}

	return nil
}

func assertMatches(fieldPath string, actual, expected interface{}) error {
	pattern, ok := expected.(string)
	if !ok {
		return fmt.Errorf("field %q (matches): expected string pattern, got %T", fieldPath, expected)
	}

	actualStr := fmt.Sprintf("%v", actual)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("field %q (matches): invalid regex %q: %w", fieldPath, pattern, err)
	}

	if !re.MatchString(actualStr) {
		return fmt.Errorf("field %q (matches): value %q does not match pattern %q",
			fieldPath, actualStr, pattern)
	}

	return nil
}

// valuesEqual compares two values, handling the fact that JSON/YAML numbers
// may appear as float64 or int.
func valuesEqual(a, b interface{}) bool {
	// Handle numeric comparisons across types.
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	// Fall back to direct comparison.
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

func toInt(v interface{}) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case float32:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case int32:
		return int(n), nil
	default:
		return 0, fmt.Errorf("expected numeric value, got %T (%v)", v, v)
	}
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return strings.Join([]string{prefix, key}, ".")
}
