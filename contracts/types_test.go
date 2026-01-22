package contracts

import (
	"testing"
	"time"
)

func TestPackageType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		pkgType  PackageType
		wantType string
	}{
		{
			name:     "pipeline",
			pkgType:  PackageTypePipeline,
			wantType: "pipeline",
		},
		{
			name:     "model",
			pkgType:  PackageTypeModel,
			wantType: "model",
		},
		{
			name:     "dataset",
			pkgType:  PackageTypeDataset,
			wantType: "dataset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.pkgType); got != tt.wantType {
				t.Errorf("PackageType = %v, want %v", got, tt.wantType)
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
