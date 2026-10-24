package sqlite

import (
	"embed"
	"fmt"
	"strings"
)

// queriesFS embeds all raw parameterized SQL query files under queries/*.sql.
//
//go:embed queries/*.sql
var queriesFS embed.FS

var (
	sqlPragmas = mustLoadQuery("queries/pragmas.sql")

	// Dataset & Column queries
	sqlInsertDataset         = mustLoadQuery("queries/datasets_insert.sql")
	sqlGetDatasetByID        = mustLoadQuery("queries/datasets_get_by_id.sql")
	sqlListDatasets          = mustLoadQuery("queries/datasets_list.sql")
	sqlUpdateDatasetStatus   = mustLoadQuery("queries/datasets_update_status.sql")
	sqlDatasetExists         = mustLoadQuery("queries/datasets_exists.sql")
	sqlTouchDatasetUpdatedAt = mustLoadQuery("queries/datasets_touch_updated_at.sql")
	sqlDeleteDatasetColumns  = mustLoadQuery("queries/columns_delete_by_dataset.sql")
	sqlInsertDatasetColumn   = mustLoadQuery("queries/columns_insert.sql")
	sqlListDatasetColumns    = mustLoadQuery("queries/columns_list_by_dataset.sql")

	// Job & JobError queries
	sqlInsertJob         = mustLoadQuery("queries/jobs_insert.sql")
	sqlGetJobByID        = mustLoadQuery("queries/jobs_get_by_id.sql")
	sqlGetLatestJob      = mustLoadQuery("queries/jobs_get_latest_by_dataset.sql")
	sqlUpdateJobProgress = mustLoadQuery("queries/jobs_update_progress.sql")
	sqlInsertJobError    = mustLoadQuery("queries/job_errors_insert.sql")
	sqlListJobErrors     = mustLoadQuery("queries/job_errors_list_by_job.sql")

	// Record queries
	sqlUpsertRecord = mustLoadQuery("queries/records_upsert.sql")
	sqlListRecords  = mustLoadQuery("queries/records_list_by_dataset.sql")
)

func mustLoadQuery(path string) string {
	b, err := queriesFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("sqlite: failed to load embedded query %q: %v", path, err))
	}
	return strings.TrimSpace(string(b))
}
