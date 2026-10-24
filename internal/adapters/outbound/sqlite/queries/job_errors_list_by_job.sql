SELECT
    id, job_id, row_number, column_name,
    raw_value, error_code, message, created_at
FROM job_errors
WHERE job_id = ?
ORDER BY row_number ASC, id ASC
LIMIT ? OFFSET ?;
