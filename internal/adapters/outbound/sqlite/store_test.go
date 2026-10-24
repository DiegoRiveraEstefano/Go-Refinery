package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/diego/go-refinery/internal/adapters/outbound/sqlite"
	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
	"github.com/diego/go-refinery/migrations"
)

func TestStore_RepositoriesLifecycle(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_refinery.db")

	store, err := sqlite.OpenStore(ctx, "sqlite", dbPath)
	if err != nil {
		t.Fatalf("OpenStore error: %v", err)
	}
	defer store.Close()

	ddlBytes, err := migrations.FS.ReadFile("00001_initial_schema.sql")
	if err != nil {
		t.Fatalf("reading embedded migration: %v", err)
	}
	if err := store.ExecSchema(ctx, string(ddlBytes)); err != nil {
		t.Fatalf("ExecSchema error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	ds := &dataset.Dataset{
		ID:           "ds-001",
		OriginalName: "customers.csv",
		StoragePath:  "/tmp/ds-001.raw",
		Format:       dataset.FormatCSV,
		Delimiter:    ',',
		HasHeader:    true,
		SizeBytes:    2048,
		TotalRows:    100,
		Status:       dataset.StatusDraft,
		SampleRows:   [][]string{{"1", "alice@example.com"}},
		Schema: dataset.Schema{
			Columns: []dataset.ColumnSchema{
				{
					Index:          0,
					OriginalName:   "ID",
					NormalizedName: "id",
					InferredType:   dataset.TypeInteger,
					TargetType:     dataset.TypeInteger,
					Nullable:       false,
					SampleDistinct: 1,
					SampleValues:   []string{"1"},
					RuleSpecs:      []dataset.RuleSpec{{Type: "trim_space"}},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 1. DatasetRepository: Create & GetByID
	if err := store.Datasets().Create(ctx, ds); err != nil {
		t.Fatalf("Create dataset failed: %v", err)
	}

	gotDS, err := store.Datasets().GetByID(ctx, "ds-001")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if gotDS.OriginalName != "customers.csv" || len(gotDS.Schema.Columns) != 1 {
		t.Fatalf("unexpected dataset fetched: %+v", gotDS)
	}
	if gotDS.Schema.Columns[0].NormalizedName != "id" || len(gotDS.Schema.Columns[0].RuleSpecs) != 1 {
		t.Fatalf("unexpected schema column fetched: %+v", gotDS.Schema.Columns[0])
	}

	// Verify not found sentinel error
	if _, err := store.Datasets().GetByID(ctx, "missing"); !errors.Is(err, dataset.ErrDatasetNotFound) {
		t.Fatalf("expected ErrDatasetNotFound, got %v", err)
	}

	// 2. DatasetRepository: SaveSchema & UpdateStatus
	ds.Schema.Columns[0].TargetType = dataset.TypeString
	if err := store.Datasets().SaveSchema(ctx, ds.ID, ds.Schema); err != nil {
		t.Fatalf("SaveSchema failed: %v", err)
	}
	if err := store.Datasets().UpdateStatus(ctx, ds.ID, dataset.StatusReady, 250); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	gotDS, err = store.Datasets().GetByID(ctx, ds.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if gotDS.Status != dataset.StatusReady || gotDS.TotalRows != 250 || gotDS.Schema.Columns[0].TargetType != dataset.TypeString {
		t.Fatalf("expected updated dataset state, got status=%s totalRows=%d targetType=%s",
			gotDS.Status, gotDS.TotalRows, gotDS.Schema.Columns[0].TargetType)
	}

	// 3. JobRepository: Create, GetByID, UpdateProgress, GetLatestByDataset
	j := &job.Job{
		ID:        "job-001",
		DatasetID: ds.ID,
		Status:    job.StatusQueued,
		TotalRows: 250,
		MaxErrors: 50,
		CreatedAt: now,
	}
	if err := store.Jobs().Create(ctx, j); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	j.Status = job.StatusCompleted
	j.ProcessedRows = 250
	j.ValidRows = 249
	j.ErrorRows = 1
	j.StartedAt = &now
	j.FinishedAt = &now
	if err := store.Jobs().UpdateProgress(ctx, j); err != nil {
		t.Fatalf("UpdateProgress failed: %v", err)
	}

	latestJob, err := store.Jobs().GetLatestByDataset(ctx, ds.ID)
	if err != nil {
		t.Fatalf("GetLatestByDataset failed: %v", err)
	}
	if latestJob.Status != job.StatusCompleted || latestJob.ValidRows != 249 {
		t.Fatalf("unexpected job state: %+v", latestJob)
	}

	// 4. RecordBatchWriter: WriteBatch, ListRecords, ListErrors
	records := []dataset.Record{
		{
			DatasetID: ds.ID,
			RowNumber: 1,
			Values:    map[string]any{"id": "101"},
			CreatedAt: now,
		},
	}
	rowErrors := []job.RowError{
		{
			JobID:      j.ID,
			RowNumber:  2,
			ColumnName: "id",
			RawValue:   "bad",
			ErrorCode:  "ERR_CAST",
			Message:    "cannot cast to integer",
			CreatedAt:  now,
		},
	}

	if err := store.Records().WriteBatch(ctx, j.ID, records, rowErrors); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}

	gotRecs, err := store.Records().ListRecords(ctx, ds.ID, ports.PageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(gotRecs) != 1 || gotRecs[0].Values["id"] != "101" {
		t.Fatalf("unexpected records returned: %+v", gotRecs)
	}

	gotErrs, err := store.Jobs().ListErrors(ctx, j.ID, ports.PageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListErrors failed: %v", err)
	}
	if len(gotErrs) != 1 || gotErrs[0].ErrorCode != "ERR_CAST" {
		t.Fatalf("unexpected job errors returned: %+v", gotErrs)
	}
}
