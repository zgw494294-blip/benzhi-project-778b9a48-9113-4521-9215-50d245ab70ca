package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"textilepermit/internal/domain"
	"time"
)

func (t *CommandTx) InsertCase(ctx context.Context, c domain.ArtifactCase) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO cases(case_id,accession_code,title,material_profile,dye_sensitivity,fragile_areas,historical_lux_hours,annual_limit,status,version,created_by,created_at,updated_at,measurement_stale) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.CaseID, c.AccessionCode, c.Title, c.MaterialProfile, c.DyeSensitivity, c.FragileAreas, c.HistoricalLuxHours, c.AnnualLuxHourLimit, c.Status, c.Version, c.CreatedBy, c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano), boolInt(c.MeasurementStale))
	return err
}

func (t *CommandTx) UpdateCase(ctx context.Context, c domain.ArtifactCase, expected int64) error {
	r, err := t.tx.ExecContext(ctx, `UPDATE cases SET title=?,material_profile=?,dye_sensitivity=?,fragile_areas=?,historical_lux_hours=?,annual_limit=?,status=?,version=?,updated_at=?,measurement_stale=? WHERE case_id=? AND version=?`, c.Title, c.MaterialProfile, c.DyeSensitivity, c.FragileAreas, c.HistoricalLuxHours, c.AnnualLuxHourLimit, c.Status, c.Version, c.UpdatedAt.Format(time.RFC3339Nano), boolInt(c.MeasurementStale), c.CaseID, expected)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (t *CommandTx) Case(ctx context.Context, id string) (domain.ArtifactCase, error) {
	return scanCase(t.tx.QueryRowContext(ctx, `SELECT case_id,accession_code,title,material_profile,dye_sensitivity,fragile_areas,historical_lux_hours,annual_limit,status,version,created_by,created_at,updated_at,measurement_stale FROM cases WHERE case_id=?`, id))
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (t *CommandTx) InsertRevision(ctx context.Context, p domain.DisplayPlanRevision) error {
	b, _ := json.Marshal(p)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO revisions(revision_id,case_id,revision_number,payload,submitted_at) VALUES(?,?,?,?,?)", p.RevisionID, p.CaseID, p.RevisionNumber, string(b), p.SubmittedAt.Format(time.RFC3339Nano))
	return err
}
func (t *CommandTx) InsertAssessment(ctx context.Context, a domain.ExposureAssessment) error {
	b, _ := json.Marshal(a)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO assessments(assessment_id,case_id,revision_id,payload,calculated_at) VALUES(?,?,?,?,?)", a.AssessmentID, a.CaseID, a.RevisionID, string(b), a.CalculatedAt.Format(time.RFC3339Nano))
	return err
}
func (t *CommandTx) InsertFindings(ctx context.Context, fs []domain.RiskFinding) error {
	for _, f := range fs {
		b, _ := json.Marshal(f)
		if _, err := t.tx.ExecContext(ctx, "INSERT INTO findings(finding_id,case_id,assessment_id,payload,status) VALUES(?,?,?,?,?)", f.FindingID, f.CaseID, f.AssessmentID, string(b), f.Status); err != nil {
			return err
		}
	}
	return nil
}
func (t *CommandTx) ResolveFinding(ctx context.Context, id, note, evidence, actor string) (domain.RiskFinding, error) {
	var raw string
	err := t.tx.QueryRowContext(ctx, "SELECT payload FROM findings WHERE finding_id=?", id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RiskFinding{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RiskFinding{}, err
	}
	var f domain.RiskFinding
	if err = json.Unmarshal([]byte(raw), &f); err != nil {
		return f, err
	}
	now := t.now
	f.Status = "resolved"
	f.ResolutionNote = note
	f.Evidence = evidence
	f.ResolvedBy = actor
	f.ResolvedAt = &now
	b, _ := json.Marshal(f)
	_, err = t.tx.ExecContext(ctx, "UPDATE findings SET payload=?,status='resolved' WHERE finding_id=?", string(b), id)
	return f, err
}
func (t *CommandTx) Finding(ctx context.Context, id string) (domain.RiskFinding, error) {
	var raw string
	if err := t.tx.QueryRowContext(ctx, "SELECT payload FROM findings WHERE finding_id=?", id).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RiskFinding{}, domain.ErrNotFound
		}
		return domain.RiskFinding{}, err
	}
	var f domain.RiskFinding
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return f, err
	}
	return f, nil
}
func (t *CommandTx) InsertResponse(ctx context.Context, r domain.ReviewResponse) error {
	b, _ := json.Marshal(r)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO review_responses(response_id,case_id,decision_id,revision_id,payload,created_at) VALUES(?,?,?,?,?,?)", r.ResponseID, r.CaseID, r.DecisionID, r.RevisionID, string(b), r.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (t *CommandTx) InsertDecision(ctx context.Context, d domain.ReviewDecision) error {
	b, _ := json.Marshal(d)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO decisions(decision_id,case_id,revision_id,payload,decided_at) VALUES(?,?,?,?,?)", d.DecisionID, d.CaseID, d.RevisionID, string(b), d.DecidedAt.Format(time.RFC3339Nano))
	return err
}
func (t *CommandTx) InsertFreeze(ctx context.Context, caseID, digest string, manifest any) error {
	b, _ := json.Marshal(manifest)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO freezes(case_id,digest,manifest,frozen_at) VALUES(?,?,?,?)", caseID, digest, string(b), t.now.Format(time.RFC3339Nano))
	return err
}
func (t *CommandTx) InsertPermit(ctx context.Context, p domain.DisplayPermit) error {
	b, _ := json.Marshal(p)
	_, err := t.tx.ExecContext(ctx, "INSERT INTO permits(permit_id,case_id,verification_code,payload,issued_at) VALUES(?,?,?,?,?)", p.PermitID, p.CaseID, p.VerificationCode, string(b), p.IssuedAt.Format(time.RFC3339Nano))
	return err
}
