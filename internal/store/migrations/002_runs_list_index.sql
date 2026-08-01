CREATE INDEX IF NOT EXISTS runs_updated_id_idx
ON runs (updated_at DESC, id ASC);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES (2, unixepoch());
