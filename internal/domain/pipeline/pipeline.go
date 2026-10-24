// Package pipeline defines composable data cleaning, normalization, and validation
// rules executed against raw dataset rows.
package pipeline

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

// Domain errors for pipeline rule execution.
var (
	ErrDropRow         = errors.New("row dropped by pipeline rule")
	ErrRequiredNull    = errors.New("null value in non-nullable column")
	ErrTypeCastFailed  = errors.New("failed to cast value to target data type")
	ErrValidationCheck = errors.New("value failed validation rule")
)

// Stage represents a single transformation or validation step applied to a cell value.
type Stage interface {
	// Name returns the unique rule identifier (e.g., "trim_space", "regex_replace").
	Name() string
	// Apply transforms or validates the input string and returns the resulting string
	// or a domain error.
	Apply(raw string) (string, error)
}

// ColumnPipeline holds the ordered transformation stages and target type for a single column.
type ColumnPipeline struct {
	Column dataset.ColumnSchema
	Stages []Stage
}

// Execute runs all configured stages sequentially and casts the final string into
// the column's TargetType.
func (cp *ColumnPipeline) Execute(raw string) (any, error) {
	val := raw
	for _, stage := range cp.Stages {
		var err error
		val, err = stage.Apply(val)
		if err != nil {
			return nil, fmt.Errorf("stage %s: %w", stage.Name(), err)
		}
	}

	if val == "" {
		if !cp.Column.Nullable {
			return nil, ErrRequiredNull
		}
		return nil, nil
	}

	return castToTargetType(val, cp.Column.TargetType)
}

func castToTargetType(val string, target dataset.DataType) (any, error) {
	switch target {
	case dataset.TypeString, dataset.TypeEmail, dataset.TypeUUID, dataset.TypeJSON:
		return val, nil
	case dataset.TypeInteger:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot parse %q as INTEGER", ErrTypeCastFailed, val)
		}
		return n, nil
	case dataset.TypeFloat:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot parse %q as FLOAT", ErrTypeCastFailed, val)
		}
		return f, nil
	case dataset.TypeBoolean:
		switch strings.ToLower(val) {
		case "true", "yes", "1", "t", "y":
			return true, nil
		case "false", "no", "0", "f", "n":
			return false, nil
		default:
			return nil, fmt.Errorf("%w: cannot parse %q as BOOLEAN", ErrTypeCastFailed, val)
		}
	case dataset.TypeDateTime:
		layouts := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02/01/2006",
			"01/02/2006",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, val); err == nil {
				return t.UTC().Format(time.RFC3339), nil
			}
		}
		return nil, fmt.Errorf("%w: cannot parse %q as DATETIME", ErrTypeCastFailed, val)
	default:
		return val, nil
	}
}

// TrimSpaceStage strips leading and trailing whitespace.
type TrimSpaceStage struct{}

func (TrimSpaceStage) Name() string                   { return "trim_space" }
func (TrimSpaceStage) Apply(v string) (string, error) { return strings.TrimSpace(v), nil }

// CaseStage normalizes text case ("lower" or "upper").
type CaseStage struct {
	Mode string
}

func (c CaseStage) Name() string { return "case_" + c.Mode }
func (c CaseStage) Apply(v string) (string, error) {
	if c.Mode == "upper" {
		return strings.ToUpper(v), nil
	}
	return strings.ToLower(v), nil
}

// DefaultValueStage replaces empty or sentinel null strings with a fallback value.
type DefaultValueStage struct {
	DefaultValue string
}

func (DefaultValueStage) Name() string { return "default_if_null" }
func (d DefaultValueStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	switch strings.ToLower(trimmed) {
	case "", "null", "nil", "n/a", "na", "none", "-":
		return d.DefaultValue, nil
	default:
		return v, nil
	}
}

// RegexReplaceStage replaces matches of a compiled regular expression.
type RegexReplaceStage struct {
	Pattern     *regexp.Regexp
	Replacement string
}

func (RegexReplaceStage) Name() string { return "regex_replace" }
func (r RegexReplaceStage) Apply(v string) (string, error) {
	if r.Pattern == nil {
		return v, nil
	}
	return r.Pattern.ReplaceAllString(v, r.Replacement), nil
}

// BuildStage constructs a concrete Stage from a declarative dataset.RuleSpec.
func BuildStage(spec dataset.RuleSpec) (Stage, error) {
	switch spec.Type {
	case "trim_space":
		return TrimSpaceStage{}, nil
	case "to_lower":
		return CaseStage{Mode: "lower"}, nil
	case "to_upper":
		return CaseStage{Mode: "upper"}, nil
	case "default_if_null":
		return DefaultValueStage{DefaultValue: spec.Params["value"]}, nil
	case "regex_replace":
		re, err := regexp.Compile(spec.Params["pattern"])
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return RegexReplaceStage{
			Pattern:     re,
			Replacement: spec.Params["replacement"],
		}, nil
	default:
		return nil, fmt.Errorf("unknown pipeline rule type %q", spec.Type)
	}
}
