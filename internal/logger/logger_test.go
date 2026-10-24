package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/diego/go-refinery/internal/logger"
)

func TestLogger_TextAndJSONFormatting(t *testing.T) {
	var textBuf bytes.Buffer
	textLog := logger.New(&textBuf, logger.LevelInfo, false).With(logger.F("component", "test-svc"))

	// Debug should be filtered out at LevelInfo
	textLog.Debug("ignored debug message", logger.F("k", "v"))
	if textBuf.Len() != 0 {
		t.Fatalf("expected debug message to be filtered out, got %q", textBuf.String())
	}

	ctx := logger.WithRequestID(context.Background(), "req-123")
	textLog.InfoContext(ctx, "dataset uploaded",
		logger.F("dataset_id", "ds-1"),
		logger.F("rows", 42),
	)

	line := textBuf.String()
	for _, expected := range []string{
		"level=INFO",
		"component=test-svc",
		"request_id=req-123",
		`msg="dataset uploaded"`,
		"dataset_id=ds-1",
		"rows=42",
	} {
		if !strings.Contains(line, expected) {
			t.Errorf("expected text log line to contain %q, got %q", expected, line)
		}
	}

	// JSON formatting test
	var jsonBuf bytes.Buffer
	jsonLog := logger.New(&jsonBuf, logger.LevelDebug, true).With(logger.F("component", "json-svc"))

	jsonLog.ErrorContext(ctx, "database failure", logger.Err(errors.New("disk full")))

	var parsed map[string]any
	if err := json.Unmarshal(jsonBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON log output: %v (raw: %s)", err, jsonBuf.String())
	}
	if parsed["level"] != "ERROR" {
		t.Errorf("expected level=ERROR, got %v", parsed["level"])
	}
	if parsed["component"] != "json-svc" {
		t.Errorf("expected component=json-svc, got %v", parsed["component"])
	}
	if parsed["request_id"] != "req-123" {
		t.Errorf("expected request_id=req-123, got %v", parsed["request_id"])
	}
	if parsed["error"] != "disk full" {
		t.Errorf("expected error='disk full', got %v", parsed["error"])
	}
}
