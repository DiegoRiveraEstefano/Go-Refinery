package parsers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/ports"
)

func drainIterator(t *testing.T, it ports.RowIterator) ([]string, [][]string) {
	t.Helper()
	defer it.Close()

	var rows [][]string
	for {
		row, err := it.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected iterator error: %v", err)
		}
		rows = append(rows, row)
	}
	return it.Headers(), rows
}

func TestCSVIterator_BOMAndSemicolonDelimiter(t *testing.T) {
	raw := "\xEF\xBB\xBFid;name;score\n10;Alice;99.5\n20;Bob;88.0\n"
	f := parsers.NewFactory()

	it, err := f.NewIterator(context.Background(), dataset.FormatCSV, io.NopCloser(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("NewIterator CSV failed: %v", err)
	}

	headers, rows := drainIterator(t, it)
	if len(headers) != 3 || headers[0] != "id" || headers[1] != "name" {
		t.Fatalf("unexpected CSV headers: %v", headers)
	}
	if len(rows) != 2 || rows[0][1] != "Alice" || rows[1][2] != "88.0" {
		t.Fatalf("unexpected CSV rows: %v", rows)
	}
}

func TestJSONIterator_StreamingArray(t *testing.T) {
	raw := `[
		{"id": 1, "email": "alice@example.com", "active": true},
		{"id": 2, "email": "bob@example.com", "active": false, "extra": "new_col"}
	]`
	f := parsers.NewFactory()

	it, err := f.NewIterator(context.Background(), dataset.FormatJSON, io.NopCloser(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("NewIterator JSON failed: %v", err)
	}

	headers, rows := drainIterator(t, it)
	if len(headers) != 4 {
		t.Fatalf("expected 4 headers after late key discovery, got %v", headers)
	}
	if len(rows) != 2 || rows[0][0] != "true" || rows[1][3] != "new_col" {
		t.Fatalf("unexpected JSON rows: %v (headers=%v)", rows, headers)
	}
}

func TestJSONLIterator_StreamingLines(t *testing.T) {
	raw := strings.Join([]string{
		`{"user":"u1","latency_ms":12.5}`,
		``,
		`{"user":"u2","latency_ms":48.0}`,
	}, "\n")
	f := parsers.NewFactory()

	it, err := f.NewIterator(context.Background(), dataset.FormatJSONL, io.NopCloser(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("NewIterator JSONL failed: %v", err)
	}

	headers, rows := drainIterator(t, it)
	if len(headers) != 2 || headers[0] != "latency_ms" || headers[1] != "user" {
		t.Fatalf("unexpected JSONL headers: %v", headers)
	}
	if len(rows) != 2 || rows[1][0] != "48.0" || rows[1][1] != "u2" {
		t.Fatalf("unexpected JSONL rows: %v", rows)
	}
}

func TestLogIterator_Modes(t *testing.T) {
	f := parsers.NewFactory()

	t.Run("StandardTimestampLevelMessage", func(t *testing.T) {
		raw := "2026-10-24T10:00:00Z INFO Server started on :8080\n2026-10-24T10:01:00Z [ERROR] Database timeout\n"
		it, err := f.NewIterator(context.Background(), dataset.FormatLog, io.NopCloser(strings.NewReader(raw)))
		if err != nil {
			t.Fatalf("NewIterator Log failed: %v", err)
		}
		headers, rows := drainIterator(t, it)
		if len(headers) != 3 || headers[0] != "timestamp" || headers[1] != "level" || headers[2] != "message" {
			t.Fatalf("unexpected log headers: %v", headers)
		}
		if len(rows) != 2 || rows[1][1] != "ERROR" || rows[1][2] != "Database timeout" {
			t.Fatalf("unexpected log rows: %v", rows)
		}
	})

	t.Run("LogfmtKeyValue", func(t *testing.T) {
		raw := `level=info service=ingest msg="file uploaded" bytes=2048` + "\n" +
			`level=error service=worker msg="cast failed" bytes=0` + "\n"
		it, err := f.NewIterator(context.Background(), dataset.FormatLog, io.NopCloser(strings.NewReader(raw)))
		if err != nil {
			t.Fatalf("NewIterator Logfmt failed: %v", err)
		}
		headers, rows := drainIterator(t, it)
		if len(headers) != 4 || headers[0] != "bytes" || headers[2] != "msg" {
			t.Fatalf("unexpected logfmt headers: %v", headers)
		}
		if len(rows) != 2 || rows[0][2] != "file uploaded" || rows[1][1] != "error" {
			t.Fatalf("unexpected logfmt rows: %v", rows)
		}
	})
}

func TestXLSXIterator_StreamingOpenXML(t *testing.T) {
	xlsxBytes := buildMinimalXLSX(t)
	f := parsers.NewFactory()

	it, err := f.NewIterator(context.Background(), dataset.FormatXLSX, io.NopCloser(bytes.NewReader(xlsxBytes)))
	if err != nil {
		t.Fatalf("NewIterator XLSX failed: %v", err)
	}

	headers, rows := drainIterator(t, it)
	if len(headers) != 3 || headers[0] != "ID" || headers[1] != "Email" || headers[2] != "Active" {
		t.Fatalf("unexpected XLSX headers: %v", headers)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 data rows, got %d", len(rows))
	}
	if rows[0][0] != "101" || rows[0][1] != "alice@example.com" || rows[0][2] != "true" {
		t.Fatalf("unexpected first XLSX row: %v", rows[0])
	}
	if rows[1][0] != "102" || rows[1][1] != "bob@example.com" || rows[1][2] != "false" {
		t.Fatalf("unexpected second XLSX row: %v", rows[1])
	}
}

func buildMinimalXLSX(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	sharedStringsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="5" uniqueCount="5">
  <si><t>ID</t></si>
  <si><t>Email</t></si>
  <si><t>Active</t></si>
  <si><t>alice@example.com</t></si>
  <si><t>bob@example.com</t></si>
</sst>`

	sheet1XML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c>
      <c r="B1" t="s"><v>1</v></c>
      <c r="C1" t="s"><v>2</v></c>
    </row>
    <row r="2">
      <c r="A2"><v>101</v></c>
      <c r="B2" t="s"><v>3</v></c>
      <c r="C2" t="b"><v>1</v></c>
    </row>
    <row r="3">
      <c r="A3"><v>102</v></c>
      <c r="B3" t="s"><v>4</v></c>
      <c r="C3" t="b"><v>0</v></c>
    </row>
  </sheetData>
</worksheet>`

	writeZipEntry := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("writing zip entry %s: %v", name, err)
		}
	}

	writeZipEntry("xl/sharedStrings.xml", sharedStringsXML)
	writeZipEntry("xl/worksheets/sheet1.xml", sheet1XML)

	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	return buf.Bytes()
}
