SELECT
    id, original_name, storage_path, file_format, delimiter,
    has_header, size_bytes, total_rows, status, created_at, updated_at
FROM datasets
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
