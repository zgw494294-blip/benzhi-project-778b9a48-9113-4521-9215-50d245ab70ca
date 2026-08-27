package store

import (
	"context"
	"textilepermit/internal/domain"
)

func (s *Store) Evidence(ctx context.Context, id string) (domain.EvidenceView, error) {
	c, e := s.Case(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	r, e := s.Revisions(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	a, e := s.Assessments(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	f, e := s.Findings(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	d, e := s.Decisions(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	p, e := s.PermitByCase(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	au, e := s.Audit(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	digest, e := s.FrozenDigest(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	resp, e := s.Responses(context.WithoutCancel(ctx), id)
	if e != nil {
		return domain.EvidenceView{}, e
	}
	return domain.EvidenceView{Case: c, Revisions: r, Assessments: a, Findings: f, Decisions: d, Permit: p, Audit: au, FrozenDigest: digest, Responses: resp}, nil
}

func LatestRevision(v domain.EvidenceView) (domain.DisplayPlanRevision, bool) {
	if len(v.Revisions) == 0 {
		return domain.DisplayPlanRevision{}, false
	}
	return v.Revisions[len(v.Revisions)-1], true
}
func LatestAssessment(v domain.EvidenceView) (domain.ExposureAssessment, bool) {
	if len(v.Assessments) == 0 {
		return domain.ExposureAssessment{}, false
	}
	return v.Assessments[len(v.Assessments)-1], true
}
func LatestFindings(v domain.EvidenceView) []domain.RiskFinding {
	a, ok := LatestAssessment(v)
	if !ok {
		return nil
	}
	var out []domain.RiskFinding
	for _, f := range v.Findings {
		if f.AssessmentID == a.AssessmentID {
			out = append(out, f)
		}
	}
	return out
}
