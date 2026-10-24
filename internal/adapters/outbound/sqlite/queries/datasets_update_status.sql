UPDATE datasets
SET status = ?,
    total_rows = CASE WHEN ? > 0 THEN ? ELSE total_rows END,
    updated_at = ?
WHERE id = ?;
