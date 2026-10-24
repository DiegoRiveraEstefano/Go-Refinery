// Command refinery is the composition root and HTTP entrypoint for Go-Refinery.
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	inboundhttp "github.com/diego/go-refinery/internal/adapters/inbound/http"
	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/adapters/outbound/sqlite"
	"github.com/diego/go-refinery/internal/adapters/outbound/sse"
	"github.com/diego/go-refinery/internal/adapters/outbound/storage"
	"github.com/diego/go-refinery/internal/application"
	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/migrations"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "HTTP listen address")
		storageDir = flag.String("storage-dir", "./data/uploads", "Directory for raw uploaded datasets")
		dbDriver   = flag.String("db-driver", "sqlite", "database/sql driver name (e.g. sqlite)")
		dbDSN      = flag.String("db-dsn", "./data/refinery.db", "SQLite database file path or DSN")
		workers    = flag.Int("workers", runtime.NumCPU(), "Number of background refinement workers")
		logLevel   = flag.String("log-level", "info", "Minimum log level: debug, info, warn, error")
		logFormat  = flag.String("log-format", "text", "Log output format: text or json")
	)
	flag.Parse()

	rootLog := logger.New(
		os.Stderr,
		logger.ParseLevel(*logLevel),
		strings.EqualFold(strings.TrimSpace(*logFormat), "json"),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	localStore, err := storage.NewLocalStorage(*storageDir)
	if err != nil {
		rootLog.Error("failed to initialize local storage", logger.F("dir", *storageDir), logger.Err(err))
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*dbDSN), 0o700); err != nil {
		rootLog.Error("failed to create database directory", logger.F("dsn", *dbDSN), logger.Err(err))
		os.Exit(1)
	}

	sqlStore, err := sqlite.OpenStore(ctx, *dbDriver, *dbDSN)
	if err != nil {
		rootLog.Error("failed to open sqlite store", logger.F("driver", *dbDriver), logger.F("dsn", *dbDSN), logger.Err(err))
		os.Exit(1)
	}
	defer sqlStore.Close()

	if ddlBytes, err := migrations.FS.ReadFile("00001_initial_schema.sql"); err == nil {
		if err := sqlStore.ExecSchema(ctx, string(ddlBytes)); err != nil {
			rootLog.Warn("failed to apply initial schema migration", logger.Err(err))
		} else {
			rootLog.Debug("verified initial sqlite database schema")
		}
	}

	parserFactory := parsers.NewFactory()
	sseBroker := sse.NewBroker()

	ingestSvc := application.NewIngestService(sqlStore.Datasets(), localStore, parserFactory)
	ingestSvc.SetLogger(rootLog)

	schemaSvc := application.NewSchemaService(sqlStore.Datasets())
	schemaSvc.SetLogger(rootLog)

	refinerySvc := application.NewRefineryService(
		sqlStore.Datasets(),
		sqlStore.Jobs(),
		sqlStore.Records(),
		localStore,
		parserFactory,
		sseBroker,
		64,
	)
	refinerySvc.SetLogger(rootLog)
	refinerySvc.StartWorkers(ctx, *workers)

	httpServer := inboundhttp.NewServer(ingestSvc, schemaSvc, refinerySvc, sseBroker)
	httpServer.SetLogger(rootLog)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		rootLog.Info("go-refinery http server listening",
			logger.F("addr", *addr),
			logger.F("workers", *workers),
			logger.F("db_dsn", *dbDSN),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rootLog.Error("http server terminated with error", logger.Err(err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	rootLog.Info("shutting down go-refinery gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		rootLog.Warn("http server graceful shutdown error", logger.Err(err))
	}

	rootLog.Info("waiting for background refinement workers to finish...")
	refinerySvc.Wait()
	rootLog.Info("go-refinery shutdown complete")
}
