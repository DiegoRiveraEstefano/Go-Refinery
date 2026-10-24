package application_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/adapters/outbound/sse"
	"github.com/diego/go-refinery/internal/adapters/outbound/storage"
	"github.com/diego/go-refinery/internal/application"
	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
)

type memRepo struct {
	mu       sync.RWMutex
	datasets map[dataset.ID]*dataset.Dataset
	jobs     map[job.ID]*job.Job
	records  map[dataset.ID][]dataset.Record
	errors   map[job.ID][]job.RowError
}

func newMemRepo() *memRepo {
	return &memRepo{
		datasets: make(map[dataset.ID]*dataset.Dataset),
		jobs:     make(map[job.ID]*job.Job),
		records:  make(map[dataset.ID][]dataset.Record),
		errors:   make(map[job.ID][]job.RowError),
	}
}

func (m *memRepo) Create(_ context.Context, ds *dataset.Dataset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *ds
	m.datasets[ds.ID] = &cp
	return nil
}

func (m *memRepo) GetByID(_ context.Context, id dataset.ID) (*dataset.Dataset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ds, ok := m.datasets[id]
	if !ok {
		return nil, dataset.ErrDatasetNotFound
	}
	cp := *ds
	return &cp, nil
}

func (m *memRepo) List(_ context.Context, _ ports.PageFilter) ([]dataset.Dataset, error) {
	return nil, nil
}

func (m *memRepo) UpdateStatus(_ context.Context, id dataset.ID, status dataset.Status, totalRows int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ds, ok := m.datasets[id]
	if !ok {
		return dataset.ErrDatasetNotFound
	}
	ds.Status = status
	if totalRows > 0 {
		ds.TotalRows = totalRows
	}
	return nil
}

func (m *memRepo) SaveSchema(_ context.Context, id dataset.ID, schema dataset.Schema) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ds, ok := m.datasets[id]
	if !ok {
		return dataset.ErrDatasetNotFound
	}
	ds.Schema = schema
	return nil
}

func (m *memRepo) CreateJob(_ context.Context, j *job.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	m.jobs[j.ID] = &cp
	return nil
}

func (m *memRepo) GetJobByID(_ context.Context, id job.ID) (*job.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, job.ErrJobNotFound
	}
	cp := *j
	return &cp, nil
}

func (m *memRepo) GetLatestByDataset(_ context.Context, datasetID dataset.ID) (*job.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, j := range m.jobs {
		if j.DatasetID == datasetID {
			cp := *j
			return &cp, nil
		}
	}
	return nil, job.ErrJobNotFound
}

func (m *memRepo) UpdateProgress(_ context.Context, j *job.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	m.jobs[j.ID] = &cp
	return nil
}

func (m *memRepo) ListErrors(_ context.Context, jobID job.ID, _ ports.PageFilter) ([]job.RowError, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]job.RowError(nil), m.errors[jobID]...), nil
}

func (m *memRepo) WriteBatch(_ context.Context, jobID job.ID, records []dataset.Record, rowErrors []job.RowError) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range records {
		m.records[r.DatasetID] = append(m.records[r.DatasetID], r)
	}
	m.errors[jobID] = append(m.errors[jobID], rowErrors...)
	return nil
}

func (m *memRepo) ListRecords(_ context.Context, datasetID dataset.ID, _ ports.PageFilter) ([]dataset.Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]dataset.Record(nil), m.records[datasetID]...), nil
}

type jobRepoAdapter struct{ *memRepo }

func (a jobRepoAdapter) Create(ctx context.Context, j *job.Job) error { return a.CreateJob(ctx, j) }
func (a jobRepoAdapter) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	return a.GetJobByID(ctx, id)
}

func TestRefineryWorkerPool_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newMemRepo()
	localStore, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage failed: %v", err)
	}
	parserFactory := parsers.NewFactory()
	broker := sse.NewBroker()

	ingestSvc := application.NewIngestService(repo, localStore, parserFactory)
	refinerySvc := application.NewRefineryService(
		repo,
		jobRepoAdapter{repo},
		repo,
		localStore,
		parserFactory,
		broker,
		16,
	)
	refinerySvc.SetBatchSize(2)
	refinerySvc.StartWorkers(ctx, 2)

	rawCSV := strings.Join([]string{
		"id,email,amount",
		"1, ALICE@EXAMPLE.COM ,10.50",
		"2,bob@example.com,20.00",
		"bad_int,charlie@example.com,30.00",
		"4,diana@example.com,40.25",
	}, "\n")

	ds, err := ingestSvc.UploadAndInfer(ctx, ports.UploadInput{
		OriginalName: "sales.csv",
		Format:       dataset.FormatCSV,
		SizeBytes:    int64(len(rawCSV)),
		Reader:       strings.NewReader(rawCSV),
	})
	if err != nil {
		t.Fatalf("UploadAndInfer failed: %v", err)
	}

	// Attach trim_space and to_lower rules on email column (index 1)
	ds.Schema.Columns[1].RuleSpecs = []dataset.RuleSpec{
		{Type: "trim_space"},
		{Type: "to_lower"},
	}
	if err := repo.SaveSchema(ctx, ds.ID, ds.Schema); err != nil {
		t.Fatalf("SaveSchema failed: %v", err)
	}

	startedJob, err := refinerySvc.StartRefinementJob(ctx, ds.ID)
	if err != nil {
		t.Fatalf("StartRefinementJob failed: %v", err)
	}

	eventsCh, unsub := broker.Subscribe(ctx, startedJob.ID)
	defer unsub()

	// Wait until job reaches COMPLETED state
	var finalJob *job.Job
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		j, err := refinerySvc.GetJob(ctx, startedJob.ID)
		if err != nil {
			t.Fatalf("GetJob error: %v", err)
		}
		if j.IsTerminal() {
			finalJob = j
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	if finalJob == nil {
		t.Fatal("timed out waiting for worker pool to finish job")
	}
	if finalJob.Status != job.StatusCompleted {
		t.Fatalf("expected job COMPLETED, got %s (msg=%s)", finalJob.Status, finalJob.ErrorMessage)
	}
	if finalJob.ProcessedRows != 4 || finalJob.ValidRows != 3 || finalJob.ErrorRows != 1 {
		t.Fatalf("unexpected row counts: processed=%d valid=%d error=%d",
			finalJob.ProcessedRows, finalJob.ValidRows, finalJob.ErrorRows)
	}

	records, err := refinerySvc.ListRecords(ctx, ds.ID, ports.PageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListRecords error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 valid records, got %d", len(records))
	}
	if records[0].Values["email"] != "alice@example.com" {
		t.Errorf("expected cleaned email 'alice@example.com', got %v", records[0].Values["email"])
	}

	jobErrs, err := refinerySvc.ListJobErrors(ctx, startedJob.ID, ports.PageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListJobErrors error: %v", err)
	}
	if len(jobErrs) != 1 || jobErrs[0].RowNumber != 3 || jobErrs[0].ErrorCode != "ERR_TYPE_CAST" {
		t.Fatalf("unexpected job errors: %+v", jobErrs)
	}

	select {
	case ev := <-eventsCh:
		if ev.JobID != startedJob.ID {
			t.Errorf("expected SSE event for job %s, got %s", startedJob.ID, ev.JobID)
		}
	default:
		t.Error("expected at least one SSE progress event published to broker")
	}

	cancel()
	refinerySvc.Wait()
}
