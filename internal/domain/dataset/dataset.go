// Package dataset contains the core domain entities for ingested files,
// inferred column schemas, data types, and refined records.
package dataset

import (
	"errors"
	"time"
)

// Domain sentinel errors for dataset operations.
var (
	ErrDatasetNotFound   = errors.New("dataset not found")
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrEmptyDataset      = errors.New("dataset is empty")
	ErrInvalidSchema     = errors.New("invalid dataset schema")
)

// ID uniquely identifies a Dataset.
type ID string

// FileFormat represents a supported input file format.
type FileFormat string

const (
	FormatCSV   FileFormat = "CSV"
	FormatXLSX  FileFormat = "XLSX"
	FormatJSON  FileFormat = "JSON"
	FormatJSONL FileFormat = "JSONL"
	FormatLog   FileFormat = "LOG"
)

// Status represents the lifecycle state of a Dataset.
type Status string

const (
	StatusDraft      Status = "DRAFT"
	StatusReady      Status = "READY"
	StatusProcessing Status = "PROCESSING"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

// DataType represents the logical type inferred or configured for a column.
type DataType string

const (
	TypeString   DataType = "STRING"
	TypeInteger  DataType = "INTEGER"
	TypeFloat    DataType = "FLOAT"
	TypeBoolean  DataType = "BOOLEAN"
	TypeDateTime DataType = "DATETIME"
	TypeEmail    DataType = "EMAIL"
	TypeUUID     DataType = "UUID"
	TypeJSON     DataType = "JSON"
)

// Dataset represents the metadata and schema configuration of an uploaded file.
type Dataset struct {
	ID           ID
	OriginalName string
	StoragePath  string
	Format       FileFormat
	Delimiter    rune
	HasHeader    bool
	SizeBytes    int64
	TotalRows    int64
	Status       Status
	Schema       Schema
	SampleRows   [][]string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Schema holds the ordered column definitions of a Dataset.
type Schema struct {
	Columns []ColumnSchema
}

// ColumnSchema defines the structure, inferred metrics, and target rules for a single column.
type ColumnSchema struct {
	Index          int
	OriginalName   string
	NormalizedName string
	InferredType   DataType
	TargetType     DataType
	Nullable       bool
	NullCount      int64
	SampleDistinct int64
	SampleValues   []string
	RuleSpecs      []RuleSpec
}

// RuleSpec is a serializable configuration of a single cleaning or validation stage
// attached to a column.
type RuleSpec struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// Record represents a single validated and normalized row persisted after refinement.
type Record struct {
	DatasetID ID
	RowNumber int64
	Values    map[string]any
	CreatedAt time.Time
}

// Validate checks whether the Dataset and its Schema satisfy domain invariants.
func (d *Dataset) Validate() error {
	if d.ID == "" {
		return errors.New("dataset ID is required")
	}
	if d.OriginalName == "" {
		return errors.New("original file name is required")
	}
	if d.SizeBytes < 0 {
		return errors.New("dataset size cannot be negative")
	}
	switch d.Format {
	case FormatCSV, FormatXLSX, FormatJSON, FormatJSONL, FormatLog:
	default:
		return ErrUnsupportedFormat
	}
	return nil
}
