// Package parsers implements streaming file readers for CSV, XLSX, JSON/JSONL, and Logs.
package parsers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/ports"
)

// Compile-time check that Factory satisfies ports.StreamParserFactory.
var _ ports.StreamParserFactory = (*Factory)(nil)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Factory creates format-specific streaming RowIterators.
type Factory struct{}

// NewFactory creates a new stream parser Factory.
func NewFactory() *Factory {
	return &Factory{}
}

// DetectFormat infers the dataset.FileFormat from a filename extension.
func DetectFormat(filename string) (dataset.FileFormat, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv", ".tsv":
		return dataset.FormatCSV, nil
	case ".xlsx":
		return dataset.FormatXLSX, nil
	case ".json":
		return dataset.FormatJSON, nil
	case ".jsonl", ".ndjson":
		return dataset.FormatJSONL, nil
	case ".log", ".txt":
		return dataset.FormatLog, nil
	default:
		return "", fmt.Errorf("%w: extension %q", dataset.ErrUnsupportedFormat, ext)
	}
}

// NewIterator returns a streaming RowIterator for the given file format.
func (f *Factory) NewIterator(_ context.Context, format dataset.FileFormat, r io.ReadCloser) (ports.RowIterator, error) {
	switch format {
	case dataset.FormatCSV:
		return newCSVIterator(r)
	case dataset.FormatJSON:
		return newJSONIterator(r)
	case dataset.FormatJSONL:
		return newJSONLIterator(r)
	case dataset.FormatLog:
		return newLogIterator(r)
	case dataset.FormatXLSX:
		return newXLSXIterator(r)
	default:
		_ = r.Close()
		return nil, fmt.Errorf("%w: %s", dataset.ErrUnsupportedFormat, format)
	}
}

type csvIterator struct {
	closer  io.Closer
	reader  *csv.Reader
	headers []string
}

func newCSVIterator(rc io.ReadCloser) (ports.RowIterator, error) {
	br := bufio.NewReaderSize(rc, 64*1024)

	// Strip optional UTF-8 BOM at the beginning of the stream.
	if peek, err := br.Peek(3); err == nil && bytes.Equal(peek, utf8BOM) {
		_, _ = br.Discard(3)
	}

	// Peek the first line to auto-detect the delimiter (',', ';', '\t', '|').
	firstLine, _ := br.Peek(4096)
	delim := detectCSVDelimiter(firstLine)

	cr := csv.NewReader(br)
	cr.Comma = delim
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	headers, err := cr.Read()
	if err != nil {
		_ = rc.Close()
		return nil, fmt.Errorf("reading csv header row: %w", err)
	}

	return &csvIterator{
		closer:  rc,
		reader:  cr,
		headers: headers,
	}, nil
}

func (it *csvIterator) Headers() []string {
	return it.headers
}

func (it *csvIterator) Next() ([]string, error) {
	return it.reader.Read()
}

func (it *csvIterator) Close() error {
	return it.closer.Close()
}

func detectCSVDelimiter(sample []byte) rune {
	if idx := bytes.IndexByte(sample, '\n'); idx != -1 {
		sample = sample[:idx]
	}
	candidates := []rune{',', ';', '\t', '|'}
	best := ','
	bestCount := 0
	line := string(sample)
	for _, c := range candidates {
		cnt := strings.Count(line, string(c))
		if cnt > bestCount {
			bestCount = cnt
			best = c
		}
	}
	return best
}
