package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/logger"
)

func (s *Server) registerJobRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/jobs/", s.handleJobsSubroutes)
}

func (s *Server) handleStartJob(w http.ResponseWriter, r *http.Request, dsID dataset.ID) {
	j, err := s.refinery.StartRefinementJob(r.Context(), dsID)
	if errors.Is(err, dataset.ErrDatasetNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, job.ErrJobAlreadyActive) {
		s.log.WarnContext(r.Context(), "refinement job conflict",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		s.log.ErrorContext(r.Context(), "starting refinement job failed",
			logger.F("dataset_id", string(dsID)),
			logger.Err(err),
		)
		http.Error(w, fmt.Sprintf("starting job: %v", err), http.StatusInternalServerError)
		return
	}

	renderPage(w, progressTmpl, http.StatusAccepted, map[string]any{
		"Job":        j,
		"Percentage": j.Percentage(),
	})
}

func (s *Server) handleJobsSubroutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/jobs/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	jobID := job.ID(parts[0])
	if len(parts) == 1 && r.Method == http.MethodGet {
		j, err := s.refinery.GetJob(r.Context(), jobID)
		if errors.Is(err, job.ErrJobNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			s.log.ErrorContext(r.Context(), "loading job failed",
				logger.F("job_id", string(jobID)),
				logger.Err(err),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderPage(w, progressTmpl, http.StatusOK, map[string]any{
			"Job":        j,
			"Percentage": j.Percentage(),
		})
		return
	}

	if len(parts) == 2 && parts[1] == "events" && r.Method == http.MethodGet {
		s.handleJobSSE(w, r, jobID)
		return
	}

	http.NotFound(w, r)
}

// handleJobSSE streams real-time job.ProgressEvent updates over Server-Sent Events (SSE)
// using standard net/http and http.Flusher.
func (s *Server) handleJobSSE(w http.ResponseWriter, r *http.Request, jobID job.ID) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported by response writer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	s.log.DebugContext(r.Context(), "sse client subscribed",
		logger.F("job_id", string(jobID)),
	)

	// Send initial snapshot if job exists.
	if s.refinery != nil {
		if j, err := s.refinery.GetJob(r.Context(), jobID); err == nil && j != nil {
			writeSSEEvent(w, flusher, "progress", job.ProgressEvent{
				JobID:         j.ID,
				DatasetID:     j.DatasetID,
				Status:        j.Status,
				ProcessedRows: j.ProcessedRows,
				TotalRows:     j.TotalRows,
				ValidRows:     j.ValidRows,
				ErrorRows:     j.ErrorRows,
				Percentage:    j.Percentage(),
				Message:       j.ErrorMessage,
			})
			if j.IsTerminal() {
				return
			}
		}
	}

	if s.broker == nil {
		return
	}

	eventsCh, unsubscribe := s.broker.Subscribe(r.Context(), jobID)
	defer func() {
		unsubscribe()
		s.log.DebugContext(r.Context(), "sse client disconnected",
			logger.F("job_id", string(jobID)),
		)
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-eventsCh:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, "progress", ev)
			if ev.Status == job.StatusCompleted || ev.Status == job.StatusFailed || ev.Status == job.StatusCancelled {
				return
			}
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventName string, payload job.ProgressEvent) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, data)
	flusher.Flush()
}
