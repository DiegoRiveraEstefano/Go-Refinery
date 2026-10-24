package dataset

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SearchOperator defines a MongoDB-inspired comparison or text matching operator.
type SearchOperator string

const (
	OpContains SearchOperator = "$contains"
	OpRegex    SearchOperator = "$regex"
	OpEq       SearchOperator = "$eq"
	OpNe       SearchOperator = "$ne"
	OpGt       SearchOperator = "$gt"
	OpGte      SearchOperator = "$gte"
	OpLt       SearchOperator = "$lt"
	OpLte      SearchOperator = "$lte"
	OpIn       SearchOperator = "$in"
	OpExists   SearchOperator = "$exists"
)

// SearchQuery encapsulates user-supplied search parameters from the dataset detail view.
// It supports both structured field/operator/value controls and a MongoDB-style JSON
// or mini-DSL query string (Query).
type SearchQuery struct {
	Query    string         // Free text, /regex/, mini-DSL ("age > 20"), or Mongo JSON (`{"age": {"$gt": 20}}`)
	Field    string         // Optional column name (empty or "*" means any column)
	Operator SearchOperator // Optional operator for the structured control ($contains, $regex, $eq, $gt, etc.)
	Value    string         // Optional target value for the structured control
}

// IsEmpty reports whether no search filter is active.
func (q SearchQuery) IsEmpty() bool {
	return strings.TrimSpace(q.Query) == "" &&
		strings.TrimSpace(q.Value) == "" &&
		q.Operator != OpExists
}

// CompiledQuery is a pre-parsed, ready-to-evaluate predicate tree over dataset.Record rows.
type CompiledQuery struct {
	matchers []recordMatcher
}

type recordMatcher interface {
	Match(rec Record) bool
}

// CompileSearchQuery parses and validates a SearchQuery (including regex compilation
// and MongoDB JSON or mini-DSL parsing) into a fast, reusable CompiledQuery.
func CompileSearchQuery(q SearchQuery) (*CompiledQuery, error) {
	var matchers []recordMatcher

	// 1. Compile structured Field + Operator + Value control if provided.
	op := q.Operator
	if op == "" {
		op = OpContains
	}
	val := strings.TrimSpace(q.Value)
	field := strings.TrimSpace(q.Field)
	if field == "*" {
		field = ""
	}

	if val != "" || op == OpExists {
		if op == OpExists && val == "" {
			val = "true"
		}
		m, err := buildConditionMatcher(field, op, val)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}

	// 2. Compile Query string (MongoDB JSON, /regex/, mini-DSL, or free text).
	raw := strings.TrimSpace(q.Query)
	if raw != "" {
		m, err := parseRawQueryString(raw, field)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}

	return &CompiledQuery{matchers: matchers}, nil
}

// Match evaluates whether a Record satisfies all compiled search conditions.
func (cq *CompiledQuery) Match(rec Record) bool {
	if cq == nil || len(cq.matchers) == 0 {
		return true
	}
	for _, m := range cq.matchers {
		if !m.Match(rec) {
			return false
		}
	}
	return true
}

func parseRawQueryString(raw string, defaultField string) (recordMatcher, error) {
	// 1. MongoDB-style JSON object: {"age": {"$gt": 25}, "name": {"$regex": "^al"}}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, fmt.Errorf("invalid Mongo JSON query: %w", err)
		}
		return compileMongoDoc(doc)
	}

	// 2. Slash-delimited regular expression: /pattern/ or /pattern/i
	if strings.HasPrefix(raw, "/") && len(raw) > 2 {
		if lastSlash := strings.LastIndex(raw[1:], "/"); lastSlash >= 0 {
			pattern := raw[1 : 1+lastSlash]
			flags := raw[2+lastSlash:]
			if strings.Contains(flags, "i") && !strings.HasPrefix(pattern, "(?i)") {
				pattern = "(?i)" + pattern
			}
			return buildConditionMatcher(defaultField, OpRegex, pattern)
		}
	}

	// 3. Mini-DSL expression (e.g. "age >= 21 AND status = active" or "name =~ ^alice")
	if dslMatcher, ok, err := tryParseMiniDSL(raw); err != nil {
		return nil, err
	} else if ok {
		return dslMatcher, nil
	}

	// 4. Default: case-insensitive free-text substring search across defaultField (or all fields).
	return buildConditionMatcher(defaultField, OpContains, raw)
}

func compileMongoDoc(doc map[string]any) (recordMatcher, error) {
	var all []recordMatcher
	for key, rawVal := range doc {
		switch key {
		case "$and":
			subMatchers, err := compileMongoLogicalArray(rawVal)
			if err != nil {
				return nil, fmt.Errorf("$and: %w", err)
			}
			all = append(all, andMatcher{items: subMatchers})
		case "$or":
			subMatchers, err := compileMongoLogicalArray(rawVal)
			if err != nil {
				return nil, fmt.Errorf("$or: %w", err)
			}
			all = append(all, orMatcher{items: subMatchers})
		default:
			fieldMatcher, err := compileMongoFieldExpr(key, rawVal)
			if err != nil {
				return nil, err
			}
			all = append(all, fieldMatcher)
		}
	}
	return andMatcher{items: all}, nil
}

func compileMongoLogicalArray(rawVal any) ([]recordMatcher, error) {
	arr, ok := rawVal.([]any)
	if !ok || len(arr) == 0 {
		return nil, fmt.Errorf("expected non-empty array of filter objects")
	}
	out := make([]recordMatcher, 0, len(arr))
	for _, elem := range arr {
		subDoc, ok := elem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("logical array element must be an object")
		}
		m, err := compileMongoDoc(subDoc)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func compileMongoFieldExpr(field string, rawVal any) (recordMatcher, error) {
	if opMap, ok := rawVal.(map[string]any); ok {
		var fieldOps []recordMatcher
		for opKey, opVal := range opMap {
			op := normalizeOperator(opKey)
			if op == "" {
				return nil, fmt.Errorf("unsupported operator %q on field %q", opKey, field)
			}
			strVal := stringifyQueryValue(opVal)
			m, err := buildConditionMatcher(field, op, strVal)
			if err != nil {
				return nil, err
			}
			fieldOps = append(fieldOps, m)
		}
		return andMatcher{items: fieldOps}, nil
	}

	// Direct equality: {"country": "ES"} or {"age": 30}
	return buildConditionMatcher(field, OpEq, stringifyQueryValue(rawVal))
}

func stringifyQueryValue(v any) string {
	if v == nil {
		return ""
	}
	if arr, ok := v.([]any); ok {
		parts := make([]string, len(arr))
		for i, item := range arr {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprintf("%v", v)
}

func normalizeOperator(raw string) SearchOperator {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "$contains", "contains", "~":
		return OpContains
	case "$regex", "regex", "=~":
		return OpRegex
	case "$eq", "eq", "=", "==":
		return OpEq
	case "$ne", "ne", "!=":
		return OpNe
	case "$gt", "gt", ">":
		return OpGt
	case "$gte", "gte", ">=":
		return OpGte
	case "$lt", "lt", "<":
		return OpLt
	case "$lte", "lte", "<=":
		return OpLte
	case "$in", "in":
		return OpIn
	case "$exists", "exists":
		return OpExists
	default:
		return ""
	}
}

var dslClausePattern = regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*(=~|!=|>=|<=|==|=|>|<|~|\$[a-zA-Z]+)\s*(.+?)\s*$`)

func tryParseMiniDSL(raw string) (recordMatcher, bool, error) {
	// Split by " AND " (case-insensitive) or ";"
	clauses := splitByLogicalAnd(raw)
	var matchers []recordMatcher

	for _, c := range clauses {
		sub := dslClausePattern.FindStringSubmatch(c)
		if len(sub) != 4 {
			return nil, false, nil
		}
		field := sub[1]
		op := normalizeOperator(sub[2])
		if op == "" {
			return nil, false, nil
		}
		val := strings.Trim(strings.TrimSpace(sub[3]), `"'`)
		m, err := buildConditionMatcher(field, op, val)
		if err != nil {
			return nil, true, err
		}
		matchers = append(matchers, m)
	}

	if len(matchers) == 0 {
		return nil, false, nil
	}
	return andMatcher{items: matchers}, true, nil
}

func splitByLogicalAnd(s string) []string {
	var parts []string
	re := regexp.MustCompile(`(?i)\s+AND\s+|\s*;\s*`)
	for _, p := range re.Split(s, -1) {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

type andMatcher struct {
	items []recordMatcher
}

func (a andMatcher) Match(rec Record) bool {
	for _, m := range a.items {
		if !m.Match(rec) {
			return false
		}
	}
	return true
}

type orMatcher struct {
	items []recordMatcher
}

func (o orMatcher) Match(rec Record) bool {
	for _, m := range o.items {
		if m.Match(rec) {
			return true
		}
	}
	return false
}

type conditionMatcher struct {
	field     string
	op        SearchOperator
	targetStr string
	targetNum float64
	hasNum    bool
	re        *regexp.Regexp
	inSet     map[string]struct{}
	existsVal bool
}

func buildConditionMatcher(field string, op SearchOperator, rawVal string) (recordMatcher, error) {
	cm := conditionMatcher{
		field:     field,
		op:        op,
		targetStr: rawVal,
	}

	if f, err := strconv.ParseFloat(strings.TrimSpace(rawVal), 64); err == nil {
		cm.targetNum = f
		cm.hasNum = true
	}

	switch op {
	case OpRegex:
		re, err := regexp.Compile(rawVal)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression %q: %w", rawVal, err)
		}
		cm.re = re
	case OpIn:
		cm.inSet = make(map[string]struct{})
		for _, part := range strings.Split(rawVal, ",") {
			cleaned := strings.ToLower(strings.Trim(strings.TrimSpace(part), `"'[]`))
			if cleaned != "" {
				cm.inSet[cleaned] = struct{}{}
			}
		}
	case OpExists:
		lower := strings.ToLower(strings.TrimSpace(rawVal))
		cm.existsVal = lower != "false" && lower != "0" && lower != "no"
	}

	return cm, nil
}

func (c conditionMatcher) Match(rec Record) bool {
	if c.field != "" {
		val, exists := rec.Values[c.field]
		return c.matchSingleValue(val, exists && val != nil && fmt.Sprintf("%v", val) != "")
	}

	// Wildcard / all-columns search: returns true if ANY column matches (or for $ne, if ALL columns satisfy !=).
	if c.op == OpNe {
		for _, val := range rec.Values {
			if !c.matchSingleValue(val, val != nil && fmt.Sprintf("%v", val) != "") {
				return false
			}
		}
		return true
	}

	for _, val := range rec.Values {
		if c.matchSingleValue(val, val != nil && fmt.Sprintf("%v", val) != "") {
			return true
		}
	}
	return false
}

func (c conditionMatcher) matchSingleValue(val any, nonNull bool) bool {
	if c.op == OpExists {
		return nonNull == c.existsVal
	}
	if !nonNull {
		return c.op == OpNe && c.targetStr != ""
	}

	strVal := formatRecordValue(val)

	switch c.op {
	case OpContains:
		return strings.Contains(strings.ToLower(strVal), strings.ToLower(c.targetStr))
	case OpRegex:
		return c.re != nil && c.re.MatchString(strVal)
	case OpEq:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum == c.targetNum
			}
		}
		return strings.EqualFold(strings.TrimSpace(strVal), strings.TrimSpace(c.targetStr))
	case OpNe:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum != c.targetNum
			}
		}
		return !strings.EqualFold(strings.TrimSpace(strVal), strings.TrimSpace(c.targetStr))
	case OpGt:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum > c.targetNum
			}
		}
		return strVal > c.targetStr
	case OpGte:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum >= c.targetNum
			}
		}
		return strVal >= c.targetStr
	case OpLt:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum < c.targetNum
			}
		}
		return strVal < c.targetStr
	case OpLte:
		if c.hasNum {
			if vNum, ok := toFloat64(val); ok {
				return vNum <= c.targetNum
			}
		}
		return strVal <= c.targetStr
	case OpIn:
		_, ok := c.inSet[strings.ToLower(strings.TrimSpace(strVal))]
		return ok
	default:
		return false
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func formatRecordValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
