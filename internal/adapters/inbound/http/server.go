package http

import (
	"net/http"

	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/internal/ports"
	"github.com/diego/go-refinery/web"
)

const defaultMaxUploadBytes int64 = 512 * 1024 * 1024 // 512 MB

// Server wires inbound use-case ports to modular net/http route groups and embedded assets.
type Server struct {
	ingest         ports.IngestUseCase
	schema         ports.SchemaUseCase
	refinery       ports.RefineryUseCase
	broker         ports.ProgressBroker
	maxUploadBytes int64
	mux            *http.ServeMux
	log            *logger.Logger
}

// NewServer constructs an HTTP Server and registers all view/router groups.
func NewServer(
	ingest ports.IngestUseCase,
	schema ports.SchemaUseCase,
	refinery ports.RefineryUseCase,
	broker ports.ProgressBroker,
) *Server {
	s := &Server{
		ingest:         ingest,
		schema:         schema,
		refinery:       refinery,
		broker:         broker,
		maxUploadBytes: defaultMaxUploadBytes,
		mux:            http.NewServeMux(),
		log:            logger.Default().With(logger.F("component", "http_server")),
	}
	s.registerRoutes()
	return s
}

// SetLogger overrides the default structured logger on Server.
func (s *Server) SetLogger(l *logger.Logger) {
	if l != nil {
		s.log = l.With(logger.F("component", "http_server"))
	}
}

// SetMaxUploadBytes configures the maximum allowed request body size for uploads.
func (s *Server) SetMaxUploadBytes(limit int64) {
	if limit > 0 {
		s.maxUploadBytes = limit
	}
}

// Handler returns the root http.Handler with request logging, panic recovery, and security headers.
func (s *Server) Handler() http.Handler {
	return s.withRequestLogging(withSecurityHeaders(s.mux))
}

func (s *Server) registerRoutes() {
	s.mux.Handle("/static/", http.FileServer(http.FS(web.StaticFS)))
	s.registerDashboardRoutes(s.mux)
	s.registerDatasetRoutes(s.mux)
	s.registerJobRoutes(s.mux)
}
