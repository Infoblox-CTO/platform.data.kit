// Package lineage provides OpenLineage event types and emission capabilities
// for tracking data pipeline lineage across environments.
package lineage

import (
	"time"
)

// EventType represents the type of lineage event.
type EventType string

const (
	// EventTypeStart indicates a run has started.
	EventTypeStart EventType = "START"
	// EventTypeRunning indicates a run is in progress.
	EventTypeRunning EventType = "RUNNING"
	// EventTypeComplete indicates a run completed successfully.
	EventTypeComplete EventType = "COMPLETE"
	// EventTypeFail indicates a run failed.
	EventTypeFail EventType = "FAIL"
	// EventTypeAbort indicates a run was aborted.
	EventTypeAbort EventType = "ABORT"
)

// Event represents an OpenLineage event per the OpenLineage spec.
// https://openlineage.io/spec/2-0-2/OpenLineage.json
type Event struct {
	// EventTime is when the event occurred.
	EventTime time.Time `json:"eventTime"`
	// Producer is the URI identifying the producer of this event.
	Producer string `json:"producer"`
	// SchemaURL is the URL of the OpenLineage schema version.
	SchemaURL string `json:"schemaURL,omitempty"`
	// EventType is the type of event (START, COMPLETE, FAIL, etc.).
	EventType EventType `json:"eventType"`
	// Run identifies the run this event is for.
	Run Run `json:"run"`
	// Job identifies the job this event is for.
	Job Job `json:"job"`
	// Inputs are the input datasets for this run.
	Inputs []Dataset `json:"inputs,omitempty"`
	// Outputs are the output datasets for this run.
	Outputs []Dataset `json:"outputs,omitempty"`
}

// Run identifies a specific run of a job.
type Run struct {
	// RunID is a unique identifier for this run.
	RunID string `json:"runId"`
	// Facets are additional metadata about the run.
	Facets map[string]interface{} `json:"facets,omitempty"`
}

// Job identifies a specific job.
type Job struct {
	// Namespace is the namespace containing the job.
	Namespace string `json:"namespace"`
	// Name is the name of the job.
	Name string `json:"name"`
	// Facets are additional metadata about the job.
	Facets map[string]interface{} `json:"facets,omitempty"`
}

// Dataset represents an input or output dataset.
type Dataset struct {
	// Namespace is the namespace containing the dataset.
	Namespace string `json:"namespace"`
	// Name is the name of the dataset.
	Name string `json:"name"`
	// Facets are additional metadata about the dataset.
	Facets map[string]interface{} `json:"facets,omitempty"`
	// InputFacets are input-specific facets.
	InputFacets map[string]interface{} `json:"inputFacets,omitempty"`
	// OutputFacets are output-specific facets.
	OutputFacets map[string]interface{} `json:"outputFacets,omitempty"`
}

// SchemaVersion is the OpenLineage schema version used.
const SchemaVersion = "https://openlineage.io/spec/2-0-2/OpenLineage.json"

// NewEvent creates a new lineage event with common fields populated.
func NewEvent(eventType EventType, runID, namespace, jobName string) *Event {
	return &Event{
		EventTime: time.Now().UTC(),
		Producer:  "dp",
		SchemaURL: SchemaVersion,
		EventType: eventType,
		Run: Run{
			RunID:  runID,
			Facets: make(map[string]interface{}),
		},
		Job: Job{
			Namespace: namespace,
			Name:      jobName,
			Facets:    make(map[string]interface{}),
		},
		Inputs:  []Dataset{},
		Outputs: []Dataset{},
	}
}

// NewDataset creates a new dataset reference.
func NewDataset(namespace, name string) Dataset {
	return Dataset{
		Namespace:    namespace,
		Name:         name,
		Facets:       make(map[string]interface{}),
		InputFacets:  make(map[string]interface{}),
		OutputFacets: make(map[string]interface{}),
	}
}

// AddInput adds an input dataset to the event.
func (e *Event) AddInput(dataset Dataset) {
	e.Inputs = append(e.Inputs, dataset)
}

// AddOutput adds an output dataset to the event.
func (e *Event) AddOutput(dataset Dataset) {
	e.Outputs = append(e.Outputs, dataset)
}

// AddRunFacet adds a facet to the run.
func (e *Event) AddRunFacet(name string, value interface{}) {
	e.Run.Facets[name] = value
}

// AddJobFacet adds a facet to the job.
func (e *Event) AddJobFacet(name string, value interface{}) {
	e.Job.Facets[name] = value
}

// WithDataQualityFacet adds data quality metrics to a dataset.
func (d Dataset) WithDataQualityFacet(rowCount int64, byteCount int64) Dataset {
	d.Facets["dataQualityMetrics"] = map[string]interface{}{
		"rowCount":  rowCount,
		"byteCount": byteCount,
	}
	return d
}

// WithSchemaFacet adds schema information to a dataset.
func (d Dataset) WithSchemaFacet(fields []SchemaField) Dataset {
	d.Facets["schema"] = map[string]interface{}{
		"fields": fields,
	}
	return d
}

// SchemaField represents a field in a dataset schema.
type SchemaField struct {
	// Name is the field name.
	Name string `json:"name"`
	// Type is the field data type.
	Type string `json:"type"`
	// Description is an optional description.
	Description string `json:"description,omitempty"`
}

// DataSourceFacet represents a data source facet for a dataset.
type DataSourceFacet struct {
	// Name is a human-readable name.
	Name string `json:"name,omitempty"`
	// URI is the data source URI.
	URI string `json:"uri"`
}

// WithDataSourceFacet adds data source information to a dataset.
func (d Dataset) WithDataSourceFacet(name, uri string) Dataset {
	d.Facets["dataSource"] = DataSourceFacet{
		Name: name,
		URI:  uri,
	}
	return d
}

// DocumentationFacet represents documentation for a job or dataset.
type DocumentationFacet struct {
	// Description is a description of the job or dataset.
	Description string `json:"description"`
}

// OwnershipFacet represents ownership information.
type OwnershipFacet struct {
	// Owners is the list of owners.
	Owners []Owner `json:"owners"`
}

// Owner represents an owner of a job or dataset.
type Owner struct {
	// Name is the owner's name.
	Name string `json:"name"`
	// Type is the type of owner (e.g., "user", "team", "service").
	Type string `json:"type,omitempty"`
}

// ColumnLineageFacet tracks column-level lineage.
type ColumnLineageFacet struct {
	// Fields maps output field names to their input sources.
	Fields map[string]ColumnLineage `json:"fields"`
}

// ColumnLineage tracks the lineage for a single column.
type ColumnLineage struct {
	// InputFields are the input fields that contribute to this output field.
	InputFields []ColumnReference `json:"inputFields"`
	// TransformationDescription describes how inputs become the output.
	TransformationDescription string `json:"transformationDescription,omitempty"`
	// TransformationType is the type of transformation.
	TransformationType string `json:"transformationType,omitempty"`
}

// ColumnReference references a specific column in a dataset.
type ColumnReference struct {
	// Namespace is the dataset namespace.
	Namespace string `json:"namespace"`
	// Name is the dataset name.
	Name string `json:"name"`
	// Field is the field/column name.
	Field string `json:"field"`
}

// ErrorMessageFacet captures error information for failed runs.
type ErrorMessageFacet struct {
	// Message is the error message.
	Message string `json:"message"`
	// ProgrammingLanguage is the language of the stack trace.
	ProgrammingLanguage string `json:"programmingLanguage,omitempty"`
	// StackTrace is the stack trace if available.
	StackTrace string `json:"stackTrace,omitempty"`
}

// WithErrorFacet adds error information to a run event.
func (e *Event) WithErrorFacet(message, stackTrace string) *Event {
	e.Run.Facets["errorMessage"] = ErrorMessageFacet{
		Message:             message,
		ProgrammingLanguage: "go",
		StackTrace:          stackTrace,
	}
	return e
}

// NominalTimeFacet tracks expected vs actual run times.
type NominalTimeFacet struct {
	// NominalStartTime is the expected start time.
	NominalStartTime time.Time `json:"nominalStartTime"`
	// NominalEndTime is the expected end time.
	NominalEndTime time.Time `json:"nominalEndTime,omitempty"`
}

// WithNominalTimeFacet adds nominal time information to a run event.
func (e *Event) WithNominalTimeFacet(nominalStart, nominalEnd time.Time) *Event {
	e.Run.Facets["nominalTime"] = NominalTimeFacet{
		NominalStartTime: nominalStart,
		NominalEndTime:   nominalEnd,
	}
	return e
}

// ParentRunFacet tracks parent-child run relationships.
type ParentRunFacet struct {
	// Run identifies the parent run.
	Run ParentRun `json:"run"`
	// Job identifies the parent job.
	Job ParentJob `json:"job"`
}

// ParentRun identifies a parent run.
type ParentRun struct {
	// RunID is the parent run ID.
	RunID string `json:"runId"`
}

// ParentJob identifies a parent job.
type ParentJob struct {
	// Namespace is the parent job namespace.
	Namespace string `json:"namespace"`
	// Name is the parent job name.
	Name string `json:"name"`
}

// WithParentFacet adds parent run information to a run event.
func (e *Event) WithParentFacet(parentRunID, parentNamespace, parentJobName string) *Event {
	e.Run.Facets["parent"] = ParentRunFacet{
		Run: ParentRun{RunID: parentRunID},
		Job: ParentJob{Namespace: parentNamespace, Name: parentJobName},
	}
	return e
}
