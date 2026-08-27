package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"textilepermit/internal/domain"
	"time"
)

type rowScanner interface{ Scan(...any) error }

func scanCase(r rowScanner) (domain.ArtifactCase, error) {
	var c domain.ArtifactCase
	var status, created, updated string
	var stale int
	err := r.Scan(&c.CaseID, &c.AccessionCode, &c.Title, &c.MaterialProfile, &c.DyeSensitivity, &c.FragileAreas, &c.HistoricalLuxHours, &c.AnnualLuxHourLimit, &status, &c.Version, &c.CreatedBy, &created, &updated, &stale)
	if errors.Is(err, sql.ErrNoRows) {
		return c, domain.ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.Status = domain.CaseStatus(status)
	c.MeasurementStale = stale != 0
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return c, nil
}
func (s *Store) Case(ctx context.Context, id string) (domain.ArtifactCase, error) {
	return scanCase(s.db.QueryRowContext(ctx, `SELECT case_id,accession_code,title,material_profile,dye_sensitivity,fragile_areas,historical_lux_hours,annual_limit,status,version,created_by,created_at,updated_at,measurement_stale FROM cases WHERE case_id=?`, id))
}

func (s *Store) ListCases(ctx context.Context) ([]domain.ArtifactCase, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT case_id,accession_code,title,material_profile,dye_sensitivity,fragile_areas,historical_lux_hours,annual_limit,status,version,created_by,created_at,updated_at,measurement_stale FROM cases ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ArtifactCase
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanJSONRows[T any](rows *sql.Rows) ([]T, error) {
	defer rows.Close()
	var out []T
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Revisions(ctx context.Context, id string) ([]domain.DisplayPlanRevision, error) {
	r, e := s.db.QueryContext(ctx, "SELECT payload FROM revisions WHERE case_id=? ORDER BY revision_number", id)
	if e != nil {
		return nil, e
	}
	return scanJSONRows[domain.DisplayPlanRevision](r)
}
func (s *Store) Assessments(ctx context.Context, id string) ([]domain.ExposureAssessment, error) {
	r, e := s.db.QueryContext(ctx, "SELECT payload FROM assessments WHERE case_id=? ORDER BY calculated_at", id)
	if e != nil {
		return nil, e
	}
	return scanJSONRows[domain.ExposureAssessment](r)
}
func (s *Store) Findings(ctx context.Context, id string) ([]domain.RiskFinding, error) {
	r, e := s.db.QueryContext(ctx, "SELECT payload FROM findings WHERE case_id=? ORDER BY rowid", id)
	if e != nil {
		return nil, e
	}
	return scanJSONRows[domain.RiskFinding](r)
}
func (s *Store) Decisions(ctx context.Context, id string) ([]domain.ReviewDecision, error) {
	r, e := s.db.QueryContext(ctx, "SELECT payload FROM decisions WHERE case_id=? ORDER BY decided_at", id)
	if e != nil {
		return nil, e
	}
	return scanJSONRows[domain.ReviewDecision](r)
}
func (s *Store) Responses(ctx context.Context, id string) ([]domain.ReviewResponse, error) {
	r, e := s.db.QueryContext(ctx, "SELECT payload FROM review_responses WHERE case_id=? ORDER BY created_at", id)
	if e != nil {
		return nil, e
	}
	return scanJSONRows[domain.ReviewResponse](r)
}
func (s *Store) Audit(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT sequence,case_id,event_type,actor,detail,occurred_at FROM audit_events WHERE case_id=? ORDER BY sequence", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []domain.AuditEvent
	for rows.Next() {
		var a domain.AuditEvent
		var at string
		if e = rows.Scan(&a.Sequence, &a.CaseID, &a.EventType, &a.Actor, &a.Detail, &at); e != nil {
			return nil, e
		}
		a.OccurredAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) PermitByCode(ctx context.Context, code string) (domain.DisplayPermit, error) {
	var raw string
	e := s.db.QueryRowContext(ctx, "SELECT payload FROM permits WHERE verification_code=?", code).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return domain.DisplayPermit{}, domain.ErrNotFound
	}
	var p domain.DisplayPermit
	if e == nil {
		e = json.Unmarshal([]byte(raw), &p)
	}
	return p, e
}
func (s *Store) PermitByCase(ctx context.Context, id string) (*domain.DisplayPermit, error) {
	var raw string
	e := s.db.QueryRowContext(ctx, "SELECT payload FROM permits WHERE case_id=?", id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var p domain.DisplayPermit
	e = json.Unmarshal([]byte(raw), &p)
	return &p, e
}
func (s *Store) FrozenDigest(ctx context.Context, id string) (string, error) {
	var d string
	e := s.db.QueryRowContext(ctx, "SELECT digest FROM freezes WHERE case_id=?", id).Scan(&d)
	if errors.Is(e, sql.ErrNoRows) {
		return "", nil
	}
	return d, e
}

type FrozenRecord struct {
	Digest   string
	Manifest json.RawMessage
	FrozenAt time.Time
}

func (s *Store) Frozen(ctx context.Context, id string) (FrozenRecord, error) {
	var result FrozenRecord
	var raw, at string
	err := s.db.QueryRowContext(ctx, "SELECT digest,manifest,frozen_at FROM freezes WHERE case_id=?", id).Scan(&result.Digest, &raw, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return result, domain.ErrNotFound
	}
	if err != nil {
		return result, err
	}
	result.Manifest = json.RawMessage(raw)
	result.FrozenAt, err = time.Parse(time.RFC3339Nano, at)
	return result, err
}
