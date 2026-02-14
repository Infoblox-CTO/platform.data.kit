package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineCreate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
		wantFile  bool
		wantOut   string // substring expected in stdout
	}{
		{
			name:     "success - default template",
			args:     []string{"my-pipeline"},
			wantFile: true,
			wantOut:  "Created pipeline",
		},
		{
			name:     "success - sync-only template",
			args:     []string{"my-pipeline", "--template", "sync-only"},
			wantFile: true,
			wantOut:  "Created pipeline",
		},
		{
			name:     "success - custom template",
			args:     []string{"my-pipeline", "--template", "custom"},
			wantFile: true,
			wantOut:  "Created pipeline",
		},
		{
			name:      "invalid template",
			args:      []string{"my-pipeline", "--template", "nonexistent"},
			wantErr:   true,
			errSubstr: "unknown template",
		},
		{
			name: "existing file without force",
			args: []string{"my-pipeline"},
			setup: func(dir string) {
				if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte("existing"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite",
			args: []string{"my-pipeline", "--force"},
			setup: func(dir string) {
				if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte("old"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantFile: true,
			wantOut:  "Created pipeline",
		},
		{
			name:    "missing name",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "list templates",
			args:    []string{"--list-templates"},
			wantOut: "sync-transform-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(tmpDir)
			}

			// Change to temp directory
			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// Reset global flags
			pipelineCreateTemplate = "sync-transform-test"
			pipelineCreateForce = false
			pipelineCreateListTemplates = false

			// Execute command
			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"pipeline", "create"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify output message
			if tt.wantOut != "" {
				output := buf.String()
				if !strings.Contains(output, tt.wantOut) {
					t.Errorf("output %q should contain %q", output, tt.wantOut)
				}
			}

			// Verify file was created
			if tt.wantFile {
				pipelinePath := filepath.Join(tmpDir, "pipeline.yaml")
				if _, err := os.Stat(pipelinePath); os.IsNotExist(err) {
					t.Error("expected pipeline.yaml to be created")
				}
			}
		})
	}
}

func TestPipelineCreateFlags(t *testing.T) {
	// Verify flag registration
	cmd := pipelineCreateCmd

	templateFlag := cmd.Flags().Lookup("template")
	if templateFlag == nil {
		t.Fatal("--template flag not registered")
	}
	if templateFlag.Shorthand != "t" {
		t.Errorf("template shorthand = %q, want %q", templateFlag.Shorthand, "t")
	}
	if templateFlag.DefValue != "sync-transform-test" {
		t.Errorf("template default = %q, want %q", templateFlag.DefValue, "sync-transform-test")
	}

	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("--force flag not registered")
	}

	listFlag := cmd.Flags().Lookup("list-templates")
	if listFlag == nil {
		t.Fatal("--list-templates flag not registered")
	}
}
