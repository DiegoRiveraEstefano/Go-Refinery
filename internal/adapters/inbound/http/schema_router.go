package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/pipeline"
	"github.com/diego/go-refinery/internal/logger"
)

type ruleView struct {
	Type           string
	Label          string
	Description    string
	Category       string
	SupportedCSV   string
	Checked        bool
	VisibleInitial bool
}

type columnView struct {
	dataset.ColumnSchema
	AppliedBlocks  []ruleView
	AvailableRules []ruleView
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request, dsID dataset.ID) {
	ds, err := s.ingest.GetDataset(r.Context(), dsID)
	if errors.Is(err, dataset.ErrDatasetNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.log.ErrorContext(r.Context(), "loading dataset for schema view failed",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
		http.Error(w, fmt.Sprintf("loading dataset: %v", err), http.StatusInternalServerError)
		return
	}

	catalog := pipeline.AvailableRules()
	catalogByType := make(map[string]pipeline.RuleDescriptor, len(catalog))
	for _, desc := range catalog {
		catalogByType[desc.Type] = desc
	}

	cols := make([]columnView, len(ds.Schema.Columns))

	for i, c := range ds.Schema.Columns {
		activeRules := make(map[string]bool, len(c.RuleSpecs))
		appliedList := make([]ruleView, 0, len(c.RuleSpecs))

		for _, rs := range c.RuleSpecs {
			activeRules[rs.Type] = true
			desc, ok := catalogByType[rs.Type]
			if !ok {
				continue
			}
			supportedStrs := make([]string, len(desc.SupportedTypes))
			compatible := false
			for k, st := range desc.SupportedTypes {
				supportedStrs[k] = string(st)
				if st == c.TargetType {
					compatible = true
				}
			}
			appliedList = append(appliedList, ruleView{
				Type:           desc.Type,
				Label:          desc.Label,
				Description:    desc.Description,
				Category:       string(desc.Category),
				SupportedCSV:   strings.Join(supportedStrs, ","),
				Checked:        true,
				VisibleInitial: compatible,
			})
		}

		rvList := make([]ruleView, len(catalog))
		for j, desc := range catalog {
			supportedStrs := make([]string, len(desc.SupportedTypes))
			visible := false
			for k, st := range desc.SupportedTypes {
				supportedStrs[k] = string(st)
				if st == c.TargetType {
					visible = true
				}
			}
			rvList[j] = ruleView{
				Type:           desc.Type,
				Label:          desc.Label,
				Description:    desc.Description,
				Category:       string(desc.Category),
				SupportedCSV:   strings.Join(supportedStrs, ","),
				Checked:        activeRules[desc.Type],
				VisibleInitial: visible,
			}
		}

		cols[i] = columnView{
			ColumnSchema:   c,
			AppliedBlocks:  appliedList,
			AvailableRules: rvList,
		}
	}

	activeCol := 0
	if len(cols) > 0 {
		activeCol = cols[0].Index
	}
	if rawActive := strings.TrimSpace(r.FormValue("active_col")); rawActive != "" {
		if parsed, err := strconv.Atoi(rawActive); err == nil {
			for _, col := range cols {
				if col.Index == parsed {
					activeCol = parsed
					break
				}
			}
		}
	}

	renderPage(w, schemaTmpl, http.StatusOK, map[string]any{
		"Dataset":   ds,
		"Columns":   cols,
		"ActiveCol": activeCol,
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
		s.log.ErrorContext(r.Context(), "loading dataset for schema update failed",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
		http.Error(w, fmt.Sprintf("loading dataset: %v", err), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.log.WarnContext(r.Context(), "invalid schema form payload",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
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
		s.log.WarnContext(r.Context(), "schema validation failed",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
		http.Error(w, fmt.Sprintf("updating schema: %v", err), http.StatusUnprocessableEntity)
		return
	}

	s.handleGetSchema(w, r, dsID)
}
