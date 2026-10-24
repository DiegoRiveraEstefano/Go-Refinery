package http

import (
	"fmt"
	"net/http"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/ports"
)

func (s *Server) registerDashboardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/", s.handleDashboard)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/datasets" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var list []dataset.Dataset
	if s.ingest != nil {
		var err error
		list, err = s.ingest.ListDatasets(r.Context(), ports.PageFilter{Limit: 50})
		if err != nil {
			http.Error(w, fmt.Sprintf("loading datasets: %v", err), http.StatusInternalServerError)
			return
		}
	}

	renderPage(w, dashboardTmpl, http.StatusOK, map[string]any{
		"Datasets": list,
	})
}
