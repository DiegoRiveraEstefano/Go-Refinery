package pipeline_test

import (
	"testing"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/pipeline"
)

func TestColumnPipeline_Execute(t *testing.T) {
	col := dataset.ColumnSchema{
		Index:          0,
		OriginalName:   "Price",
		NormalizedName: "price",
		InferredType:   dataset.TypeFloat,
		TargetType:     dataset.TypeFloat,
		Nullable:       false,
	}

	regexStage, err := pipeline.BuildStage(dataset.RuleSpec{
		Type:   "regex_replace",
		Params: map[string]string{"pattern": `[^0-9.]`, "replacement": ""},
	})
	if err != nil {
		t.Fatalf("unexpected error building stage: %v", err)
	}

	cp := &pipeline.ColumnPipeline{
		Column: col,
		Stages: []pipeline.Stage{
			pipeline.TrimSpaceStage{},
			regexStage,
		},
	}

	got, err := cp.Execute("  $1,250.75 ")
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	val, ok := got.(float64)
	if !ok || val != 1250.75 {
		t.Fatalf("expected float64(1250.75), got %#v", got)
	}
}
