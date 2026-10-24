// Command refinery is the composition root and HTTP entrypoint for Go-Refinery.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	inboundhttp "github.com/diego/go-refinery/internal/adapters/inbound/http"
	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/adapters/outbound/sqlite"
	"github.com/diego/go-refinery/internal/adapters/outbound/sse"
	"github.com/diego/go-refinery/internal/adapters/outbound/storage"
	"github.com/diego/go-refinery/internal/application"
	"github.com/diego/go-refinery/migrations"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "HTTP listen address")
		storageDir = flag.String("storage-dir", "./data/uploads", "Directory for raw uploaded datasets")
		dbDriver   = flag.String("db-driver", "sqlite", "database/sql driver name (e.g. sqlite)")
		dbDSN      = flag.String("db-dsn", "./data/refinery.db", "SQLite database file path or DSN")
		workers    = flag.Int("workers", runtime.NumCPU(), "Number of background refinement workers")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	localStore, err := storage.NewLocalStorage(*storageDir)
	if err != nil {
		log.Fatalf("initializing local storage: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*dbDSN), 0o700); err != nil {
		log.Fatalf("creating database directory: %v", err)
	}

	sqlStore, err := sqlite.OpenStore(ctx, *dbDriver, *dbDSN)
	if err != nil {
		log.Fatalf("opening sqlite store: %v", err)
	}
	defer sqlStore.Close()

	if ddlBytes, err := migrations.FS.ReadFile("00001_initial_schema.sql"); err == nil {
		if err := sqlStore.ExecSchema(ctx, string(ddlBytes)); err != nil {
			log.Printf("warning: applying initial schema: %v", err)
		}
	}

	parserFactory := parsers.NewFactory()
	sseBroker := sse.NewBroker()

	ingestSvc := application.NewIngestService(sqlStore.Datasets(), localStore, parserFactory)
	schemaSvc := application.NewSchemaService(sqlStore.Datasets())
	refinerySvc := application.NewRefineryService(
		sqlStore.Datasets(),
		sqlStore.Jobs(),
		sqlStore.Records(),
		localStore,
		parserFactory,
		sseBroker,
		64,
	)

	refinerySvc.StartWorkers(ctx, *workers)

	httpServer := inboundhttp.NewServer(ingestSvc, schemaSvc, refinerySvc, sseBroker)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("Go-Refinery listening on %s (workers=%d)", *addr, *workers)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down Go-Refinery gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}

	log.Println("waiting for background refinement workers to finish...")
	refinerySvc.Wait()
}
