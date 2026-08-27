package workflow

import (
	"context"
	"textilepermit/internal/domain"
	"textilepermit/internal/exposure"
	"textilepermit/internal/store"
)

type SubmitPlanInput struct {
	IdempotencyKey   string                `json:"idempotencyKey"`
	ExpectedVersion  int64                 `json:"expectedVersion"`
	CabinetCode      string                `json:"cabinetCode"`
	LightingSlots    []domain.LightingSlot `json:"lightingSlots"`
	DailyOpenMinutes int                   `json:"dailyOpenMinutes"`
	DisplayStartDate string                `json:"displayStartDate"`
	DisplayEndDate   string                `json:"displayEndDate"`
	RestRotationDays int                   `json:"restRotationDays"`
	UVProtection     bool                  `json:"uvProtection"`
	Actor            string                `json:"actor"`
}
type PlanResult struct {
	Case        domain.ArtifactCase        `json:"case"`
	Revision    domain.DisplayPlanRevision `json:"revision"`
	Assessment  domain.ExposureAssessment  `json:"assessment"`
	Findings    []domain.RiskFinding       `json:"findings"`
	Differences []exposure.Difference      `json:"differences,omitempty"`
}

func (s *Service) SubmitPlan(ctx context.Context, id string, in SubmitPlanInput) (PlanResult, error) {
	view, err := s.store.Evidence(ctx, id)
	if err != nil {
		return PlanResult{}, err
	}
	if err = requireVersion(view.Case.Version, in.ExpectedVersion); err != nil {
		return PlanResult{}, err
	}
	if err = view.Case.EnsureMutable(); err != nil {
		return PlanResult{}, err
	}
	now := clock()
	prev, hasPrev := store.LatestRevision(view)
	p := domain.DisplayPlanRevision{RevisionID: newID("rev"), CaseID: id, RevisionNumber: len(view.Revisions) + 1, CabinetCode: in.CabinetCode, LightingSlots: in.LightingSlots, DailyOpenMinutes: in.DailyOpenMinutes, DisplayStartDate: in.DisplayStartDate, DisplayEndDate: in.DisplayEndDate, RestRotationDays: in.RestRotationDays, UVProtection: in.UVProtection, SubmittedBy: in.Actor, SubmittedAt: now}
	if hasPrev {
		p.SupersedesRevisionID = prev.RevisionID
	}
	if err = domain.ValidatePlan(p); err != nil {
		return PlanResult{}, err
	}
	calc, err := exposure.Calculate(view.Case, p, now)
	if err != nil {
		return PlanResult{}, err
	}
	diff := []exposure.Difference{}
	if hasPrev {
		diff = exposure.Compare(prev, p)
	}
	raw, err := s.command(ctx, in.IdempotencyKey, "plan.assessed", id, in.Actor, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, id)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if e = c.EnsureMutable(); e != nil {
			return nil, e
		}
		blocking := false
		for _, f := range calc.Findings {
			if f.Severity == "blocking" {
				blocking = true
			}
		}
		next := domain.StatusReview
		if blocking {
			next = domain.StatusRemediating
		}
		if e = c.Move(next); e != nil {
			return nil, e
		}
		c.MeasurementStale = false
		old := c.Version
		c.Version++
		c.UpdatedAt = now
		if e = tx.InsertRevision(ctx, p); e != nil {
			return nil, e
		}
		if e = tx.InsertAssessment(ctx, calc.Assessment); e != nil {
			return nil, e
		}
		if e = tx.InsertFindings(ctx, calc.Findings); e != nil {
			return nil, e
		}
		if e = tx.UpdateCase(ctx, c, old); e != nil {
			return nil, e
		}
		return PlanResult{Case: c, Revision: p, Assessment: calc.Assessment, Findings: calc.Findings, Differences: diff}, nil
	})
	if err != nil {
		return PlanResult{}, err
	}
	return decodeResult[PlanResult](raw)
}

type ResolveRiskInput struct {
	IdempotencyKey  string `json:"idempotencyKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	ResolutionNote  string `json:"resolutionNote"`
	Evidence        string `json:"evidence"`
	Actor           string `json:"actor"`
}

type BatchResolution struct {
	FindingID      string `json:"findingId"`
	ResolutionNote string `json:"resolutionNote"`
	Evidence       string `json:"evidence"`
}
type BatchResolveInput struct {
	IdempotencyKey  string            `json:"idempotencyKey"`
	ExpectedVersion int64             `json:"expectedVersion"`
	Items           []BatchResolution `json:"items"`
	Actor           string            `json:"actor"`
}

func (s *Service) BatchResolveRisk(ctx context.Context, caseID string, in BatchResolveInput) ([]domain.RiskFinding, error) {
	if len(in.Items) == 0 {
		return nil, &domain.RuleError{Field: "items", Message: "至少提交一项整改"}
	}
	seen := map[string]bool{}
	for i, it := range in.Items {
		if it.FindingID == "" || seen[it.FindingID] {
			return nil, &domain.RuleError{Field: "items", Message: "风险编号不能为空且不能重复"}
		}
		if it.ResolutionNote == "" || it.Evidence == "" {
			return nil, &domain.RuleError{Field: "items", Message: "每项整改措施和佐证均不能为空"}
		}
		seen[it.FindingID] = true
		_ = i
	}
	v, err := s.store.Evidence(ctx, caseID)
	if err != nil {
		return nil, err
	}
	latest, ok := store.LatestAssessment(v)
	if !ok {
		return nil, &domain.RuleError{Field: "assessment", Message: "尚未完成测算"}
	}
	raw, err := s.command(ctx, in.IdempotencyKey, "risks.batch-resolved", caseID, in.Actor, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, caseID)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if e = c.EnsureMutable(); e != nil {
			return nil, e
		}
		out := make([]domain.RiskFinding, 0, len(in.Items))
		for _, it := range in.Items {
			f, e := tx.Finding(ctx, it.FindingID)
			if e != nil {
				return nil, e
			}
			if f.CaseID != caseID || f.AssessmentID != latest.AssessmentID {
				return nil, &domain.RuleError{Field: "findingId", Message: "风险不属于当前最新测算"}
			}
			if f.Status != "open" {
				return nil, &domain.RuleError{Field: "findingId", Message: "风险已闭环"}
			}
			rf, e := tx.ResolveFinding(ctx, it.FindingID, it.ResolutionNote, it.Evidence, in.Actor)
			if e != nil {
				return nil, e
			}
			out = append(out, rf)
		}
		old := c.Version
		c.Version++
		c.UpdatedAt = clock()
		if e = tx.UpdateCase(ctx, c, old); e != nil {
			return nil, e
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return decodeResult[[]domain.RiskFinding](raw)
}

func (s *Service) ResolveRisk(ctx context.Context, caseID, findingID string, in ResolveRiskInput) (domain.RiskFinding, error) {
	if in.ResolutionNote == "" || in.Evidence == "" {
		return domain.RiskFinding{}, &domain.RuleError{Field: "resolutionNote", Message: "整改说明和佐证均不能为空"}
	}
	raw, err := s.command(ctx, in.IdempotencyKey, "risk.resolved", caseID, in.Actor, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, caseID)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if e = c.EnsureMutable(); e != nil {
			return nil, e
		}
		f, e := tx.ResolveFinding(ctx, findingID, in.ResolutionNote, in.Evidence, in.Actor)
		if e != nil {
			return nil, e
		}
		if f.CaseID != caseID {
			return nil, domain.ErrNotFound
		}
		old := c.Version
		c.Version++
		c.UpdatedAt = clock()
		if e = tx.UpdateCase(ctx, c, old); e != nil {
			return nil, e
		}
		return f, nil
	})
	if err != nil {
		return domain.RiskFinding{}, err
	}
	return decodeResult[domain.RiskFinding](raw)
}
