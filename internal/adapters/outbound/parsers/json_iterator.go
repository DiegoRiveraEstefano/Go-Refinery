package parsers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/diego/go-refinery/internal/ports"
)

const maxScannerLineBytes = 4 * 1024 * 1024 // 4 MB per JSONL line

type jsonArrayIterator struct {
	closer  io.Closer
	dec     *json.Decoder
	headers []string
	keyMap  map[string]int
	pending []string
	hasNext bool
}

// newJSONIterator streams a top-level JSON array of objects (`[ {...}, {...} ]`)
// or falls back to newline-delimited JSON objects if the stream does not start with `[`.
func newJSONIterator(rc io.ReadCloser) (ports.RowIterator, error) {
	br := bufio.NewReaderSize(rc, 64*1024)
	if peek, err := br.Peek(3); err == nil && bytes.Equal(peek, utf8BOM) {
		_, _ = br.Discard(3)
	}

	dec := json.NewDecoder(br)
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		_ = rc.Close()
		return nil, fmt.Errorf("reading initial json token: %w", err)
	}

	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		_ = rc.Close()
		return nil, errors.New("expected top-level JSON array '['")
	}

	it := &jsonArrayIterator{
		closer: rc,
		dec:    dec,
		keyMap: make(map[string]int),
	}

	if dec.More() {
		var firstObj map[string]any
		if err := dec.Decode(&firstObj); err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("decoding first json array element: %w", err)
		}
		it.initHeaders(firstObj)
		it.pending = it.mapToRow(firstObj)
		it.hasNext = true
	}

	return it, nil
}

func (it *jsonArrayIterator) Headers() []string {
	return it.headers
}

func (it *jsonArrayIterator) Next() ([]string, error) {
	if it.hasNext {
		it.hasNext = false
		row := it.pending
		it.pending = nil
		return row, nil
	}

	if !it.dec.More() {
		return nil, io.EOF
	}

	var obj map[string]any
	if err := it.dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("decoding json object: %w", err)
	}

	if len(it.headers) == 0 {
		it.initHeaders(obj)
	}
	return it.mapToRow(obj), nil
}

func (it *jsonArrayIterator) Close() error {
	return it.closer.Close()
}

func (it *jsonArrayIterator) initHeaders(obj map[string]any) {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	it.headers = keys
	for idx, k := range keys {
		it.keyMap[k] = idx
	}
}

func (it *jsonArrayIterator) mapToRow(obj map[string]any) []string {
	for k := range obj {
		if _, exists := it.keyMap[k]; !exists {
			it.keyMap[k] = len(it.headers)
			it.headers = append(it.headers, k)
		}
	}
	row := make([]string, len(it.headers))
	for k, v := range obj {
		idx := it.keyMap[k]
		row[idx] = stringifyJSONValue(v)
	}
	return row
}

type jsonlIterator struct {
	closer  io.Closer
	scanner *bufio.Scanner
	headers []string
	keyMap  map[string]int
	pending []string
	hasNext bool
}

// newJSONLIterator streams newline-delimited JSON objects (JSONL / NDJSON).
func newJSONLIterator(rc io.ReadCloser) (ports.RowIterator, error) {
	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScannerLineBytes)

	it := &jsonlIterator{
		closer:  rc,
		scanner: scanner,
		keyMap:  make(map[string]int),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, string(utf8BOM))
		if line == "" {
			continue
		}
		obj, err := decodeJSONMap(line)
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("decoding first jsonl record: %w", err)
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		it.headers = keys
		for idx, k := range keys {
			it.keyMap[k] = idx
		}
		it.pending = it.mapToRow(obj)
		it.hasNext = true
		break
	}

	if err := scanner.Err(); err != nil {
		_ = rc.Close()
		return nil, fmt.Errorf("scanning jsonl stream: %w", err)
	}

	return it, nil
}

func (it *jsonlIterator) Headers() []string {
	return it.headers
}

func (it *jsonlIterator) Next() ([]string, error) {
	if it.hasNext {
		it.hasNext = false
		row := it.pending
		it.pending = nil
		return row, nil
	}

	for it.scanner.Scan() {
		line := strings.TrimSpace(it.scanner.Text())
		if line == "" {
			continue
		}
		obj, err := decodeJSONMap(line)
		if err != nil {
			return nil, err
		}
		return it.mapToRow(obj), nil
	}

	if err := it.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (it *jsonlIterator) Close() error {
	return it.closer.Close()
}

func (it *jsonlIterator) mapToRow(obj map[string]any) []string {
	for k := range obj {
		if _, exists := it.keyMap[k]; !exists {
			it.keyMap[k] = len(it.headers)
			it.headers = append(it.headers, k)
		}
	}
	row := make([]string, len(it.headers))
	for k, v := range obj {
		idx := it.keyMap[k]
		row[idx] = stringifyJSONValue(v)
	}
	return row
}

func decodeJSONMap(line string) (map[string]any, error) {
	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("invalid json line: %w", err)
	}
	return obj, nil
}

func stringifyJSONValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	case bool:
		return strconv.FormatBool(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}
