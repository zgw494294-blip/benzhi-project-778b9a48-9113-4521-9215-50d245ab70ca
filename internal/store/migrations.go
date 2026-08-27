package store

import "context"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS cases(case_id TEXT PRIMARY KEY, accession_code TEXT NOT NULL UNIQUE, title TEXT NOT NULL, material_profile TEXT NOT NULL, dye_sensitivity TEXT NOT NULL, fragile_areas TEXT NOT NULL, historical_lux_hours REAL NOT NULL, annual_limit REAL NOT NULL, status TEXT NOT NULL, version INTEGER NOT NULL, created_by TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS revisions(revision_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(case_id), revision_number INTEGER NOT NULL, payload TEXT NOT NULL, submitted_at TEXT NOT NULL, UNIQUE(case_id, revision_number));
CREATE TABLE IF NOT EXISTS assessments(assessment_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(case_id), revision_id TEXT NOT NULL REFERENCES revisions(revision_id), payload TEXT NOT NULL, calculated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS findings(finding_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(case_id), assessment_id TEXT NOT NULL REFERENCES assessments(assessment_id), payload TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS decisions(decision_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(case_id), revision_id TEXT NOT NULL REFERENCES revisions(revision_id), payload TEXT NOT NULL, decided_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS freezes(case_id TEXT PRIMARY KEY REFERENCES cases(case_id), digest TEXT NOT NULL, manifest TEXT NOT NULL, frozen_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS permits(permit_id TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE REFERENCES cases(case_id), verification_code TEXT NOT NULL UNIQUE, payload TEXT NOT NULL, issued_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS idempotency(idempotency_key TEXT PRIMARY KEY, operation TEXT NOT NULL, case_id TEXT, response TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS audit_events(sequence INTEGER PRIMARY KEY AUTOINCREMENT, case_id TEXT NOT NULL, event_type TEXT NOT NULL, actor TEXT NOT NULL, detail TEXT NOT NULL, occurred_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS idx_revision_case ON revisions(case_id, revision_number);
CREATE INDEX IF NOT EXISTS idx_assessment_case ON assessments(case_id);
CREATE INDEX IF NOT EXISTS idx_finding_case ON findings(case_id);
CREATE INDEX IF NOT EXISTS idx_audit_case ON audit_events(case_id, sequence);`,
	`ALTER TABLE cases ADD COLUMN measurement_stale INTEGER NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS review_responses(response_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(case_id), decision_id TEXT NOT NULL REFERENCES decisions(decision_id), revision_id TEXT NOT NULL REFERENCES revisions(revision_id), payload TEXT NOT NULL, created_at TEXT NOT NULL);`,
}

func (s *Store) Migrate(ctx context.Context) error {
	for i, script := range migrations {
		version := i + 1
		var found int
		err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations WHERE version=?", version).Scan(&found)
		if err != nil && version == 1 {
			found = 0
		}
		if found > 0 {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, script); err != nil {
			tx.Rollback()
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(?,datetime('now'))", version); err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
