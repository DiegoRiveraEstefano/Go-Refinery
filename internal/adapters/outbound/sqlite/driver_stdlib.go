package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	RegisterStdlibDriver("sqlite")
}

// RegisterStdlibDriver registers a zero-dependency database/sql driver backed by
// the system's sqlite3 engine if driverName is not already registered.
func RegisterStdlibDriver(driverName string) {
	for _, existing := range sql.Drivers() {
		if existing == driverName {
			return
		}
	}
	sql.Register(driverName, &cliDriver{})
}

type cliDriver struct{}

func (d *cliDriver) Open(dsn string) (driver.Conn, error) {
	return &cliConn{dsn: dsn}, nil
}

type cliConn struct {
	mu      sync.Mutex
	dsn     string
	inTx    bool
	txBatch []string
}

func (c *cliConn) Prepare(query string) (driver.Stmt, error) {
	return &cliStmt{conn: c, query: query}, nil
}

func (c *cliConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inTx = false
	c.txBatch = nil
	return nil
}

func (c *cliConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *cliConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inTx = true
	c.txBatch = c.txBatch[:0]
	return &cliTx{conn: c}, nil
}

func (c *cliConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	boundSQL, err := interpolateSQL(query, namedToValues(args))
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inTx {
		c.txBatch = append(c.txBatch, ensureSemicolon(boundSQL))
		return driver.RowsAffected(1), nil
	}

	return c.runExec(ctx, boundSQL)
}

func (c *cliConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	boundSQL, err := interpolateSQL(query, namedToValues(args))
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var script strings.Builder
	script.WriteString(".timeout 5000\n")
	script.WriteString("PRAGMA foreign_keys = ON;\n")
	if c.inTx && len(c.txBatch) > 0 {
		for _, stmt := range c.txBatch {
			script.WriteString(stmt)
			script.WriteByte('\n')
		}
	}
	script.WriteString(ensureSemicolon(boundSQL))

	return c.runQuery(ctx, query, script.String())
}

func (c *cliConn) runExec(ctx context.Context, sqlText string) (driver.Result, error) {
	var script strings.Builder
	script.WriteString(".timeout 5000\n")
	script.WriteString("PRAGMA foreign_keys = ON;\n")
	script.WriteString(ensureSemicolon(sqlText))
	script.WriteString("\nSELECT changes() AS _changes, last_insert_rowid() AS _last_id;\n")

	out, err := execSQLite3(ctx, c.dsn, true, script.String())
	if err != nil {
		return nil, err
	}

	var meta []struct {
		Changes int64 `json:"_changes"`
		LastID  int64 `json:"_last_id"`
	}
	if len(out) > 0 {
		if idx := bytes.LastIndexByte(out, '['); idx != -1 {
			_ = json.Unmarshal(out[idx:], &meta)
		}
	}
	if len(meta) > 0 {
		return cliResult{changes: meta[0].Changes, lastID: meta[0].LastID}, nil
	}
	return cliResult{changes: 0, lastID: 0}, nil
}

func (c *cliConn) runQuery(ctx context.Context, rawSelectQuery, script string) (driver.Rows, error) {
	out, err := execSQLite3(ctx, c.dsn, true, script)
	if err != nil {
		return nil, err
	}

	fallbackCols := parseSelectColumns(rawSelectQuery)
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return &cliRows{cols: fallbackCols}, nil
	}

	if idx := bytes.LastIndex(trimmed, []byte("\n[")); idx != -1 {
		trimmed = bytes.TrimSpace(trimmed[idx+1:])
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil || tok != json.Delim('[') {
		return &cliRows{cols: fallbackCols}, nil
	}

	var (
		cols []string
		data [][]driver.Value
	)

	for dec.More() {
		nextCols, rowVals, err := decodeOrderedJSONObject(dec, cols)
		if err != nil {
			return nil, err
		}
		cols = nextCols
		data = append(data, rowVals)
	}

	if len(cols) == 0 {
		cols = fallbackCols
	}

	return &cliRows{cols: cols, data: data}, nil
}

func parseSelectColumns(q string) []string {
	upper := strings.ToUpper(q)
	selIdx := strings.Index(upper, "SELECT")
	fromIdx := strings.Index(upper, "FROM")
	if selIdx == -1 || fromIdx == -1 || fromIdx <= selIdx+6 {
		return []string{"_col0"}
	}
	projection := q[selIdx+6 : fromIdx]
	parts := strings.Split(projection, ",")
	cols := make([]string, len(parts))
	for i, p := range parts {
		cols[i] = strings.TrimSpace(p)
	}
	return cols
}

type cliTx struct {
	conn *cliConn
}

func (tx *cliTx) Commit() error {
	tx.conn.mu.Lock()
	defer tx.conn.mu.Unlock()

	if !tx.conn.inTx {
		return nil
	}
	batch := tx.conn.txBatch
	tx.conn.inTx = false
	tx.conn.txBatch = nil

	if len(batch) == 0 {
		return nil
	}

	var script strings.Builder
	script.WriteString(".timeout 5000\n")
	script.WriteString("PRAGMA foreign_keys = ON;\n")
	script.WriteString("BEGIN IMMEDIATE;\n")
	for _, s := range batch {
		script.WriteString(s)
		script.WriteByte('\n')
	}
	script.WriteString("COMMIT;\n")

	_, err := execSQLite3(context.Background(), tx.conn.dsn, false, script.String())
	return err
}

func (tx *cliTx) Rollback() error {
	tx.conn.mu.Lock()
	defer tx.conn.mu.Unlock()
	tx.conn.inTx = false
	tx.conn.txBatch = nil
	return nil
}

type cliStmt struct {
	conn  *cliConn
	query string
}

func (s *cliStmt) Close() error  { return nil }
func (s *cliStmt) NumInput() int { return -1 }

func (s *cliStmt) Exec(args []driver.Value) (driver.Result, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.conn.ExecContext(context.Background(), s.query, named)
}

func (s *cliStmt) Query(args []driver.Value) (driver.Rows, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.conn.QueryContext(context.Background(), s.query, named)
}

type cliResult struct {
	changes int64
	lastID  int64
}

func (r cliResult) LastInsertId() (int64, error) { return r.lastID, nil }
func (r cliResult) RowsAffected() (int64, error) { return r.changes, nil }

type cliRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func (r *cliRows) Columns() []string { return r.cols }
func (r *cliRows) Close() error      { return nil }

func (r *cliRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func execSQLite3(ctx context.Context, dsn string, jsonOutput bool, sqlScript string) ([]byte, error) {
	args := []string{"-bail", "-batch", "-noheader"}
	if jsonOutput {
		args = append(args, "-json")
	}
	args = append(args, dsn)

	cmd := exec.CommandContext(ctx, "sqlite3", args...)
	cmd.Stdin = strings.NewReader(sqlScript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("sqlite3 error: %s", msg)
	}
	return stdout.Bytes(), nil
}

func decodeOrderedJSONObject(dec *json.Decoder, existingCols []string) ([]string, []driver.Value, error) {
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, nil, fmt.Errorf("expected json object start '{'")
	}

	cols := existingCols
	var vals []driver.Value
	firstRow := len(cols) == 0
	colMap := make(map[string]driver.Value, len(cols))

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key := keyTok.(string)

		var rawVal any
		if err := dec.Decode(&rawVal); err != nil {
			return nil, nil, err
		}
		dv := toDriverValue(rawVal)
		if firstRow {
			cols = append(cols, key)
			vals = append(vals, dv)
		} else {
			colMap[key] = dv
		}
	}

	// Consume closing '}'
	_, _ = dec.Token()

	if !firstRow {
		vals = make([]driver.Value, len(cols))
		for i, c := range cols {
			vals[i] = colMap[c]
		}
	}
	return cols, vals, nil
}

func toDriverValue(v any) driver.Value {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	case string:
		return val
	case bool:
		if val {
			return int64(1)
		}
		return int64(0)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func namedToValues(named []driver.NamedValue) []driver.Value {
	out := make([]driver.Value, len(named))
	for i, nv := range named {
		out[i] = nv.Value
	}
	return out
}

func interpolateSQL(query string, args []driver.Value) (string, error) {
	if len(args) == 0 {
		return query, nil
	}
	var (
		sb      strings.Builder
		argIdx  int
		inQuote bool
	)

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			inQuote = !inQuote
			sb.WriteByte(ch)
			continue
		}
		if ch == '?' && !inQuote {
			if argIdx >= len(args) {
				return "", fmt.Errorf("not enough sql arguments for query")
			}
			formatted, err := formatSQLLiteral(args[argIdx])
			if err != nil {
				return "", err
			}
			sb.WriteString(formatted)
			argIdx++
			continue
		}
		sb.WriteByte(ch)
	}
	return sb.String(), nil
}

func formatSQLLiteral(v driver.Value) (string, error) {
	if v == nil {
		return "NULL", nil
	}
	switch val := v.(type) {
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64), nil
	case bool:
		if val {
			return "1", nil
		}
		return "0", nil
	case []byte:
		return "X'" + hex.EncodeToString(val) + "'", nil
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'", nil
	case time.Time:
		return "'" + val.UTC().Format(time.RFC3339Nano) + "'", nil
	default:
		return "", fmt.Errorf("unsupported sql parameter type %T", v)
	}
}

func ensureSemicolon(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || strings.HasSuffix(trimmed, ";") {
		return trimmed
	}
	return trimmed + ";"
}
