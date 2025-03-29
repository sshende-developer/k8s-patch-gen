// Copyright (c) 2024 Swanand Shende
// 
// This file is part of k8s-patch-gen.
//
// k8s-patch-gen is licensed under the MIT License.
// You may obtain a copy of the license at:
//
//     https://opensource.org/licenses/MIT
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/stretchr/testify/assert"
)

// Test detectGroupResource function
func TestDetectGroupResource(t *testing.T) {
	tests := []struct {
		name           string
		manifestData   map[string]interface{}
		expectedOutput string
	}{
		{"Valid API Version", map[string]interface{}{"apiVersion": "v1/apps", "kind": "Deployment"}, "apps.v1"},
		{"Only Kind", map[string]interface{}{"kind": "Pod"}, "pod"},
		{"Unknown Resource", map[string]interface{}{}, "unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := detectGroupResource(test.manifestData)
			if output != test.expectedOutput {
				t.Errorf("Expected %s but got %s", test.expectedOutput, output)
			}
		})
	}
}

// Test extractPathFromLine function
func TestExtractPathFromLine(t *testing.T) {
	// Create a test manifest file
	testFile := "test_manifest.yaml"
	content := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: my-app
spec:
  containers:
    - name: nginx
      image: nginx:latest`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile) // Cleanup after test

	tests := []struct {
		name         string
		lineNumber   int
		expectedPath string
	}{
		{"Metadata Name", 4, "/metadata/name"},
		{"Labels Key", 5, "/metadata/labels"},
		{"Containers List", 8, "/spec/containers"},
		{"Invalid Line", 20, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := extractPathFromLine(testFile, test.lineNumber)
			if output != test.expectedPath {
				t.Errorf("Expected %s but got %s", test.expectedPath, output)
			}
		})
	}
}

// Test printManifestWithLineNumbers function
func TestPrintManifestWithLineNumbers(t *testing.T) {
	// Create a sample manifest file
	testFile := "test_manifest.yaml"
	content := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile) // Cleanup after test

	// Capture printed output
	output := captureOutput(func() {
		printManifestWithLineNumbers(testFile)
	})

	// Check if line numbers appear
	expectedLines := []string{
		"1: apiVersion: v1",
		"2: kind: Pod",
		"3: metadata:",
		"4:   name: test-pod",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s", expected)
		}
	}
}

// Helper function to capture console output
func captureOutput(f func()) string {
	old := os.Stdout // Save the original stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f() // Run the function that produces output

	w.Close()
	os.Stdout = old // Restore original stdout

	var buf bytes.Buffer
	io.Copy(&buf, r) // Correct way to read from pipe
	return buf.String()
}

// Mock function to simulate survey responses
func mockSurveyAskOne(responses []string) func(survey.Prompt, interface{}, ...survey.AskOpt) error {
	index := 0
	return func(prompt survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
		if index >= len(responses) {
			return nil
		}

		switch r := response.(type) {
		case *string:
			*r = responses[index]
		case *bool:
			*r = responses[index] == "true"
		}
		index++
		return nil
	}
}

func TestBuildNestedJSON(t *testing.T) {
	tests := []struct {
		name           string
		mockResponses  []string
		expectedOutput string
	}{
		{
			name: "Simple key-value pairs",
			mockResponses: []string{
				"title", "String", "MyApp",
				"version", "Number", "1.0",
				"isActive", "Boolean", "true",
				"", // End input
			},
			expectedOutput: `{"title":"MyApp","version":1.0,"isActive":true}`,
		},
		{
			name: "Nested JSON object",
			mockResponses: []string{
				"metadata", "Object", // Creating a nested object
				"author", "String", "Alice",
				"stars", "Number", "5",
				"", // End nested object
				"isPublished", "Boolean", "true",
				"", // End input
			},
			expectedOutput: `{"metadata":{"author":"Alice","stars":5},"isPublished":true}`,
		},
		{
			name:           "User exits immediately (Empty JSON)",
			mockResponses:  []string{""}, // No input, just exits
			expectedOutput: `{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockAskOne := mockSurveyAskOne(test.mockResponses)
			outputJSON := buildNestedJSON(mockAskOne)

			// Validate JSON formatting
			var expectedMap, outputMap map[string]interface{}
			err1 := json.Unmarshal([]byte(test.expectedOutput), &expectedMap)
			err2 := json.Unmarshal([]byte(outputJSON), &outputMap)

			assert.NoError(t, err1, "Failed to parse expected JSON")
			assert.NoError(t, err2, "Failed to parse output JSON")
			assert.Equal(t, expectedMap, outputMap, "Generated JSON does not match expected output")
		})
	}
}
