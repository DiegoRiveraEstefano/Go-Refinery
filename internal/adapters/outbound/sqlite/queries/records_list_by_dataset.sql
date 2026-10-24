SELECT dataset_id, row_number, values_json, created_at
FROM dataset_records
WHERE dataset_id = ?
ORDER BY row_number ASC
LIMIT ? OFFSET ?;
