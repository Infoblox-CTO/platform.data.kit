package contracts

import (
	"testing"
	"time"
)

func TestKind_Constants(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		wantKind string
	}{
		// New kinds
		{name: "connector", kind: KindConnector, wantKind: "Connector"},
		{name: "store", kind: KindStore, wantKind: "Store"},
		{name: "asset", kind: KindAsset, wantKind: "Asset"},
		{name: "asset-group", kind: KindAssetGroup, wantKind: "AssetGroup"},
		{name: "transform", kind: KindTransform, wantKind: "Transform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.kind); got != tt.wantKind {
				t.Errorf("Kind = %v, want %v", got, tt.wantKind)
			}
		})
	}
}

func TestKind_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		kind  Kind
		valid bool
	}{
		// New kinds
		{name: "connector is valid", kind: KindConnector, valid: true},
		{name: "store is valid", kind: KindStore, valid: true},
		{name: "asset is valid", kind: KindAsset, valid: true},
		{name: "asset-group is valid", kind: KindAssetGroup, valid: true},
		{name: "transform is valid", kind: KindTransform, valid: true},
		// Invalid kinds
		{name: "empty is invalid", kind: "", valid: false},
		{name: "unknown is invalid", kind: Kind("unknown"), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.IsValid(); got != tt.valid {
				t.Errorf("Kind.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestAllKinds(t *testing.T) {
	kinds := AllKinds()
	expected := []Kind{KindConnector, KindStore, KindAsset, KindAssetGroup, KindTransform}
	if len(kinds) != len(expected) {
		t.Fatalf("AllKinds() returned %d kinds, want %d", len(kinds), len(expected))
	}
	for i, k := range kinds {
		if k != expected[i] {
			t.Errorf("AllKinds()[%d] = %v, want %v", i, k, expected[i])
		}
	}
}

func TestRuntime_Constants(t *testing.T) {
	tests := []struct {
		name        string
		runtime     Runtime
		wantRuntime string
	}{
		{name: "cloudquery", runtime: RuntimeCloudQuery, wantRuntime: "cloudquery"},
		{name: "generic-go", runtime: RuntimeGenericGo, wantRuntime: "generic-go"},
		{name: "generic-python", runtime: RuntimeGenericPython, wantRuntime: "generic-python"},
		{name: "dbt", runtime: RuntimeDBT, wantRuntime: "dbt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.runtime); got != tt.wantRuntime {
				t.Errorf("Runtime = %v, want %v", got, tt.wantRuntime)
			}
		})
	}
}

func TestRuntime_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		runtime Runtime
		valid   bool
	}{
		{name: "cloudquery is valid", runtime: RuntimeCloudQuery, valid: true},
		{name: "generic-go is valid", runtime: RuntimeGenericGo, valid: true},
		{name: "generic-python is valid", runtime: RuntimeGenericPython, valid: true},
		{name: "dbt is valid", runtime: RuntimeDBT, valid: true},
		{name: "empty is invalid", runtime: "", valid: false},
		{name: "unknown is invalid", runtime: Runtime("unknown"), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.runtime.IsValid(); got != tt.valid {
				t.Errorf("Runtime.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestRunStatus_Constants(t *testing.T) {
	tests := []struct {
		name       string
		status     RunStatus
		wantStatus string
	}{
		{
			name:       "pending",
			status:     RunStatusPending,
			wantStatus: "pending",
		},
		{
			name:       "running",
			status:     RunStatusRunning,
			wantStatus: "running",
		},
		{
			name:       "completed",
			status:     RunStatusCompleted,
			wantStatus: "completed",
		},
		{
			name:       "failed",
			status:     RunStatusFailed,
			wantStatus: "failed",
		},
		{
			name:       "cancelled",
			status:     RunStatusCancelled,
			wantStatus: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.status); got != tt.wantStatus {
				t.Errorf("RunStatus = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestRunTrigger_Constants(t *testing.T) {
	tests := []struct {
		name        string
		trigger     RunTrigger
		wantTrigger string
	}{
		{
			name:        "schedule",
			trigger:     RunTriggerSchedule,
			wantTrigger: "schedule",
		},
		{
			name:        "event",
			trigger:     RunTriggerEvent,
			wantTrigger: "event",
		},
		{
			name:        "manual",
			trigger:     RunTriggerManual,
			wantTrigger: "manual",
		},
		{
			name:        "promotion",
			trigger:     RunTriggerPromotion,
			wantTrigger: "promotion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.trigger); got != tt.wantTrigger {
				t.Errorf("RunTrigger = %v, want %v", got, tt.wantTrigger)
			}
		})
	}
}

func TestRunRecord_Fields(t *testing.T) {
	now := time.Now()
	end := now.Add(5 * time.Minute)

	tests := []struct {
		name       string
		record     RunRecord
		wantID     string
		wantStatus RunStatus
		wantEnv    string
	}{
		{
			name: "completed run",
			record: RunRecord{
				ID: "run-123",
				PackageRef: ArtifactRef{
					Name:    "my-pkg",
					Version: "1.0.0",
				},
				Environment:      "dev",
				Status:           RunStatusCompleted,
				Trigger:          RunTriggerManual,
				StartTime:        now,
				EndTime:          &end,
				RecordsProcessed: 1000,
			},
			wantID:     "run-123",
			wantStatus: RunStatusCompleted,
			wantEnv:    "dev",
		},
		{
			name: "failed run",
			record: RunRecord{
				ID:           "run-456",
				Environment:  "prod",
				Status:       RunStatusFailed,
				Trigger:      RunTriggerSchedule,
				StartTime:    now,
				ErrorMessage: "out of memory",
			},
			wantID:     "run-456",
			wantStatus: RunStatusFailed,
			wantEnv:    "prod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.ID; got != tt.wantID {
				t.Errorf("ID = %v, want %v", got, tt.wantID)
			}
			if got := tt.record.Status; got != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got, tt.wantStatus)
			}
			if got := tt.record.Environment; got != tt.wantEnv {
				t.Errorf("Environment = %v, want %v", got, tt.wantEnv)
			}
		})
	}
}

func TestRunRecord_Duration(t *testing.T) {
	now := time.Now()
	end := now.Add(5 * time.Minute)

	record := RunRecord{
		ID:        "run-123",
		StartTime: now,
		EndTime:   &end,
	}

	if record.EndTime == nil {
		t.Fatal("EndTime should not be nil")
	}

	duration := record.EndTime.Sub(record.StartTime)
	if duration != 5*time.Minute {
		t.Errorf("Duration = %v, want 5m", duration)
	}
}

func TestMode_Constants(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		wantMode string
	}{
		{
			name:     "batch",
			mode:     ModeBatch,
			wantMode: "batch",
		},
		{
			name:     "streaming",
			mode:     ModeStreaming,
			wantMode: "streaming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.mode); got != tt.wantMode {
				t.Errorf("Mode = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

func TestMode_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		mode  Mode
		valid bool
	}{
		{
			name:  "batch is valid",
			mode:  ModeBatch,
			valid: true,
		},
		{
			name:  "streaming is valid",
			mode:  ModeStreaming,
			valid: true,
		},
		{
			name:  "empty is invalid (use Default() to get batch)",
			mode:  "",
			valid: false,
		},
		{
			name:  "invalid mode",
			mode:  Mode("invalid"),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.IsValid(); got != tt.valid {
				t.Errorf("Mode.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestMode_Default(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		wantMode Mode
	}{
		{
			name:     "empty defaults to batch",
			mode:     "",
			wantMode: ModeBatch,
		},
		{
			name:     "batch stays batch",
			mode:     ModeBatch,
			wantMode: ModeBatch,
		},
		{
			name:     "streaming stays streaming",
			mode:     ModeStreaming,
			wantMode: ModeStreaming,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.Default(); got != tt.wantMode {
				t.Errorf("Mode.Default() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}
