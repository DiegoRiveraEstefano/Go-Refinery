// Package pipeline defines composable data cleaning, normalization, and validation
// rules executed against raw dataset rows.
package pipeline

import (
	"errors"
	"fmt"
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

var supportedDateLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"2006/01/02",
	time.RFC1123,
}

// Stage represents a single transformation or validation step applied to a cell value.
type Stage interface {
	// Name returns the unique rule identifier (e.g., "trim_space", "strip_currency").
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
			if f, ferr := strconv.ParseFloat(val, 64); ferr == nil {
				return int64(f), nil
			}
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
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "true", "yes", "1", "t", "y", "on", "si", "sí":
			return true, nil
		case "false", "no", "0", "f", "n", "off":
			return false, nil
		default:
			return nil, fmt.Errorf("%w: cannot parse %q as BOOLEAN", ErrTypeCastFailed, val)
		}
	case dataset.TypeDateTime:
		for _, layout := range supportedDateLayouts {
			if t, err := time.Parse(layout, val); err == nil {
				return t.UTC().Format(time.RFC3339), nil
			}
		}
		return nil, fmt.Errorf("%w: cannot parse %q as DATETIME", ErrTypeCastFailed, val)
	default:
		return val, nil
	}
}

func isNullSentinel(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "null", "nil", "n/a", "na", "none", "-", "undefined":
		return true
	default:
		return false
	}
}
