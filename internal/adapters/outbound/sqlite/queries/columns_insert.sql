INSERT INTO dataset_columns (
    dataset_id, col_index, original_name, normalized_name,
    inferred_type, target_type, nullable, null_count,
    sample_distinct, sample_values_json, rules_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
