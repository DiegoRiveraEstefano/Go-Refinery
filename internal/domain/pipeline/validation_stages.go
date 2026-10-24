package pipeline

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var strictUUIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// NullIfEmptyStage normalizes sentinel null literals ("null", "N/A", "-", etc.) into an empty string.
type NullIfEmptyStage struct{}

func (NullIfEmptyStage) Name() string { return "null_if_empty" }
func (NullIfEmptyStage) Apply(v string) (string, error) {
	if isNullSentinel(v) {
		return "", nil
	}
	return v, nil
}

// DropRowIfNullStage signals the worker pool to discard the entire row when the cell is empty or null.
type DropRowIfNullStage struct{}

func (DropRowIfNullStage) Name() string { return "drop_row_if_null" }
func (DropRowIfNullStage) Apply(v string) (string, error) {
	if isNullSentinel(v) {
		return "", ErrDropRow
	}
	return v, nil
}

// DefaultValueStage replaces empty or sentinel null strings with a fallback value.
type DefaultValueStage struct {
	NameID       string
	DefaultValue string
}

func (d DefaultValueStage) Name() string {
	if d.NameID != "" {
		return d.NameID
	}
	return "default_if_null"
}

func (d DefaultValueStage) Apply(v string) (string, error) {
	if isNullSentinel(v) {
		return d.DefaultValue, nil
	}
	return v, nil
}

// ValidateEmailStage verifies RFC 5322 email syntax and normalizes the domain to lowercase.
type ValidateEmailStage struct{}

func (ValidateEmailStage) Name() string { return "validate_email" }
func (ValidateEmailStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address == "" {
		return "", fmt.Errorf("%w: invalid email address %q", ErrValidationCheck, v)
	}
	return strings.ToLower(addr.Address), nil
}

// MaskEmailUserStage masks the local part of an email address for privacy (e.g., "a***@example.com").
type MaskEmailUserStage struct{}

func (MaskEmailUserStage) Name() string { return "mask_email_user" }
func (MaskEmailUserStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	atIdx := strings.IndexByte(trimmed, '@')
	if atIdx <= 0 {
		return trimmed, nil
	}
	firstChar := trimmed[:1]
	domain := trimmed[atIdx:]
	return firstChar + "***" + strings.ToLower(domain), nil
}

// ValidateUUIDStage validates UUID format and normalizes hexadecimal characters to lowercase.
type ValidateUUIDStage struct{}

func (ValidateUUIDStage) Name() string { return "validate_uuid" }
func (ValidateUUIDStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	if !strictUUIDRegex.MatchString(trimmed) {
		return "", fmt.Errorf("%w: invalid UUID %q", ErrValidationCheck, v)
	}
	return strings.ToLower(trimmed), nil
}

// NormalizeISODateStage parses common date formats and normalizes them to YYYY-MM-DD.
type NormalizeISODateStage struct{}

func (NormalizeISODateStage) Name() string { return "normalize_iso_date" }
func (NormalizeISODateStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	for _, layout := range supportedDateLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t.UTC().Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("%w: unrecognized date format %q", ErrValidationCheck, v)
}

// CompactJSONStage validates JSON syntax and compacts whitespace.
type CompactJSONStage struct{}

func (CompactJSONStage) Name() string { return "compact_json" }
func (CompactJSONStage) Apply(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", nil
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("%w: invalid JSON payload: %v", ErrValidationCheck, err)
	}
	var buf strings.Builder
	b, err := json.Marshal(raw)
	if err != nil {
		return trimmed, nil
	}
	buf.Write(b)
	return buf.String(), nil
}
