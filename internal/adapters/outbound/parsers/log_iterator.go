package parsers

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/diego/go-refinery/internal/ports"
)

var (
	// Matches common structured/semi-structured log prefixes:
	// e.g. "2026-10-24T12:00:00Z INFO [auth] user logged in" or "2026-10-24 12:00:00 [ERROR] connection failed"
	standardLogRegex = regexp.MustCompile(`^(?P<timestamp>\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+\[?(?P<level>DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\]?\s+(?P<message>.*)$`)

	// Matches key=value or key="quoted value" pairs in logfmt lines.
	kvLogRegex = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_.-]*)=("[^"]*"|\S+)`)
)

type logMode int

const (
	logModeStandard logMode = iota
	logModeKeyValue
	logModeRaw
)

type logIterator struct {
	closer  io.Closer
	scanner *bufio.Scanner
	mode    logMode
	headers []string
	keyMap  map[string]int
	pending []string
	hasNext bool
}

// newLogIterator streams plain-text or logfmt log files line-by-line in O(1) memory.
func newLogIterator(rc io.ReadCloser) (ports.RowIterator, error) {
	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScannerLineBytes)

	it := &logIterator{
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
		it.detectModeAndHeaders(line)
		it.pending = it.parseLine(line)
		it.hasNext = true
		break
	}

	if err := scanner.Err(); err != nil {
		_ = rc.Close()
		return nil, fmt.Errorf("scanning log stream: %w", err)
	}

	if len(it.headers) == 0 {
		it.mode = logModeRaw
		it.headers = []string{"raw_line"}
	}

	return it, nil
}

func (it *logIterator) Headers() []string {
	return it.headers
}

func (it *logIterator) Next() ([]string, error) {
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
		return it.parseLine(line), nil
	}

	if err := it.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (it *logIterator) Close() error {
	return it.closer.Close()
}

func (it *logIterator) detectModeAndHeaders(firstLine string) {
	if standardLogRegex.MatchString(firstLine) {
		it.mode = logModeStandard
		it.headers = []string{"timestamp", "level", "message"}
		return
	}

	matches := kvLogRegex.FindAllStringSubmatch(firstLine, -1)
	if len(matches) >= 2 {
		it.mode = logModeKeyValue
		keys := make([]string, 0, len(matches))
		seen := make(map[string]struct{}, len(matches))
		for _, m := range matches {
			k := m[1]
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		it.headers = keys
		for idx, k := range keys {
			it.keyMap[k] = idx
		}
		return
	}

	it.mode = logModeRaw
	it.headers = []string{"raw_line"}
}

func (it *logIterator) parseLine(line string) []string {
	switch it.mode {
	case logModeStandard:
		m := standardLogRegex.FindStringSubmatch(line)
		if len(m) == 4 {
			return []string{m[1], m[2], m[3]}
		}
		return []string{"", "INFO", line}

	case logModeKeyValue:
		matches := kvLogRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			k := m[1]
			if _, exists := it.keyMap[k]; !exists {
				it.keyMap[k] = len(it.headers)
				it.headers = append(it.headers, k)
			}
		}
		row := make([]string, len(it.headers))
		for _, m := range matches {
			k := m[1]
			val := strings.Trim(m[2], `"`)
			row[it.keyMap[k]] = val
		}
		return row

	default:
		return []string{line}
	}
}
