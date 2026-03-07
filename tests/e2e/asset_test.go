package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssetWorkflow tests the complete asset lifecycle:
// create → validate → add to dk.yaml → validate project → list → show
func TestAssetWorkflow(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Step 0: Initialize a data package project
	result, err := runDKInDir(t, tmpDir, "init", "--runtime", "generic-go", "asset-test")
	if err != nil {
		t.Fatalf("dk init failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk init exited %d: %s", result.ExitCode, result.Stderr)
	}

	projectDir := filepath.Join(tmpDir, "asset-test")
	assertFileExists(t, filepath.Join(projectDir, "dk.yaml"))

	// Step 1: Create an asset from an extension
	result, err = runDKInDir(t, projectDir, "asset", "create", "aws-security", "--ext", "cloudquery.source.aws")
	if err != nil {
		t.Fatalf("dk asset create failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset create exited %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify asset file was created
	assetPath := filepath.Join(projectDir, "assets", "sources", "aws-security", "asset.yaml")
	assertFileExists(t, assetPath)
	assertFileContains(t, assetPath, "cloudquery.source.aws")
	assertFileContains(t, assetPath, "aws-security")

	// Verify success message
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "Created asset") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Step 2: Fill in valid config values
	validAsset := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
name: aws-security
type: source
extension: cloudquery.source.aws
version: v24.0.2
ownerTeam: security-data
config:
  accounts:
    - "123456789012"
  regions:
    - us-east-1
  tables:
    - aws_s3_buckets
    - aws_iam_roles
`
	if err := os.WriteFile(assetPath, []byte(validAsset), 0644); err != nil {
		t.Fatalf("failed to write asset.yaml: %v", err)
	}

	// Step 3: Validate the single asset
	result, err = runDKInDir(t, projectDir, "asset", "validate", "assets/sources/aws-security/")
	if err != nil {
		t.Fatalf("dk asset validate failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset validate exited %d: %s\n%s", result.ExitCode, result.Stderr, result.Stdout)
	}
	validateOutput := result.Stdout + result.Stderr
	if !strings.Contains(validateOutput, "valid") {
		t.Errorf("expected validation success, got: %s", validateOutput)
	}

	// Step 4: Add the asset to dk.yaml
	dkYamlPath := filepath.Join(projectDir, "dk.yaml")
	dkData, err := os.ReadFile(dkYamlPath)
	if err != nil {
		t.Fatalf("failed to read dk.yaml: %v", err)
	}
	dkContent := string(dkData)

	// Append assets section to the spec
	if !strings.Contains(dkContent, "assets:") {
		dkContent += "\n  assets:\n    - aws-security\n"
		if err := os.WriteFile(dkYamlPath, []byte(dkContent), 0644); err != nil {
			t.Fatalf("failed to write dk.yaml: %v", err)
		}
	}

	// Step 5: List assets
	result, err = runDKInDir(t, projectDir, "asset", "list")
	if err != nil {
		t.Fatalf("dk asset list failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset list exited %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify list output contains the asset
	listOutput := result.Stdout + result.Stderr
	if !strings.Contains(listOutput, "aws-security") {
		t.Errorf("expected asset list to contain 'aws-security', got: %s", listOutput)
	}
	if !strings.Contains(listOutput, "source") {
		t.Errorf("expected asset list to contain 'source', got: %s", listOutput)
	}
	if !strings.Contains(listOutput, "cloudquery.source.aws") {
		t.Errorf("expected asset list to contain extension FQN, got: %s", listOutput)
	}

	// Step 6: List assets as JSON
	result, err = runDKInDir(t, projectDir, "asset", "list", "--output", "json")
	if err != nil {
		t.Fatalf("dk asset list --output json failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset list --output json exited %d: %s", result.ExitCode, result.Stderr)
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
		t.Fatalf("expected 1 asset in JSON list, got %d", len(jsonList))
	}
	if jsonList[0]["name"] != "aws-security" {
		t.Errorf("expected name 'aws-security' in JSON, got: %v", jsonList[0]["name"])
	}

	// Step 7: Show asset details (YAML)
	result, err = runDKInDir(t, projectDir, "asset", "show", "aws-security")
	if err != nil {
		t.Fatalf("dk asset show failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset show exited %d: %s", result.ExitCode, result.Stderr)
	}

	showOutput := result.Stdout + result.Stderr
	if !strings.Contains(showOutput, "aws-security") {
		t.Errorf("expected show output to contain 'aws-security', got: %s", showOutput)
	}
	if !strings.Contains(showOutput, "cloudquery.source.aws") {
		t.Errorf("expected show output to contain extension, got: %s", showOutput)
	}

	// Step 8: Show asset details (JSON)
	result, err = runDKInDir(t, projectDir, "asset", "show", "aws-security", "--output", "json")
	if err != nil {
		t.Fatalf("dk asset show --output json failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset show --output json exited %d: %s", result.ExitCode, result.Stderr)
	}

	var jsonShow map[string]any
	jsonShowData := result.Stdout
	if jsonShowData == "" {
		jsonShowData = result.Stderr
	}
	if err := json.Unmarshal([]byte(jsonShowData), &jsonShow); err != nil {
		t.Fatalf("failed to parse JSON show output: %v\nOutput: %s", err, jsonShowData)
	}
	if jsonShow["name"] != "aws-security" {
		t.Errorf("expected name 'aws-security' in JSON show, got: %v", jsonShow["name"])
	}
}

// TestAssetCreateInvalidName tests that asset creation rejects invalid names.
func TestAssetCreateInvalidName(t *testing.T) {
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

	// Try creating an asset with an invalid name
	result, err = runDKInDir(t, projectDir, "asset", "create", "AB", "--ext", "cloudquery.source.aws")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid asset name")
	}
}

// TestAssetValidateInvalid tests that validation reports errors for invalid assets.
func TestAssetValidateInvalid(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Create a project with an invalid asset (missing required config)
	projectDir := filepath.Join(tmpDir, "test-pkg")
	assetDir := filepath.Join(projectDir, "assets", "sources", "bad-asset")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	invalidAsset := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
name: bad-asset
type: source
extension: cloudquery.source.aws
version: v1.0.0
ownerTeam: team
config:
  accounts: "not-an-array"
  regions:
    - us-east-1
  tables:
    - aws_s3_buckets
`
	if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), []byte(invalidAsset), 0644); err != nil {
		t.Fatal(err)
	}

	// Validate should fail
	result, _ := runDKInDir(t, projectDir, "asset", "validate", "assets/sources/bad-asset/")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid asset config")
	}
}

// TestAssetShowNotFound tests that showing a non-existent asset fails gracefully.
func TestAssetShowNotFound(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, _ := runDKInDir(t, tmpDir, "asset", "show", "nonexistent")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent asset")
	}
}

// TestAssetListEmpty tests that listing assets in an empty project is handled gracefully.
func TestAssetListEmpty(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDKInDir(t, tmpDir, "asset", "list")
	if err != nil {
		t.Fatalf("dk asset list failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("dk asset list exited %d: %s", result.ExitCode, result.Stderr)
	}

	emptyOutput := result.Stdout + result.Stderr
	if !strings.Contains(emptyOutput, "No assets found") {
		t.Errorf("expected 'No assets found' message, got: %s", emptyOutput)
	}
}
