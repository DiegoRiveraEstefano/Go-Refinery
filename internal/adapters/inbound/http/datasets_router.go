package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
)

func (s *Server) registerDatasetRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/datasets", s.handleDatasetsCollection)
	mux.HandleFunc("/datasets/", s.handleDatasetsSubroutes)
}

func (s *Server) handleDatasetsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleDashboard(w, r)
	case http.MethodPost:
		s.handleUploadDataset(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDatasetsSubroutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/datasets/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	dsID := dataset.ID(parts[0])
	action := parts[1]

	switch action {
	case "schema":
		switch r.Method {
		case http.MethodGet:
			s.handleGetSchema(w, r, dsID)
		case http.MethodPost:
			s.handlePostSchema(w, r, dsID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "jobs":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleStartJob(w, r, dsID)
	case "records":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleListRecords(w, r, dsID)
	default:
		http.NotFound(w, r)
	}
}

// handleUploadDataset streams the uploaded multipart file directly to storage without
// buffering the entire file in memory.
func (s *Server) handleUploadDataset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUploadBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "expected multipart/form-data upload", http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("reading multipart stream: %v", err), http.StatusBadRequest)
			return
		}

		if part.FormName() != "file" || part.FileName() == "" {
			_ = part.Close()
			continue
		}

		filename := part.FileName()
		format, err := parsers.DetectFormat(filename)
		if err != nil {
			_ = part.Close()
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		ds, err := s.ingest.UploadAndInfer(r.Context(), ports.UploadInput{
			OriginalName: filename,
			Format:       format,
			Reader:       part,
		})
		_ = part.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("ingesting dataset: %v", err), http.StatusUnprocessableEntity)
			return
		}

		targetURL := fmt.Sprintf("/datasets/%s/schema", ds.ID)
		w.Header().Set("X-Up-Location", targetURL)
		http.Redirect(w, r, targetURL, http.StatusSeeOther)
		return
	}

	http.Error(w, "no file part found in multipart request", http.StatusBadRequest)
}

func (s *Server) handleListRecords(w http.ResponseWriter, r *http.Request, dsID dataset.ID) {
	ds, err := s.ingest.GetDataset(r.Context(), dsID)
	if errors.Is(err, dataset.ErrDatasetNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("loading dataset: %v", err), http.StatusInternalServerError)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	records, err := s.refinery.ListRecords(r.Context(), dsID, ports.PageFilter{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("loading records: %v", err), http.StatusInternalServerError)
		return
	}

	var jobErrors []job.RowError
	if jobID := r.URL.Query().Get("job_id"); jobID != "" {
		jobErrors, _ = s.refinery.ListJobErrors(r.Context(), job.ID(jobID), ports.PageFilter{Limit: 100})
	}

	renderPage(w, recordsTmpl, http.StatusOK, map[string]any{
		"Dataset": ds,
		"Columns": ds.Schema.Columns,
		"Records": records,
		"Errors":  jobErrors,
	})
}
