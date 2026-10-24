package pipeline_test

import (
	"errors"
	"testing"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/pipeline"
)

func TestColumnPipeline_Execute(t *testing.T) {
	t.Run("CurrencyToRoundedFloat", func(t *testing.T) {
		col := dataset.ColumnSchema{
			Index:          0,
			OriginalName:   "Price",
			NormalizedName: "price",
			InferredType:   dataset.TypeFloat,
			TargetType:     dataset.TypeFloat,
			Nullable:       false,
		}

		s1, err := pipeline.BuildStage(dataset.RuleSpec{Type: "strip_currency"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s2, err := pipeline.BuildStage(dataset.RuleSpec{Type: "round_2_decimals"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cp := &pipeline.ColumnPipeline{
			Column: col,
			Stages: []pipeline.Stage{s1, s2},
		}

		got, err := cp.Execute("  $1,250.756 ")
		if err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		val, ok := got.(float64)
		if !ok || val != 1250.76 {
			t.Fatalf("expected float64(1250.76), got %#v", got)
		}
	})

	t.Run("TextStages_HTMLTitleAndFormulaSanitization", func(t *testing.T) {
		col := dataset.ColumnSchema{
			Index:          1,
			OriginalName:   "Name",
			NormalizedName: "name",
			TargetType:     dataset.TypeString,
			Nullable:       false,
		}

		stripHTML, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "strip_html"})
		collapseWS, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "collapse_whitespace"})
		titleCase, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "to_title"})

		cp := &pipeline.ColumnPipeline{
			Column: col,
			Stages: []pipeline.Stage{stripHTML, collapseWS, titleCase},
		}

		got, err := cp.Execute("  <b>john    doe</b> ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "John Doe" {
			t.Fatalf("expected 'John Doe', got %q", got)
		}
	})

	t.Run("NumericPositiveValidationFailure", func(t *testing.T) {
		col := dataset.ColumnSchema{
			Index:      2,
			TargetType: dataset.TypeInteger,
			Nullable:   false,
		}
		posStage, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "positive_only"})
		cp := &pipeline.ColumnPipeline{
			Column: col,
			Stages: []pipeline.Stage{posStage},
		}

		_, err := cp.Execute("-5")
		if !errors.Is(err, pipeline.ErrValidationCheck) {
			t.Fatalf("expected ErrValidationCheck for -5, got %v", err)
		}
	})

	t.Run("EmailMaskingAndDropRowIfNull", func(t *testing.T) {
		col := dataset.ColumnSchema{
			Index:      3,
			TargetType: dataset.TypeEmail,
			Nullable:   true,
		}
		dropStage, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "drop_row_if_null"})
		valStage, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "validate_email"})
		maskStage, _ := pipeline.BuildStage(dataset.RuleSpec{Type: "mask_email_user"})

		cp := &pipeline.ColumnPipeline{
			Column: col,
			Stages: []pipeline.Stage{dropStage, valStage, maskStage},
		}

		if _, err := cp.Execute("N/A"); !errors.Is(err, pipeline.ErrDropRow) {
			t.Fatalf("expected ErrDropRow for 'N/A', got %v", err)
		}

		masked, err := cp.Execute("Alice.Smith@Example.COM")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if masked != "a***@example.com" {
			t.Fatalf("expected 'a***@example.com', got %q", masked)
		}
	})

	t.Run("AllAvailableRulesBuildCleanly", func(t *testing.T) {
		for _, desc := range pipeline.AvailableRules() {
			if _, err := pipeline.BuildStage(dataset.RuleSpec{Type: desc.Type}); err != nil {
				t.Errorf("rule %q failed BuildStage: %v", desc.Type, err)
			}
		}
	})
}
