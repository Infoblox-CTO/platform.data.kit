package contracts

import (
	"testing"
)

func TestDataPackage_Fields(t *testing.T) {
	tests := []struct {
		name     string
		pkg      DataPackage
		wantAPI  string
		wantKind string
		wantName string
	}{
		{
			name: "complete package",
			pkg: DataPackage{
				APIVersion: "dp.io/v1alpha1",
				Kind:       "DataPackage",
				Metadata: PackageMetadata{
					Name:      "test-pkg",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: DataPackageSpec{
					Type:        PackageTypePipeline,
					Description: "Test pipeline",
					Owner:       "data-team",
				},
			},
			wantAPI:  "dp.io/v1alpha1",
			wantKind: "DataPackage",
			wantName: "test-pkg",
		},
		{
			name: "minimal package",
			pkg: DataPackage{
				APIVersion: "dp.io/v1alpha1",
				Kind:       "DataPackage",
				Metadata: PackageMetadata{
					Name: "minimal",
				},
			},
			wantAPI:  "dp.io/v1alpha1",
			wantKind: "DataPackage",
			wantName: "minimal",
		},
		{
			name:     "empty package",
			pkg:      DataPackage{},
			wantAPI:  "",
			wantKind: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pkg.APIVersion; got != tt.wantAPI {
				t.Errorf("APIVersion = %v, want %v", got, tt.wantAPI)
			}
			if got := tt.pkg.Kind; got != tt.wantKind {
				t.Errorf("Kind = %v, want %v", got, tt.wantKind)
			}
			if got := tt.pkg.Metadata.Name; got != tt.wantName {
				t.Errorf("Metadata.Name = %v, want %v", got, tt.wantName)
			}
		})
	}
}

func TestPackageMetadata_Labels(t *testing.T) {
	tests := []struct {
		name       string
		metadata   PackageMetadata
		wantLabels int
	}{
		{
			name: "with labels",
			metadata: PackageMetadata{
				Name: "test",
				Labels: map[string]string{
					"team": "analytics",
					"env":  "prod",
				},
			},
			wantLabels: 2,
		},
		{
			name: "no labels",
			metadata: PackageMetadata{
				Name: "test",
			},
			wantLabels: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(tt.metadata.Labels); got != tt.wantLabels {
				t.Errorf("len(Labels) = %v, want %v", got, tt.wantLabels)
			}
		})
	}
}

func TestDataPackageSpec_Type(t *testing.T) {
	tests := []struct {
		name     string
		spec     DataPackageSpec
		wantType PackageType
	}{
		{
			name:     "pipeline type",
			spec:     DataPackageSpec{Type: PackageTypePipeline},
			wantType: PackageTypePipeline,
		},
		{
			name:     "model type",
			spec:     DataPackageSpec{Type: PackageTypeModel},
			wantType: PackageTypeModel,
		},
		{
			name:     "dataset type",
			spec:     DataPackageSpec{Type: PackageTypeDataset},
			wantType: PackageTypeDataset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.Type; got != tt.wantType {
				t.Errorf("Type = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestScheduleSpec(t *testing.T) {
	tests := []struct {
		name        string
		schedule    *ScheduleSpec
		wantCron    string
		wantSuspend bool
	}{
		{
			name: "cron schedule",
			schedule: &ScheduleSpec{
				Cron:     "0 */6 * * *",
				Timezone: "UTC",
				Suspend:  false,
			},
			wantCron:    "0 */6 * * *",
			wantSuspend: false,
		},
		{
			name: "suspended schedule",
			schedule: &ScheduleSpec{
				Cron:    "0 0 * * *",
				Suspend: true,
			},
			wantCron:    "0 0 * * *",
			wantSuspend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.schedule.Cron; got != tt.wantCron {
				t.Errorf("Cron = %v, want %v", got, tt.wantCron)
			}
			if got := tt.schedule.Suspend; got != tt.wantSuspend {
				t.Errorf("Suspend = %v, want %v", got, tt.wantSuspend)
			}
		})
	}
}

func TestResourceSpec(t *testing.T) {
	tests := []struct {
		name       string
		resources  *ResourceSpec
		wantCPU    string
		wantMemory string
	}{
		{
			name: "standard resources",
			resources: &ResourceSpec{
				CPU:    "2",
				Memory: "4Gi",
			},
			wantCPU:    "2",
			wantMemory: "4Gi",
		},
		{
			name: "millicpu resources",
			resources: &ResourceSpec{
				CPU:    "500m",
				Memory: "512Mi",
			},
			wantCPU:    "500m",
			wantMemory: "512Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resources.CPU; got != tt.wantCPU {
				t.Errorf("CPU = %v, want %v", got, tt.wantCPU)
			}
			if got := tt.resources.Memory; got != tt.wantMemory {
				t.Errorf("Memory = %v, want %v", got, tt.wantMemory)
			}
		})
	}
}
