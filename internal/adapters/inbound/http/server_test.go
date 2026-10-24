package http_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	inboundhttp "github.com/diego/go-refinery/internal/adapters/inbound/http"
	"github.com/diego/go-refinery/internal/adapters/outbound/parsers"
	"github.com/diego/go-refinery/internal/adapters/outbound/sse"
	"github.com/diego/go-refinery/internal/adapters/outbound/storage"
	"github.com/diego/go-refinery/internal/application"
	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
)

type httpTestRepo struct {
	mu       sync.RWMutex
	datasets map[dataset.ID]*dataset.Dataset
	jobs     map[job.ID]*job.Job
	records  map[dataset.ID][]dataset.Record
	errors   map[job.ID][]job.RowError
}

func newHTTPTestRepo() *httpTestRepo {
	return &httpTestRepo{
		datasets: make(map[dataset.ID]*dataset.Dataset),
		jobs:     make(map[job.ID]*job.Job),
		records:  make(map[dataset.ID][]dataset.Record),
		errors:   make(map[job.ID][]job.RowError),
	}
}

func (m *httpTestRepo) Create(_ context.Context, ds *dataset.Dataset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *ds
	m.datasets[ds.ID] = &cp
	return nil
}

func (m *httpTestRepo) GetByID(_ context.Context, id dataset.ID) (*dataset.Dataset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ds, ok := m.datasets[id]
	if !ok {
		return nil, dataset.ErrDatasetNotFound
	}
	cp := *ds
	return &cp, nil
}

func (m *httpTestRepo) List(_ context.Context, _ ports.PageFilter) ([]dataset.Dataset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []dataset.Dataset
	for _, ds := range m.datasets {
		out = append(out, *ds)
	}
	return out, nil
}

func (m *httpTestRepo) UpdateStatus(_ context.Context, id dataset.ID, status dataset.Status, totalRows int64) error {
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

func (m *httpTestRepo) SaveSchema(_ context.Context, id dataset.ID, schema dataset.Schema) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ds, ok := m.datasets[id]
	if !ok {
		return dataset.ErrDatasetNotFound
	}
	ds.Schema = schema
	return nil
}

type httpJobRepo struct{ *httpTestRepo }

func (r httpJobRepo) Create(_ context.Context, j *job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *j
	r.jobs[j.ID] = &cp
	return nil
}

func (r httpJobRepo) GetByID(_ context.Context, id job.ID) (*job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[id]
	if !ok {
		return nil, job.ErrJobNotFound
	}
	cp := *j
	return &cp, nil
}

func (r httpJobRepo) GetLatestByDataset(_ context.Context, datasetID dataset.ID) (*job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, j := range r.jobs {
		if j.DatasetID == datasetID {
			cp := *j
			return &cp, nil
		}
	}
	return nil, job.ErrJobNotFound
}

func (r httpJobRepo) UpdateProgress(_ context.Context, j *job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *j
	r.jobs[j.ID] = &cp
	return nil
}

func (r httpJobRepo) ListErrors(_ context.Context, jobID job.ID, _ ports.PageFilter) ([]job.RowError, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]job.RowError(nil), r.errors[jobID]...), nil
}

func (m *httpTestRepo) WriteBatch(_ context.Context, jobID job.ID, records []dataset.Record, rowErrors []job.RowError) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range records {
		m.records[rec.DatasetID] = append(m.records[rec.DatasetID], rec)
	}
	m.errors[jobID] = append(m.errors[jobID], rowErrors...)
	return nil
}

func (m *httpTestRepo) ListRecords(_ context.Context, datasetID dataset.ID, page ports.PageFilter) ([]dataset.Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := m.records[datasetID]
	if page.Offset >= len(all) {
		return nil, nil
	}
	end := len(all)
	if page.Limit > 0 && page.Offset+page.Limit < end {
		end = page.Offset + page.Limit
	}
	return append([]dataset.Record(nil), all[page.Offset:end]...), nil
}

func TestHTTPAdapter_UploadSchemaJobSSEAndExport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newHTTPTestRepo()
	localStore, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	parserFactory := parsers.NewFactory()
	broker := sse.NewBroker()

	ingestSvc := application.NewIngestService(repo, localStore, parserFactory)
	schemaSvc := application.NewSchemaService(repo)
	refinerySvc := application.NewRefineryService(repo, httpJobRepo{repo}, repo, localStore, parserFactory, broker, 16)
	refinerySvc.StartWorkers(ctx, 1)
	defer func() {
		cancel()
		refinerySvc.Wait()
	}()

	srv := inboundhttp.NewServer(ingestSvc, schemaSvc, refinerySvc, broker)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Multipart upload to POST /datasets
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "users.csv")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = io.WriteString(part, "id,name\n1, Alice \n2,Bob\n")
	_ = mw.Close()

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Post(ts.URL+"/datasets", mw.FormDataContentType(), &body)
	if err != nil {
		t.Fatalf("POST /datasets failed: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", resp.StatusCode)
	}
	if csp := resp.Header.Get("Content-Security-Policy"); csp == "" {
		t.Error("expected Content-Security-Policy header on response")
	}
	if reqID := resp.Header.Get("X-Request-ID"); reqID == "" {
		t.Error("expected X-Request-ID header on response")
	}

	schemaPath := resp.Header.Get("Location")
	if !strings.HasPrefix(schemaPath, "/datasets/") || !strings.HasSuffix(schemaPath, "/schema") {
		t.Fatalf("unexpected redirect location: %q", schemaPath)
	}
	dsID := strings.TrimSuffix(strings.TrimPrefix(schemaPath, "/datasets/"), "/schema")

	// 2. GET /datasets/{id}/schema
	schemaResp, err := http.Get(ts.URL + schemaPath)
	if err != nil {
		t.Fatalf("GET %s failed: %v", schemaPath, err)
	}
	htmlBytes, _ := io.ReadAll(schemaResp.Body)
	_ = schemaResp.Body.Close()
	if !strings.Contains(string(htmlBytes), "<refinery-schema-editor>") {
		t.Fatalf("expected <refinery-schema-editor> in HTML response")
	}

	// 3. POST /datasets/{id}/schema to configure trim_space and to_lower on column 1
	form := url.Values{
		"col_0_name":     {"id"},
		"col_0_type":     {"INTEGER"},
		"col_1_name":     {"name"},
		"col_1_type":     {"STRING"},
		"col_1_nullable": {"1"},
		"col_1_rules":    {"trim_space", "to_lower"},
	}
	postSchemaResp, err := http.PostForm(ts.URL+schemaPath, form)
	if err != nil {
		t.Fatalf("POST %s failed: %v", schemaPath, err)
	}
	_ = postSchemaResp.Body.Close()
	if postSchemaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from schema update, got %d", postSchemaResp.StatusCode)
	}

	// 4. POST /datasets/{id}/jobs to start refinement
	jobResp, err := http.Post(ts.URL+"/datasets/"+dsID+"/jobs", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST /datasets/%s/jobs failed: %v", dsID, err)
	}
	jobHTMLBytes, _ := io.ReadAll(jobResp.Body)
	_ = jobResp.Body.Close()
	if jobResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted when starting job, got %d", jobResp.StatusCode)
	}
	jobHTML := string(jobHTMLBytes)
	if !strings.Contains(jobHTML, "<refinery-progress") {
		t.Fatalf("expected <refinery-progress> element in response")
	}

	// Extract job ID from job-id="..."
	idx := strings.Index(jobHTML, `job-id="`)
	if idx == -1 {
		t.Fatalf("could not find job-id attribute in progress HTML")
	}
	rest := jobHTML[idx+len(`job-id="`):]
	jobID := rest[:strings.Index(rest, `"`)]

	// 5. Connect to GET /jobs/{id}/events (SSE stream) and verify event payload
	sseResp, err := http.Get(ts.URL + "/jobs/" + jobID + "/events")
	if err != nil {
		t.Fatalf("GET /jobs/%s/events failed: %v", jobID, err)
	}
	defer sseResp.Body.Close()

	if ct := sseResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content-type, got %q", ct)
	}

	scanner := bufio.NewScanner(sseResp.Body)
	var gotProgressEvent bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var ev job.ProgressEvent
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
				t.Fatalf("invalid SSE JSON data: %v", err)
			}
			if string(ev.JobID) == jobID {
				gotProgressEvent = true
			}
			if ev.Status == job.StatusCompleted {
				break
			}
		}
	}
	if !gotProgressEvent {
		t.Fatal("did not receive expected SSE progress event")
	}

	// 6. GET /datasets/{id}/records to inspect cleaned table, search engine UI, and export buttons
	recResp, err := http.Get(ts.URL + "/datasets/" + dsID + "/records")
	if err != nil {
		t.Fatalf("GET /datasets/%s/records failed: %v", dsID, err)
	}
	recHTML, _ := io.ReadAll(recResp.Body)
	_ = recResp.Body.Close()
	if !strings.Contains(string(recHTML), "alice") || !strings.Contains(string(recHTML), "bob") {
		t.Fatalf("expected cleaned records 'alice' and 'bob' in records HTML table, got: %s", string(recHTML))
	}
	if !strings.Contains(string(recHTML), "<refinery-search-box>") {
		t.Fatalf("expected <refinery-search-box> search engine in records view")
	}
	if !strings.Contains(string(recHTML), "/export?format=csv") {
		t.Fatalf("expected export toolbar links in records view")
	}

	// 7. Test MongoDB-style JSON search in GET /datasets/{id}/records?q=...
	mongoQuery := url.QueryEscape(`{"id": {"$gt": 1}, "name": {"$regex": "^b"}}`)
	filteredResp, err := http.Get(ts.URL + "/datasets/" + dsID + "/records?q=" + mongoQuery)
	if err != nil {
		t.Fatalf("GET filtered records failed: %v", err)
	}
	filteredHTML, _ := io.ReadAll(filteredResp.Body)
	_ = filteredResp.Body.Close()
	if !strings.Contains(string(filteredHTML), "bob") || strings.Contains(string(filteredHTML), `<td class="u-mono">alice</td>`) {
		t.Fatalf("expected Mongo JSON query to match only 'bob', got:\n%s", string(filteredHTML))
	}

	// 8. Verify multi-format export endpoints (CSV, TSV, JSON, JSONL)
	exportCases := []struct {
		format      string
		contentType string
		mustContain string
	}{
		{format: "csv", contentType: "text/csv", mustContain: "1,alice"},
		{format: "tsv", contentType: "text/tab-separated-values", mustContain: "1\talice"},
		{format: "json", contentType: "application/json", mustContain: `"name": "alice"`},
		{format: "jsonl", contentType: "application/x-ndjson", mustContain: `"name":"alice"`},
	}

	for _, tc := range exportCases {
		expResp, err := http.Get(ts.URL + "/datasets/" + dsID + "/export?format=" + tc.format)
		if err != nil {
			t.Fatalf("GET export format=%s failed: %v", tc.format, err)
		}
		expBytes, _ := io.ReadAll(expResp.Body)
		_ = expResp.Body.Close()

		if expResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for format=%s, got %d: %s", tc.format, expResp.StatusCode, string(expBytes))
		}
		if ct := expResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, tc.contentType) {
			t.Errorf("format=%s: expected Content-Type prefix %q, got %q", tc.format, tc.contentType, ct)
		}
		if cd := expResp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") {
			t.Errorf("format=%s: expected Content-Disposition attachment header, got %q", tc.format, cd)
		}
		if !strings.Contains(string(expBytes), tc.mustContain) {
			t.Errorf("format=%s: expected payload to contain %q, got:\n%s", tc.format, tc.mustContain, string(expBytes))
		}
	}
}
