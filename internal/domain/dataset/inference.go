// Package dataset provides heuristic schema and data type inference over raw samples.
package dataset

import (
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const inferenceMajorityThreshold = 0.70

var (
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	nonIdent  = regexp.MustCompile(`[^a-z0-9_]+`)
)

var supportedDateLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
}

// InferSchema analyzes a header slice and a sample of raw string rows to produce
// an initial Schema with inferred DataTypes, null counts, and normalized column names.
func InferSchema(headers []string, sampleRows [][]string) Schema {
	colCount := len(headers)
	for _, r := range sampleRows {
		if len(r) > colCount {
			colCount = len(r)
		}
	}

	cols := make([]ColumnSchema, colCount)
	for i := 0; i < colCount; i++ {
		rawHeader := ""
		if i < len(headers) {
			rawHeader = strings.TrimSpace(headers[i])
		}
		if rawHeader == "" {
			rawHeader = "column_" + strconv.Itoa(i+1)
		}

		distinct := make(map[string]struct{})
		var nonEmpty []string
		var nullCount int64

		for _, row := range sampleRows {
			var val string
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}
			if isNullLiteral(val) {
				nullCount++
				continue
			}
			distinct[val] = struct{}{}
			if len(nonEmpty) < 5 {
				nonEmpty = append(nonEmpty, val)
			}
		}

		inferred := inferColumnType(sampleRows, i)
		cols[i] = ColumnSchema{
			Index:          i,
			OriginalName:   rawHeader,
			NormalizedName: NormalizeIdentifier(rawHeader),
			InferredType:   inferred,
			TargetType:     inferred,
			Nullable:       nullCount > 0,
			NullCount:      nullCount,
			SampleDistinct: int64(len(distinct)),
			SampleValues:   nonEmpty,
		}
	}

	return Schema{Columns: cols}
}

// NormalizeIdentifier converts an arbitrary column header into a safe snake_case SQL/JSON key.
func NormalizeIdentifier(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = nonIdent.ReplaceAllString(s, "")
	s = strings.Trim(s, "_")
	if s == "" {
		return "col"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "c_" + s
	}
	return s
}

func inferColumnType(rows [][]string, colIdx int) DataType {
	var (
		seenNonEmpty int
		intMatches   int
		floatMatches int
		boolMatches  int
		dateMatches  int
		uuidMatches  int
		emailMatches int
	)

	for _, row := range rows {
		if colIdx >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[colIdx])
		if isNullLiteral(val) {
			continue
		}
		seenNonEmpty++

		if _, err := strconv.ParseInt(val, 10, 64); err == nil {
			intMatches++
		}
		if _, err := strconv.ParseFloat(val, 64); err == nil {
			floatMatches++
		}
		if isBoolLiteral(val) {
			boolMatches++
		}
		if uuidRegex.MatchString(val) {
			uuidMatches++
		}
		if addr, err := mail.ParseAddress(val); err == nil && addr.Address == val {
			emailMatches++
		}
		if isDateLiteral(val) {
			dateMatches++
		}
	}

	if seenNonEmpty == 0 {
		return TypeString
	}

	meetsThreshold := func(matches int) bool {
		return float64(matches)/float64(seenNonEmpty) >= inferenceMajorityThreshold
	}

	switch {
	case meetsThreshold(boolMatches):
		return TypeBoolean
	case meetsThreshold(intMatches):
		return TypeInteger
	case meetsThreshold(floatMatches):
		return TypeFloat
	case meetsThreshold(uuidMatches):
		return TypeUUID
	case meetsThreshold(emailMatches):
		return TypeEmail
	case meetsThreshold(dateMatches):
		return TypeDateTime
	default:
		return TypeString
	}
}

func isNullLiteral(val string) bool {
	if val == "" {
		return true
	}
	switch strings.ToLower(val) {
	case "null", "nil", "n/a", "na", "none", "-":
		return true
	default:
		return false
	}
}

func isBoolLiteral(val string) bool {
	switch strings.ToLower(val) {
	case "true", "false", "yes", "no", "1", "0", "t", "f", "y", "n":
		return true
	default:
		return false
	}
}

func isDateLiteral(val string) bool {
	for _, layout := range supportedDateLayouts {
		if _, err := time.Parse(layout, val); err == nil {
			return true
		}
	}
	return false
}
