package pipeline

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var currencyStripRegex = regexp.MustCompile(`[^\d.-]+`)

// StripCurrencyStage removes currency symbols, thousands separators, and spaces
// so dirty financial strings (e.g., "$ 1,250.75" or "USD 99.00") parse cleanly as numbers.
type StripCurrencyStage struct{}

func (StripCurrencyStage) Name() string { return "strip_currency" }
func (StripCurrencyStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if isNullSentinel(trimmed) {
		return "", nil
	}
	return currencyStripRegex.ReplaceAllString(trimmed, ""), nil
}

// AbsNumberStage converts negative numbers into their absolute positive value.
type AbsNumberStage struct{}

func (AbsNumberStage) Name() string { return "abs_number" }
func (AbsNumberStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return "", fmt.Errorf("%w: abs_number requires numeric input, got %q", ErrTypeCastFailed, v)
	}
	absVal := math.Abs(f)
	if !strings.Contains(trimmed, ".") {
		return strconv.FormatInt(int64(absVal), 10), nil
	}
	return strconv.FormatFloat(absVal, 'f', -1, 64), nil
}

// RoundFloatStage rounds floating-point values to a specified number of decimal places.
type RoundFloatStage struct {
	Decimals int
}

func (r RoundFloatStage) Name() string {
	if r.Decimals == 0 {
		return "round_integer"
	}
	return "round_2_decimals"
}

func (r RoundFloatStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return "", fmt.Errorf("%w: round requires numeric input, got %q", ErrTypeCastFailed, v)
	}
	if r.Decimals <= 0 {
		return strconv.FormatInt(int64(math.Round(f)), 10), nil
	}
	pow := math.Pow(10, float64(r.Decimals))
	rounded := math.Round(f*pow) / pow
	return strconv.FormatFloat(rounded, 'f', r.Decimals, 64), nil
}

// EnforceNonNegativeStage rejects negative numeric values with ErrValidationCheck.
type EnforceNonNegativeStage struct{}

func (EnforceNonNegativeStage) Name() string { return "non_negative" }
func (EnforceNonNegativeStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return "", fmt.Errorf("%w: non_negative requires numeric input, got %q", ErrTypeCastFailed, v)
	}
	if f < 0 {
		return "", fmt.Errorf("%w: value %v must be >= 0", ErrValidationCheck, f)
	}
	return trimmed, nil
}

// EnforcePositiveStage rejects zero or negative numeric values with ErrValidationCheck.
type EnforcePositiveStage struct{}

func (EnforcePositiveStage) Name() string { return "positive_only" }
func (EnforcePositiveStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return "", fmt.Errorf("%w: positive_only requires numeric input, got %q", ErrTypeCastFailed, v)
	}
	if f <= 0 {
		return "", fmt.Errorf("%w: value %v must be > 0", ErrValidationCheck, f)
	}
	return trimmed, nil
}
