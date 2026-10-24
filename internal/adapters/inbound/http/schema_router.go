package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

type columnView struct {
	dataset.ColumnSchema
	HasTrim  bool
	HasLower bool
	HasUpper bool
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request, dsID dataset.ID) {
	ds, err := s.ingest.GetDataset(r.Context(), dsID)
	if errors.Is(err, dataset.ErrDatasetNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("loading dataset: %v", err), http.StatusInternalServerError)
		return
	}

	cols := make([]columnView, len(ds.Schema.Columns))
	for i, c := range ds.Schema.Columns {
		cv := columnView{ColumnSchema: c}
		for _, rs := range c.RuleSpecs {
			switch rs.Type {
			case "trim_space":
				cv.HasTrim = true
			case "to_lower":
				cv.HasLower = true
			case "to_upper":
				cv.HasUpper = true
			}
		}
		cols[i] = cv
	}

	renderPage(w, schemaTmpl, http.StatusOK, map[string]any{
		"Dataset": ds,
		"Columns": cols,
		"AllTypes": []dataset.DataType{
			dataset.TypeString,
			dataset.TypeInteger,
			dataset.TypeFloat,
			dataset.TypeBoolean,
			dataset.TypeDateTime,
			dataset.TypeEmail,
			dataset.TypeUUID,
			dataset.TypeJSON,
		},
	})
}

func (s *Server) handlePostSchema(w http.ResponseWriter, r *http.Request, dsID dataset.ID) {
	ds, err := s.ingest.GetDataset(r.Context(), dsID)
	if errors.Is(err, dataset.ErrDatasetNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("loading dataset: %v", err), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form payload", http.StatusBadRequest)
		return
	}

	updatedCols := make([]dataset.ColumnSchema, len(ds.Schema.Columns))
	for i, col := range ds.Schema.Columns {
		prefix := fmt.Sprintf("col_%d_", col.Index)
		if name := strings.TrimSpace(r.FormValue(prefix + "name")); name != "" {
			col.NormalizedName = dataset.NormalizeIdentifier(name)
		}
		if t := strings.TrimSpace(r.FormValue(prefix + "type")); t != "" {
			col.TargetType = dataset.DataType(t)
		}
		if _, hasForm := r.Form[prefix+"name"]; hasForm {
			col.Nullable = r.FormValue(prefix+"nullable") == "1" || r.FormValue(prefix+"nullable") == "true"
		}

		if ruleNames, ok := r.Form[prefix+"rules"]; ok {
			specs := make([]dataset.RuleSpec, 0, len(ruleNames))
			for _, rn := range ruleNames {
				if rn != "" {
					specs = append(specs, dataset.RuleSpec{Type: rn})
				}
			}
			col.RuleSpecs = specs
		} else if _, hasForm := r.Form[prefix+"name"]; hasForm {
			col.RuleSpecs = nil
		}
		updatedCols[i] = col
	}

	if err := s.schema.UpdateSchema(r.Context(), dsID, dataset.Schema{Columns: updatedCols}); err != nil {
		http.Error(w, fmt.Sprintf("updating schema: %v", err), http.StatusUnprocessableEntity)
		return
	}

	s.handleGetSchema(w, r, dsID)
}
