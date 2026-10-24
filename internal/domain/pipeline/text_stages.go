package pipeline

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	multiSpaceRegex = regexp.MustCompile(`\s+`)
	digitsOnlyRegex = regexp.MustCompile(`[^0-9]+`)
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
)

// TrimSpaceStage strips leading and trailing whitespace.
type TrimSpaceStage struct{}

func (TrimSpaceStage) Name() string                   { return "trim_space" }
func (TrimSpaceStage) Apply(v string) (string, error) { return strings.TrimSpace(v), nil }

// CollapseWhitespaceStage collapses consecutive whitespace characters into a single space.
type CollapseWhitespaceStage struct{}

func (CollapseWhitespaceStage) Name() string { return "collapse_whitespace" }
func (CollapseWhitespaceStage) Apply(v string) (string, error) {
	return multiSpaceRegex.ReplaceAllString(strings.TrimSpace(v), " "), nil
}

// CaseStage normalizes text casing ("lower", "upper", or "title").
type CaseStage struct {
	Mode string
}

func (c CaseStage) Name() string {
	switch c.Mode {
	case "upper":
		return "to_upper"
	case "title":
		return "to_title"
	default:
		return "to_lower"
	}
}

func (c CaseStage) Apply(v string) (string, error) {
	switch c.Mode {
	case "upper":
		return strings.ToUpper(v), nil
	case "title":
		return toTitleCase(v), nil
	default:
		return strings.ToLower(v), nil
	}
}

func toTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// RemoveNonPrintableStage strips control and non-printable Unicode runes.
type RemoveNonPrintableStage struct{}

func (RemoveNonPrintableStage) Name() string { return "remove_non_printable" }
func (RemoveNonPrintableStage) Apply(v string) (string, error) {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, v), nil
}

// StripHTMLStage removes HTML/XML tags from a text cell.
type StripHTMLStage struct{}

func (StripHTMLStage) Name() string { return "strip_html" }
func (StripHTMLStage) Apply(v string) (string, error) {
	return strings.TrimSpace(htmlTagRegex.ReplaceAllString(v, "")), nil
}

// ExtractDigitsStage removes all non-digit characters (e.g., phone numbers or IDs).
type ExtractDigitsStage struct{}

func (ExtractDigitsStage) Name() string { return "extract_digits" }
func (ExtractDigitsStage) Apply(v string) (string, error) {
	return digitsOnlyRegex.ReplaceAllString(v, ""), nil
}

// SanitizeCSVFormulaStage neutralizes spreadsheet formula injection prefixes (=, +, -, @).
type SanitizeCSVFormulaStage struct{}

func (SanitizeCSVFormulaStage) Name() string { return "sanitize_csv_formula" }
func (SanitizeCSVFormulaStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return v, nil
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + trimmed, nil
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
