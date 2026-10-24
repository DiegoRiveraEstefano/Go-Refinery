package application

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/internal/ports"
)

const exportBatchPageSize = 500

// ExportRecords streams all refined records for datasetID into w in the requested format
// (CSV, TSV, JSON, or JSONL) using bounded pagination batches.
func (s *RefineryService) ExportRecords(
	ctx context.Context,
	datasetID dataset.ID,
	format ports.ExportFormat,
	w io.Writer,
) error {
	ds, err := s.datasets.GetByID(ctx, datasetID)
	if err != nil {
		return err
	}

	cols := ds.Schema.Columns
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.NormalizedName
	}

	s.log.InfoContext(ctx, "exporting refined dataset records",
		logger.F("dataset_id", datasetID),
		logger.F("format", format),
		logger.F("columns", len(headers)),
	)

	var exportedCount int64
	switch format {
	case ports.ExportCSV:
		exportedCount, err = s.exportDelimited(ctx, datasetID, headers, ',', w)
	case ports.ExportTSV:
		exportedCount, err = s.exportDelimited(ctx, datasetID, headers, '\t', w)
	case ports.ExportJSONL:
		exportedCount, err = s.exportJSONL(ctx, datasetID, w)
	case ports.ExportJSON:
		exportedCount, err = s.exportJSONArray(ctx, datasetID, w)
	default:
		return fmt.Errorf("%w: export format %q", dataset.ErrUnsupportedFormat, format)
	}

	if err != nil {
		s.log.ErrorContext(ctx, "dataset export failed",
			logger.F("dataset_id", datasetID),
			logger.F("format", format),
			logger.Err(err),
		)
		return err
	}

	s.log.InfoContext(ctx, "dataset export completed",
		logger.F("dataset_id", datasetID),
		logger.F("format", format),
		logger.F("exported_rows", exportedCount),
	)
	return nil
}

func (s *RefineryService) exportDelimited(
	ctx context.Context,
	datasetID dataset.ID,
	headers []string,
	comma rune,
	w io.Writer,
) (int64, error) {
	cw := csv.NewWriter(w)
	cw.Comma = comma

	if err := cw.Write(headers); err != nil {
		return 0, fmt.Errorf("writing csv header: %w", err)
	}

	var (
		offset int
		total  int64
		rowBuf = make([]string, len(headers))
	)

	for {
		batch, err := s.records.ListRecords(ctx, datasetID, ports.PageFilter{
			Limit:  exportBatchPageSize,
			Offset: offset,
		})
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}

		for _, rec := range batch {
			for i, key := range headers {
				rowBuf[i] = sanitizeSpreadsheetCell(formatExportCell(rec.Values[key]))
			}
			if err := cw.Write(rowBuf); err != nil {
				return total, fmt.Errorf("writing csv row %d: %w", rec.RowNumber, err)
			}
			total++
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return total, err
		}

		if len(batch) < exportBatchPageSize {
			break
		}
		offset += len(batch)
	}

	cw.Flush()
	return total, cw.Error()
}

func (s *RefineryService) exportJSONL(ctx context.Context, datasetID dataset.ID, w io.Writer) (int64, error) {
	var (
		offset int
		total  int64
		enc    = json.NewEncoder(w)
	)
	enc.SetEscapeHTML(false)

	for {
		batch, err := s.records.ListRecords(ctx, datasetID, ports.PageFilter{
			Limit:  exportBatchPageSize,
			Offset: offset,
		})
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}

		for _, rec := range batch {
			if err := enc.Encode(rec.Values); err != nil {
				return total, fmt.Errorf("encoding jsonl row %d: %w", rec.RowNumber, err)
			}
			total++
		}

		if len(batch) < exportBatchPageSize {
			break
		}
		offset += len(batch)
	}

	return total, nil
}

func (s *RefineryService) exportJSONArray(ctx context.Context, datasetID dataset.ID, w io.Writer) (int64, error) {
	if _, err := io.WriteString(w, "[\n"); err != nil {
		return 0, err
	}

	var (
		offset int
		total  int64
	)

	for {
		batch, err := s.records.ListRecords(ctx, datasetID, ports.PageFilter{
			Limit:  exportBatchPageSize,
			Offset: offset,
		})
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}

		for _, rec := range batch {
			if total > 0 {
				if _, err := io.WriteString(w, ",\n"); err != nil {
					return total, err
				}
			}
			b, err := json.MarshalIndent(rec.Values, "  ", "  ")
			if err != nil {
				return total, fmt.Errorf("marshaling json row %d: %w", rec.RowNumber, err)
			}
			if _, err := io.WriteString(w, "  "); err != nil {
				return total, err
			}
			if _, err := w.Write(b); err != nil {
				return total, err
			}
			total++
		}

		if len(batch) < exportBatchPageSize {
			break
		}
		offset += len(batch)
	}

	if _, err := io.WriteString(w, "\n]\n"); err != nil {
		return total, err
	}
	return total, nil
}

func formatExportCell(v any) string {
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

func sanitizeSpreadsheetCell(val string) string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return val
	}
	switch trimmed[0] {
	case '=', '+', '@':
		return "'" + val
	default:
		return val
	}
}
