INSERT INTO job_errors (
    job_id, row_number, column_name, raw_value, error_code, message, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?);
