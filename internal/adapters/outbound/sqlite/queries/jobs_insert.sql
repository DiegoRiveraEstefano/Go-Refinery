INSERT INTO jobs (
    id, dataset_id, status, total_rows, processed_rows,
    valid_rows, error_rows, max_errors, error_message,
    started_at, finished_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
