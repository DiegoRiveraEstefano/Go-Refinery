UPDATE jobs
SET status = ?,
    total_rows = ?,
    processed_rows = ?,
    valid_rows = ?,
    error_rows = ?,
    error_message = ?,
    started_at = ?,
    finished_at = ?
WHERE id = ?;
