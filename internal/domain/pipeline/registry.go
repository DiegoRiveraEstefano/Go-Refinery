package pipeline

import (
	"fmt"
	"regexp"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

// RuleCategory classifies which column data types a rule applies to in the UI and engine.
type RuleCategory string

const (
	CategoryCommon   RuleCategory = "COMMON"
	CategoryText     RuleCategory = "TEXT"
	CategoryNumeric  RuleCategory = "NUMERIC"
	CategoryBoolean  RuleCategory = "BOOLEAN"
	CategoryDateTime RuleCategory = "DATETIME"
	CategoryEmail    RuleCategory = "EMAIL"
	CategoryUUID     RuleCategory = "UUID"
	CategoryJSON     RuleCategory = "JSON"
)

// RuleDescriptor holds metadata for rendering and validating pipeline rules per column DataType.
type RuleDescriptor struct {
	Type           string             `json:"type"`
	Label          string             `json:"label"`
	Description    string             `json:"description"`
	Category       RuleCategory       `json:"category"`
	SupportedTypes []dataset.DataType `json:"supported_types"`
}

var allDataTypes = []dataset.DataType{
	dataset.TypeString,
	dataset.TypeInteger,
	dataset.TypeFloat,
	dataset.TypeBoolean,
	dataset.TypeDateTime,
	dataset.TypeEmail,
	dataset.TypeUUID,
	dataset.TypeJSON,
}

// AvailableRules returns the ordered catalog of pipeline rules and the DataTypes each supports.
func AvailableRules() []RuleDescriptor {
	return []RuleDescriptor{
		// Common rules (applicable to all column types)
		{
			Type:           "trim_space",
			Label:          "Trim Whitespace",
			Description:    "Strip leading and trailing spaces",
			Category:       CategoryCommon,
			SupportedTypes: allDataTypes,
		},
		{
			Type:           "null_if_empty",
			Label:          "Normalize Nulls",
			Description:    "Convert 'null', 'N/A', '-' into NULL",
			Category:       CategoryCommon,
			SupportedTypes: allDataTypes,
		},
		{
			Type:           "drop_row_if_null",
			Label:          "Drop Row if Null",
			Description:    "Discard the entire row when this cell is empty",
			Category:       CategoryCommon,
			SupportedTypes: allDataTypes,
		},

		// Text rules (STRING)
		{
			Type:           "collapse_whitespace",
			Label:          "Collapse Spaces",
			Description:    "Replace multiple spaces/tabs with a single space",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},
		{
			Type:           "to_lower",
			Label:          "Lowercase",
			Description:    "Convert text to lowercase",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString, dataset.TypeEmail, dataset.TypeUUID},
		},
		{
			Type:           "to_upper",
			Label:          "Uppercase",
			Description:    "Convert text to UPPERCASE",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},
		{
			Type:           "to_title",
			Label:          "Title Case",
			Description:    "Capitalize the first letter of each word",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},
		{
			Type:           "strip_html",
			Label:          "Strip HTML Tags",
			Description:    "Remove inline HTML/XML markup tags",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},
		{
			Type:           "remove_non_printable",
			Label:          "Strip Non-Printable",
			Description:    "Remove control characters and non-printable runes",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},
		{
			Type:           "extract_digits",
			Label:          "Extract Digits Only",
			Description:    "Keep only 0-9 digits (useful for phone numbers/IDs)",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString, dataset.TypeInteger},
		},
		{
			Type:           "sanitize_csv_formula",
			Label:          "Escape CSV Formulas",
			Description:    "Prefix cells starting with =, +, -, @ with a quote",
			Category:       CategoryText,
			SupportedTypes: []dataset.DataType{dataset.TypeString},
		},

		// Numeric rules (INTEGER, FLOAT)
		{
			Type:           "strip_currency",
			Label:          "Strip Currency & Commas",
			Description:    "Clean symbols like '$1,250.50' before numeric cast",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},
		{
			Type:           "default_zero_if_null",
			Label:          "Default 0 if Null",
			Description:    "Replace empty/null cells with 0",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},
		{
			Type:           "abs_number",
			Label:          "Absolute Value (|x|)",
			Description:    "Convert negative numbers to positive",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},
		{
			Type:           "non_negative",
			Label:          "Require >= 0",
			Description:    "Reject row if number is negative",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},
		{
			Type:           "positive_only",
			Label:          "Require > 0",
			Description:    "Reject row if number is zero or negative",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},
		{
			Type:           "round_2_decimals",
			Label:          "Round (2 Decimals)",
			Description:    "Round floating-point numbers to 2 decimal places",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeFloat},
		},
		{
			Type:           "round_integer",
			Label:          "Round to Nearest Int",
			Description:    "Round decimal numbers to the nearest integer",
			Category:       CategoryNumeric,
			SupportedTypes: []dataset.DataType{dataset.TypeInteger, dataset.TypeFloat},
		},

		// Boolean rules (BOOLEAN)
		{
			Type:           "default_false_if_null",
			Label:          "Default false if Null",
			Description:    "Replace empty/null cells with false",
			Category:       CategoryBoolean,
			SupportedTypes: []dataset.DataType{dataset.TypeBoolean},
		},

		// DateTime rules (DATETIME)
		{
			Type:           "normalize_iso_date",
			Label:          "Format YYYY-MM-DD",
			Description:    "Normalize dates to ISO-8601 calendar date",
			Category:       CategoryDateTime,
			SupportedTypes: []dataset.DataType{dataset.TypeDateTime},
		},

		// Email rules (EMAIL)
		{
			Type:           "validate_email",
			Label:          "Strict RFC-5322 Email",
			Description:    "Validate email structure and lowercase address",
			Category:       CategoryEmail,
			SupportedTypes: []dataset.DataType{dataset.TypeEmail},
		},
		{
			Type:           "mask_email_user",
			Label:          "Mask Email Username",
			Description:    "Anonymize email user (e.g. a***@domain.com)",
			Category:       CategoryEmail,
			SupportedTypes: []dataset.DataType{dataset.TypeEmail, dataset.TypeString},
		},

		// UUID rules (UUID)
		{
			Type:           "validate_uuid",
			Label:          "Strict UUID Format",
			Description:    "Validate 36-char UUID and normalize to lowercase",
			Category:       CategoryUUID,
			SupportedTypes: []dataset.DataType{dataset.TypeUUID},
		},

		// JSON rules (JSON)
		{
			Type:           "compact_json",
			Label:          "Validate & Compact JSON",
			Description:    "Verify valid JSON syntax and minify whitespace",
			Category:       CategoryJSON,
			SupportedTypes: []dataset.DataType{dataset.TypeJSON},
		},
	}
}

// BuildStage constructs a concrete Stage from a declarative dataset.RuleSpec.
func BuildStage(spec dataset.RuleSpec) (Stage, error) {
	switch spec.Type {
	// Common & Null handling
	case "trim_space":
		return TrimSpaceStage{}, nil
	case "null_if_empty":
		return NullIfEmptyStage{}, nil
	case "drop_row_if_null":
		return DropRowIfNullStage{}, nil
	case "default_if_null":
		return DefaultValueStage{NameID: "default_if_null", DefaultValue: spec.Params["value"]}, nil
	case "default_zero_if_null":
		return DefaultValueStage{NameID: "default_zero_if_null", DefaultValue: "0"}, nil
	case "default_false_if_null":
		return DefaultValueStage{NameID: "default_false_if_null", DefaultValue: "false"}, nil

	// Text stages
	case "collapse_whitespace":
		return CollapseWhitespaceStage{}, nil
	case "to_lower":
		return CaseStage{Mode: "lower"}, nil
	case "to_upper":
		return CaseStage{Mode: "upper"}, nil
	case "to_title":
		return CaseStage{Mode: "title"}, nil
	case "strip_html":
		return StripHTMLStage{}, nil
	case "remove_non_printable":
		return RemoveNonPrintableStage{}, nil
	case "extract_digits":
		return ExtractDigitsStage{}, nil
	case "sanitize_csv_formula":
		return SanitizeCSVFormulaStage{}, nil
	case "regex_replace":
		re, err := regexp.Compile(spec.Params["pattern"])
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return RegexReplaceStage{
			Pattern:     re,
			Replacement: spec.Params["replacement"],
		}, nil

	// Numeric stages
	case "strip_currency":
		return StripCurrencyStage{}, nil
	case "abs_number":
		return AbsNumberStage{}, nil
	case "round_2_decimals":
		return RoundFloatStage{Decimals: 2}, nil
	case "round_integer":
		return RoundFloatStage{Decimals: 0}, nil
	case "non_negative":
		return EnforceNonNegativeStage{}, nil
	case "positive_only":
		return EnforcePositiveStage{}, nil

	// Domain-specific validation stages
	case "normalize_iso_date":
		return NormalizeISODateStage{}, nil
	case "validate_email":
		return ValidateEmailStage{}, nil
	case "mask_email_user":
		return MaskEmailUserStage{}, nil
	case "validate_uuid":
		return ValidateUUIDStage{}, nil
	case "compact_json":
		return CompactJSONStage{}, nil

	default:
		return nil, fmt.Errorf("unknown pipeline rule type %q", spec.Type)
	}
}
