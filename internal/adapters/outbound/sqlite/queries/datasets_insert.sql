INSERT INTO datasets (
    id, original_name, storage_path, file_format, delimiter,
    has_header, size_bytes, total_rows, status, sample_rows_json,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
