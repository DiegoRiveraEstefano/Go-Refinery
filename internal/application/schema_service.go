package application

import (
	"context"
	"fmt"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/pipeline"
	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/internal/ports"
)

// SchemaService manages user modifications to inferred column types and cleaning rules.
type SchemaService struct {
	datasets ports.DatasetRepository
	log      *logger.Logger
}

// NewSchemaService constructs a SchemaService.
func NewSchemaService(datasets ports.DatasetRepository) *SchemaService {
	return &SchemaService{
		datasets: datasets,
		log:      logger.Default().With(logger.F("component", "schema_service")),
	}
}

// SetLogger overrides the default structured logger on SchemaService.
func (s *SchemaService) SetLogger(l *logger.Logger) {
	if l != nil {
		s.log = l.With(logger.F("component", "schema_service"))
	}
}

// UpdateSchema validates all configured pipeline rules in the schema and persists them.
func (s *SchemaService) UpdateSchema(ctx context.Context, datasetID dataset.ID, schema dataset.Schema) error {
	if len(schema.Columns) == 0 {
		return dataset.ErrInvalidSchema
	}

	var totalRules int
	for _, col := range schema.Columns {
		totalRules += len(col.RuleSpecs)
		for _, spec := range col.RuleSpecs {
			if _, err := pipeline.BuildStage(spec); err != nil {
				s.log.WarnContext(ctx, "invalid pipeline rule in schema update",
					logger.F("dataset_id", datasetID),
					logger.F("column", col.NormalizedName),
					logger.F("rule", spec.Type),
					logger.Err(err),
				)
				return fmt.Errorf("column %q rule %q: %w", col.NormalizedName, spec.Type, err)
			}
		}
	}

	if err := s.datasets.SaveSchema(ctx, datasetID, schema); err != nil {
		s.log.ErrorContext(ctx, "failed to save dataset schema",
			logger.F("dataset_id", datasetID),
			logger.Err(err),
		)
		return fmt.Errorf("saving schema for dataset %s: %w", datasetID, err)
	}

	if err := s.datasets.UpdateStatus(ctx, datasetID, dataset.StatusReady, 0); err != nil {
		return err
	}

	s.log.InfoContext(ctx, "dataset schema and pipeline rules updated",
		logger.F("dataset_id", datasetID),
		logger.F("columns", len(schema.Columns)),
		logger.F("total_rules", totalRules),
	)
	return nil
}
