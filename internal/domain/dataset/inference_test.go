package dataset_test

import (
	"testing"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

func TestInferSchema(t *testing.T) {
	headers := []string{"User ID", "Email Address", "Score", "Active", "Created At"}
	rows := [][]string{
		{"101", "alice@example.com", "98.5", "true", "2026-10-24"},
		{"102", "bob@example.com", "84.0", "false", "2026-10-25"},
		{"", "null", "72.25", "1", "N/A"},
	}

	schema := dataset.InferSchema(headers, rows)
	if len(schema.Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(schema.Columns))
	}

	expected := []struct {
		normName string
		dtype    dataset.DataType
		nullable bool
	}{
		{"user_id", dataset.TypeInteger, true},
		{"email_address", dataset.TypeEmail, true},
		{"score", dataset.TypeFloat, false},
		{"active", dataset.TypeBoolean, false},
		{"created_at", dataset.TypeDateTime, true},
	}

	for i, exp := range expected {
		col := schema.Columns[i]
		if col.NormalizedName != exp.normName {
			t.Errorf("col %d: expected normalized name %q, got %q", i, exp.normName, col.NormalizedName)
		}
		if col.InferredType != exp.dtype {
			t.Errorf("col %d (%s): expected inferred type %s, got %s", i, col.NormalizedName, exp.dtype, col.InferredType)
		}
		if col.Nullable != exp.nullable {
			t.Errorf("col %d (%s): expected nullable=%v, got %v", i, col.NormalizedName, exp.nullable, col.Nullable)
		}
	}
}
