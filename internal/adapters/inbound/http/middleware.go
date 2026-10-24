package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/diego/go-refinery/internal/logger"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.bytes += int64(n)
	return n, err
}

func (sr *statusRecorder) Flush() {
	if flusher, ok := sr.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
		)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newRequestID()

		ctx := logger.WithRequestID(r.Context(), reqID)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				s.log.ErrorContext(ctx, "recovered from http handler panic",
					logger.F("method", r.Method),
					logger.F("path", r.URL.Path),
					logger.F("panic", recovered),
				)
				http.Error(rec, "internal server error", http.StatusInternalServerError)
			}

			fields := []logger.Field{
				logger.F("method", r.Method),
				logger.F("path", r.URL.Path),
				logger.F("status", rec.status),
				logger.F("bytes", rec.bytes),
				logger.F("duration_ms", time.Since(start).Milliseconds()),
				logger.F("remote_addr", r.RemoteAddr),
			}

			switch {
			case rec.status >= 500:
				s.log.ErrorContext(ctx, "http request completed with server error", fields...)
			case rec.status >= 400:
				s.log.WarnContext(ctx, "http request completed with client error", fields...)
			default:
				s.log.InfoContext(ctx, "http request completed", fields...)
			}
		}()

		next.ServeHTTP(rec, r)
	})
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-fallback"
	}
	return hex.EncodeToString(b[:])
}
