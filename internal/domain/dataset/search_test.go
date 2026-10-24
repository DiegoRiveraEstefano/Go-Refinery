package dataset_test

import (
	"testing"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

func TestCompileSearchQuery_Modes(t *testing.T) {
	rows := []dataset.Record{
		{
			RowNumber: 1,
			Values: map[string]any{
				"id":      int64(1),
				"name":    "Alice Smith",
				"age":     int64(32),
				"country": "ES",
				"email":   "alice@example.com",
			},
		},
		{
			RowNumber: 2,
			Values: map[string]any{
				"id":      int64(2),
				"name":    "Bob Jones",
				"age":     int64(19),
				"country": "US",
				"email":   "bob@corp.org",
			},
		},
		{
			RowNumber: 3,
			Values: map[string]any{
				"id":      int64(3),
				"name":    "Carlos Ruiz",
				"age":     int64(45),
				"country": "MX",
				"email":   nil,
			},
		},
	}

	tests := []struct {
		name        string
		query       dataset.SearchQuery
		expectedIDs []int64
	}{
		{
			name:        "FreeText_GlobalSubstring",
			query:       dataset.SearchQuery{Query: "jones"},
			expectedIDs: []int64{2},
		},
		{
			name:        "SlashRegex_CaseInsensitive",
			query:       dataset.SearchQuery{Query: `/^(alice|carlos)/i`},
			expectedIDs: []int64{1, 3},
		},
		{
			name: "StructuredOperator_NumericGt",
			query: dataset.SearchQuery{
				Field:    "age",
				Operator: dataset.OpGt,
				Value:    "25",
			},
			expectedIDs: []int64{1, 3},
		},
		{
			name: "MongoJSON_CompoundAndRegex",
			query: dataset.SearchQuery{
				Query: `{"age": {"$gte": 30}, "name": {"$regex": "(?i)ruiz$"}}`,
			},
			expectedIDs: []int64{3},
		},
		{
			name: "MongoJSON_LogicalOrAndIn",
			query: dataset.SearchQuery{
				Query: `{"$or": [{"country": {"$in": ["ES", "AR"]}}, {"age": {"$lt": 20}}]}`,
			},
			expectedIDs: []int64{1, 2},
		},
		{
			name: "MiniDSL_Expression",
			query: dataset.SearchQuery{
				Query: `age >= 19 AND country != MX`,
			},
			expectedIDs: []int64{1, 2},
		},
		{
			name: "ExistsOperator_FalseFindsNullEmail",
			query: dataset.SearchQuery{
				Field:    "email",
				Operator: dataset.OpExists,
				Value:    "false",
			},
			expectedIDs: []int64{3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := dataset.CompileSearchQuery(tc.query)
			if err != nil {
				t.Fatalf("CompileSearchQuery(%+v) unexpected error: %v", tc.query, err)
			}

			var matched []int64
			for _, r := range rows {
				if compiled.Match(r) {
					matched = append(matched, r.RowNumber)
				}
			}

			if len(matched) != len(tc.expectedIDs) {
				t.Fatalf("expected row numbers %v, got %v", tc.expectedIDs, matched)
			}
			for i := range matched {
				if matched[i] != tc.expectedIDs[i] {
					t.Errorf("index %d: expected %d, got %d", i, tc.expectedIDs[i], matched[i])
				}
			}
		})
	}
}
