package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDataSetWorkflow tests the complete dataset lifecycle:
// create -> validate -> add to dk.yaml -> validate project -> list -> show
func TestDataSetWorkflow(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Step 0: Initialize a data package project
	result, err := runDKInDir(t, tmpDir, "init", "--runtime", "generic-go", "dataset-test")
	if err != nil {
		t.Fatalf("dk init failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk init exited %d: %s", result.ExitCode, result.Stderr)
	}

	projectDir := filepath.Join(tmpDir, "dataset-test")
	assertFileExists(t, filepath.Join(projectDir, "dk.yaml"))

	// Step 1: Create a dataset
	result, err = runDKInDir(t, projectDir, "dataset", "create", "aws-security")
	if err != nil {
		t.Fatalf("dk dataset create failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset create exited %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify dataset file was created
	datasetPath := filepath.Join(projectDir, "datasets", "aws-security", "dataset.yaml")
	assertFileExists(t, datasetPath)
	assertFileContains(t, datasetPath, "aws-security")

	// Verify success message
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "Created dataset") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Step 2: Fill in valid config values
	validDataSet := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: aws-security
spec:
  store: my-s3
  table: security_events
  format: parquet
  classification: internal
`
	if err := os.WriteFile(datasetPath, []byte(validDataSet), 0644); err != nil {
		t.Fatalf("failed to write dataset.yaml: %v", err)
	}

	// Step 3: Validate the single dataset
	result, err = runDKInDir(t, projectDir, "dataset", "validate", "datasets/aws-security/")
	if err != nil {
		t.Fatalf("dk dataset validate failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset validate exited %d: %s\n%s", result.ExitCode, result.Stderr, result.Stdout)
	}
	validateOutput := result.Stdout + result.Stderr
	if !strings.Contains(validateOutput, "valid") {
		t.Errorf("expected validation success, got: %s", validateOutput)
	}

	// Step 4: Add the dataset to dk.yaml
	dkYamlPath := filepath.Join(projectDir, "dk.yaml")
	dkData, err := os.ReadFile(dkYamlPath)
	if err != nil {
		t.Fatalf("failed to read dk.yaml: %v", err)
	}
	dkContent := string(dkData)

	// Append datasets section to the spec
	if !strings.Contains(dkContent, "datasets:") {
		dkContent += "\n  datasets:\n    - aws-security\n"
		if err := os.WriteFile(dkYamlPath, []byte(dkContent), 0644); err != nil {
			t.Fatalf("failed to write dk.yaml: %v", err)
		}
	}

	// Step 5: List datasets
	result, err = runDKInDir(t, projectDir, "dataset", "list")
	if err != nil {
		t.Fatalf("dk dataset list failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset list exited %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify list output contains the dataset
	listOutput := result.Stdout + result.Stderr
	if !strings.Contains(listOutput, "aws-security") {
		t.Errorf("expected dataset list to contain 'aws-security', got: %s", listOutput)
	}

	// Step 6: List datasets as JSON
	result, err = runDKInDir(t, projectDir, "dataset", "list", "--output", "json")
	if err != nil {
		t.Fatalf("dk dataset list --output json failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset list --output json exited %d: %s", result.ExitCode, result.Stderr)
	}

	var jsonList []map[string]any
	jsonListData := result.Stdout
	if jsonListData == "" {
		jsonListData = result.Stderr
	}
	if err := json.Unmarshal([]byte(jsonListData), &jsonList); err != nil {
		t.Fatalf("failed to parse JSON list output: %v\nOutput: %s", err, jsonListData)
	}
	if len(jsonList) != 1 {
		t.Fatalf("expected 1 dataset in JSON list, got %d", len(jsonList))
	}
	if jsonList[0]["name"] != "aws-security" {
		t.Errorf("expected name 'aws-security' in JSON, got: %v", jsonList[0]["name"])
	}

	// Step 7: Show dataset details (YAML)
	result, err = runDKInDir(t, projectDir, "dataset", "show", "aws-security")
	if err != nil {
		t.Fatalf("dk dataset show failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset show exited %d: %s", result.ExitCode, result.Stderr)
	}

	showOutput := result.Stdout + result.Stderr
	if !strings.Contains(showOutput, "aws-security") {
		t.Errorf("expected show output to contain 'aws-security', got: %s", showOutput)
	}

	// Step 8: Show dataset details (JSON)
	result, err = runDKInDir(t, projectDir, "dataset", "show", "aws-security", "--output", "json")
	if err != nil {
		t.Fatalf("dk dataset show --output json failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset show --output json exited %d: %s", result.ExitCode, result.Stderr)
	}

	var jsonShow map[string]any
	jsonShowData := result.Stdout
	if jsonShowData == "" {
		jsonShowData = result.Stderr
	}
	if err := json.Unmarshal([]byte(jsonShowData), &jsonShow); err != nil {
		t.Fatalf("failed to parse JSON show output: %v\nOutput: %s", err, jsonShowData)
	}
	// The show command outputs the full manifest; name is under metadata.
	metadata, _ := jsonShow["metadata"].(map[string]any)
	if metadata == nil || metadata["name"] != "aws-security" {
		t.Errorf("expected metadata.name 'aws-security' in JSON show, got: %v", jsonShow)
	}
}

// TestDataSetCreateInvalidName tests that dataset creation rejects invalid names.
func TestDataSetCreateInvalidName(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Initialize a project first
	result, err := runDKInDir(t, tmpDir, "init", "--runtime", "generic-go", "test-pkg")
	if err != nil {
		t.Fatalf("dk init failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk init exited %d: %s", result.ExitCode, result.Stderr)
	}

	projectDir := filepath.Join(tmpDir, "test-pkg")

	// Try creating a dataset with an invalid name
	result, err = runDKInDir(t, projectDir, "dataset", "create", "AB")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid dataset name")
	}
}

// TestDataSetValidateInvalid tests that validation reports errors for invalid datasets.
func TestDataSetValidateInvalid(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Create a project with an invalid dataset (missing required fields)
	projectDir := filepath.Join(tmpDir, "test-pkg")
	datasetDir := filepath.Join(projectDir, "datasets", "bad-dataset")
	if err := os.MkdirAll(datasetDir, 0755); err != nil {
		t.Fatal(err)
	}

	invalidDataSet := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: bad-dataset
spec:
  store: ""
`
	if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), []byte(invalidDataSet), 0644); err != nil {
		t.Fatal(err)
	}

	// Validate should fail
	result, _ := runDKInDir(t, projectDir, "dataset", "validate", "datasets/bad-dataset/")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid dataset config")
	}
}

// TestDataSetShowNotFound tests that showing a non-existent dataset fails gracefully.
func TestDataSetShowNotFound(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, _ := runDKInDir(t, tmpDir, "dataset", "show", "nonexistent")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent dataset")
	}
}

// TestDataSetListEmpty tests that listing datasets in an empty project is handled gracefully.
func TestDataSetListEmpty(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDKInDir(t, tmpDir, "dataset", "list")
	if err != nil {
		t.Fatalf("dk dataset list failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk dataset list exited %d: %s", result.ExitCode, result.Stderr)
	}

	emptyOutput := result.Stdout + result.Stderr
	if !strings.Contains(emptyOutput, "No datasets found") {
		t.Errorf("expected 'No datasets found' message, got: %s", emptyOutput)
	}
}
