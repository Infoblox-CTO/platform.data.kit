package runner

import (
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestBindingToEnvVar(t *testing.T) {
	tests := []struct {
		name        string
		bindingPath string
		want        string
	}{
		{
			name:        "input binding with property",
			bindingPath: "input.events.brokers",
			want:        "INPUT_EVENTS_BROKERS",
		},
		{
			name:        "output binding with property",
			bindingPath: "output.lake.bucket",
			want:        "OUTPUT_LAKE_BUCKET",
		},
		{
			name:        "simple binding",
			bindingPath: "kafka.topic",
			want:        "KAFKA_TOPIC",
		},
		{
			name:        "single word",
			bindingPath: "single",
			want:        "SINGLE",
		},
		{
			name:        "deep path",
			bindingPath: "input.events.kafka.consumer.group",
			want:        "INPUT_EVENTS_KAFKA_CONSUMER_GROUP",
		},
		{
			name:        "mixed case input",
			bindingPath: "Input.Events.Brokers",
			want:        "INPUT_EVENTS_BROKERS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BindingToEnvVar(tt.bindingPath); got != tt.want {
				t.Errorf("BindingToEnvVar(%q) = %q, want %q", tt.bindingPath, got, tt.want)
			}
		})
	}
}

func TestMapBindingsToEnvVars(t *testing.T) {
	tests := []struct {
		name         string
		pkg          *contracts.DataPackage
		bindings     []contracts.Binding
		wantEnvCount int
		wantWarnings int
	}{
		{
			name: "kafka input and s3 output mapping",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Inputs: []contracts.ArtifactContract{
						{Name: "events", Binding: "input.events"},
					},
					Outputs: []contracts.ArtifactContract{
						{Name: "lake", Binding: "output.lake"},
					},
				},
			},
			bindings: []contracts.Binding{
				{
					Name: "input.events",
					Type: contracts.BindingTypeKafkaTopic,
					Kafka: &contracts.KafkaTopicBinding{
						Topic:   "events-topic",
						Brokers: []string{"localhost:9092"},
					},
				},
				{
					Name: "output.lake",
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "my-bucket",
						Prefix: "data/",
					},
				},
			},
			wantEnvCount: 4, // 2 from kafka (topic, brokers) + 2 from s3 (bucket, prefix)
			wantWarnings: 0,
		},
		{
			name: "missing binding",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Inputs: []contracts.ArtifactContract{
						{Name: "events", Binding: "input.events"},
					},
				},
			},
			bindings:     []contracts.Binding{},
			wantEnvCount: 0,
			wantWarnings: 1, // "binding not found"
		},
		{
			name: "empty binding name",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Inputs: []contracts.ArtifactContract{
						{Name: "events", Binding: ""},
					},
				},
			},
			bindings:     []contracts.Binding{},
			wantEnvCount: 0,
			wantWarnings: 0,
		},
		{
			name: "postgres binding",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Outputs: []contracts.ArtifactContract{
						{Name: "db", Binding: "output.db"},
					},
				},
			},
			bindings: []contracts.Binding{
				{
					Name: "output.db",
					Type: contracts.BindingTypePostgresTable,
					Postgres: &contracts.PostgresTableBinding{
						Host:     "localhost",
						Port:     5432,
						Database: "mydb",
						Table:    "events",
					},
				},
			},
			wantEnvCount: 4, // host, port, database, table
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, warnings := MapBindingsToEnvVars(tt.pkg, tt.bindings)
			if len(result) != tt.wantEnvCount {
				t.Errorf("MapBindingsToEnvVars() returned %d env vars, want %d", len(result), tt.wantEnvCount)
			}
			if len(warnings) != tt.wantWarnings {
				t.Errorf("MapBindingsToEnvVars() returned %d warnings, want %d: %v", len(warnings), tt.wantWarnings, warnings)
			}
		})
	}
}

func TestEnvVarsFromRuntime(t *testing.T) {
	tests := []struct {
		name    string
		runtime *contracts.RuntimeSpec
		want    map[string]string
	}{
		{
			name: "with env vars",
			runtime: &contracts.RuntimeSpec{
				Image: "test:latest",
				Env: []contracts.EnvVar{
					{Name: "LOG_LEVEL", Value: "debug"},
					{Name: "DEBUG", Value: "true"},
				},
			},
			want: map[string]string{
				"LOG_LEVEL": "debug",
				"DEBUG":     "true",
			},
		},
		{
			name:    "nil runtime",
			runtime: nil,
			want:    map[string]string{},
		},
		{
			name: "empty env",
			runtime: &contracts.RuntimeSpec{
				Image: "test:latest",
				Env:   []contracts.EnvVar{},
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvVarsFromRuntime(tt.runtime)
			if len(got) != len(tt.want) {
				t.Errorf("EnvVarsFromRuntime() returned %d vars, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("EnvVarsFromRuntime()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestMergeEnvVars(t *testing.T) {
	tests := []struct {
		name         string
		bindingEnvs  []BindingProperty
		explicitEnvs map[string]string
		wantCount    int
		wantOverride string // key to check for override
		wantValue    string // expected value after override
	}{
		{
			name: "explicit overrides binding",
			bindingEnvs: []BindingProperty{
				{EnvVar: "INPUT_EVENTS_BROKERS", Value: "localhost:9092"},
				{EnvVar: "LOG_LEVEL", Value: "info"},
			},
			explicitEnvs: map[string]string{
				"LOG_LEVEL": "debug", // should override
			},
			wantCount:    2,
			wantOverride: "LOG_LEVEL",
			wantValue:    "debug",
		},
		{
			name: "no overlap",
			bindingEnvs: []BindingProperty{
				{EnvVar: "INPUT_EVENTS_BROKERS", Value: "localhost:9092"},
			},
			explicitEnvs: map[string]string{
				"LOG_LEVEL": "debug",
			},
			wantCount:    2,
			wantOverride: "LOG_LEVEL",
			wantValue:    "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeEnvVars(tt.bindingEnvs, tt.explicitEnvs)
			if len(got) != tt.wantCount {
				t.Errorf("MergeEnvVars() returned %d vars, want %d", len(got), tt.wantCount)
			}
			if got[tt.wantOverride] != tt.wantValue {
				t.Errorf("MergeEnvVars()[%q] = %q, want %q", tt.wantOverride, got[tt.wantOverride], tt.wantValue)
			}
		})
	}
}
