package schema

import "github.com/Infoblox-CTO/platform.data.kit/contracts"

// ModuleToSchemaFields converts a SchemaModule's associated field definitions
// into DK contracts.SchemaField entries. This is the bridge between APX's
// format-specific schema representation and DK's uniform SchemaField model.
//
// Today this takes explicit field data; when APX exports its validator types
// (e.g., parquetColumn), this function will convert from those types directly.
func ModuleToSchemaFields(fields []FieldDef) []contracts.SchemaField {
	out := make([]contracts.SchemaField, len(fields))
	for i, f := range fields {
		out[i] = contracts.SchemaField{
			Name: f.Name,
			Type: normalizeType(f.Type),
			PII:  f.PII,
			From: f.From,
		}
	}
	return out
}

// SchemaFieldsToFieldDefs converts DK SchemaField entries to FieldDef entries
// for use with schema comparison and breaking change detection.
func SchemaFieldsToFieldDefs(fields []contracts.SchemaField) []FieldDef {
	out := make([]FieldDef, len(fields))
	for i, f := range fields {
		out[i] = FieldDef{
			Name: f.Name,
			Type: f.Type,
			PII:  f.PII,
			From: f.From,
		}
	}
	return out
}

// FieldDef is a format-neutral field definition used as an intermediate
// representation between APX format-specific types and DK SchemaField.
type FieldDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
	PII  bool   `json:"pii,omitempty"`
	From string `json:"from,omitempty"`
}

// normalizeType maps APX/Parquet/Avro type names to DK canonical types.
func normalizeType(t string) string {
	switch t {
	case "INT32", "INT64", "int32", "int64", "int", "long":
		return "integer"
	case "FLOAT", "DOUBLE", "float", "double":
		return "float"
	case "BYTE_ARRAY", "UTF8", "byte_array", "utf8":
		return "string"
	case "BOOLEAN", "boolean", "bool":
		return "boolean"
	case "INT96", "int96", "TIMESTAMP_MILLIS", "TIMESTAMP_MICROS":
		return "timestamp"
	default:
		return t
	}
}
