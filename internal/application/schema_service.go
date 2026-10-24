package application

import (
	"context"
	"fmt"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/pipeline"
	"github.com/diego/go-refinery/internal/ports"
)

// SchemaService manages user modifications to inferred column types and cleaning rules.
type SchemaService struct {
	datasets ports.DatasetRepository
}

// NewSchemaService constructs a SchemaService.
func NewSchemaService(datasets ports.DatasetRepository) *SchemaService {
	return &SchemaService{datasets: datasets}
}

// UpdateSchema validates all configured pipeline rules in the schema and persists them.
func (s *SchemaService) UpdateSchema(ctx context.Context, datasetID dataset.ID, schema dataset.Schema) error {
	if len(schema.Columns) == 0 {
		return dataset.ErrInvalidSchema
	}

	for _, col := range schema.Columns {
		for _, spec := range col.RuleSpecs {
			if _, err := pipeline.BuildStage(spec); err != nil {
				return fmt.Errorf("column %q rule %q: %w", col.NormalizedName, spec.Type, err)
			}
		}
	}

	if err := s.datasets.SaveSchema(ctx, datasetID, schema); err != nil {
		return fmt.Errorf("saving schema for dataset %s: %w", datasetID, err)
	}

	return s.datasets.UpdateStatus(ctx, datasetID, dataset.StatusReady, 0)
}
