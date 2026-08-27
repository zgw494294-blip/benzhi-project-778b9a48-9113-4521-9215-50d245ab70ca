package workflow

import (
	"context"
	"textilepermit/internal/domain"
	"textilepermit/internal/exposure"
	"textilepermit/internal/store"
)

type CreateCaseInput struct {
	IdempotencyKey     string  `json:"idempotencyKey"`
	AccessionCode      string  `json:"accessionCode"`
	Title              string  `json:"title"`
	MaterialProfile    string  `json:"materialProfile"`
	DyeSensitivity     string  `json:"dyeSensitivity"`
	FragileAreas       string  `json:"fragileAreas"`
	HistoricalLuxHours float64 `json:"historicalLuxHours"`
	AnnualLuxHourLimit float64 `json:"annualLuxHourLimit"`
	Actor              string  `json:"actor"`
}

func (s *Service) CreateCase(ctx context.Context, in CreateCaseInput) (domain.ArtifactCase, error) {
	now := clock()
	c := domain.ArtifactCase{CaseID: newID("case"), AccessionCode: in.AccessionCode, Title: in.Title, MaterialProfile: in.MaterialProfile, DyeSensitivity: in.DyeSensitivity, FragileAreas: in.FragileAreas, HistoricalLuxHours: in.HistoricalLuxHours, AnnualLuxHourLimit: in.AnnualLuxHourLimit, Status: domain.StatusDraft, Version: 1, CreatedBy: in.Actor, CreatedAt: now, UpdatedAt: now}
	if err := domain.ValidateCase(c); err != nil {
		return c, err
	}
	if in.Actor == "" {
		return c, &domain.RuleError{Field: "actor", Message: "操作人不能为空"}
	}
	raw, err := s.command(ctx, in.IdempotencyKey, "case.created", c.CaseID, in.Actor, func(tx *store.CommandTx) (any, error) { return c, tx.InsertCase(ctx, c) })
	if err != nil {
		return c, err
	}
	return decodeResult[domain.ArtifactCase](raw)
}

type UpdateCaseInput struct {
	IdempotencyKey     string  `json:"idempotencyKey"`
	ExpectedVersion    int64   `json:"expectedVersion"`
	Title              string  `json:"title"`
	MaterialProfile    string  `json:"materialProfile"`
	DyeSensitivity     string  `json:"dyeSensitivity"`
	FragileAreas       string  `json:"fragileAreas"`
	HistoricalLuxHours float64 `json:"historicalLuxHours"`
	AnnualLuxHourLimit float64 `json:"annualLuxHourLimit"`
	Actor              string  `json:"actor"`
}

type CaseImpactPreview struct {
	Case               domain.ArtifactCase        `json:"case"`
	Assessment         *domain.ExposureAssessment `json:"assessment,omitempty"`
	Findings           []domain.RiskFinding       `json:"findings,omitempty"`
	Differences        []exposure.Difference      `json:"differences,omitempty"`
	AffectsMeasurement bool                       `json:"affectsMeasurement"`
}

func (s *Service) PreviewCaseUpdate(ctx context.Context, id string, in UpdateCaseInput) (CaseImpactPreview, error) {
	v, err := s.store.Evidence(ctx, id)
	if err != nil {
		return CaseImpactPreview{}, err
	}
	if err = requireVersion(v.Case.Version, in.ExpectedVersion); err != nil {
		return CaseImpactPreview{}, err
	}
	if err = v.Case.EnsureMutable(); err != nil {
		return CaseImpactPreview{}, err
	}
	updated := v.Case
	updated.Title = in.Title
	updated.MaterialProfile = in.MaterialProfile
	updated.DyeSensitivity = in.DyeSensitivity
	updated.FragileAreas = in.FragileAreas
	updated.HistoricalLuxHours = in.HistoricalLuxHours
	updated.AnnualLuxHourLimit = in.AnnualLuxHourLimit
	if err = domain.ValidateCase(updated); err != nil {
		return CaseImpactPreview{}, err
	}
	res := CaseImpactPreview{Case: updated}
	res.AffectsMeasurement = updated.MaterialProfile != v.Case.MaterialProfile || updated.DyeSensitivity != v.Case.DyeSensitivity || updated.FragileAreas != v.Case.FragileAreas || updated.HistoricalLuxHours != v.Case.HistoricalLuxHours || updated.AnnualLuxHourLimit != v.Case.AnnualLuxHourLimit
	if updated.MaterialProfile != v.Case.MaterialProfile {
		res.Differences = append(res.Differences, exposure.Difference{Field: "materialProfile", Before: v.Case.MaterialProfile, After: updated.MaterialProfile})
	}
	if updated.DyeSensitivity != v.Case.DyeSensitivity {
		res.Differences = append(res.Differences, exposure.Difference{Field: "dyeSensitivity", Before: v.Case.DyeSensitivity, After: updated.DyeSensitivity})
	}
	if updated.FragileAreas != v.Case.FragileAreas {
		res.Differences = append(res.Differences, exposure.Difference{Field: "fragileAreas", Before: v.Case.FragileAreas, After: updated.FragileAreas})
	}
	if updated.HistoricalLuxHours != v.Case.HistoricalLuxHours {
		res.Differences = append(res.Differences, exposure.Difference{Field: "historicalLuxHours", Before: v.Case.HistoricalLuxHours, After: updated.HistoricalLuxHours})
	}
	if updated.AnnualLuxHourLimit != v.Case.AnnualLuxHourLimit {
		res.Differences = append(res.Differences, exposure.Difference{Field: "annualLuxHourLimit", Before: v.Case.AnnualLuxHourLimit, After: updated.AnnualLuxHourLimit})
	}
	if p, ok := store.LatestRevision(v); ok {
		if calc, e := exposure.Calculate(updated, p, clock()); e == nil {
			res.Assessment = &calc.Assessment
			res.Findings = calc.Findings
		}
	}
	return res, nil
}

func (s *Service) UpdateCase(ctx context.Context, id string, in UpdateCaseInput) (domain.ArtifactCase, error) {
	raw, err := s.command(ctx, in.IdempotencyKey, "case.updated", id, in.Actor, func(tx *store.CommandTx) (any, error) {
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
		impact := c.MaterialProfile != in.MaterialProfile || c.DyeSensitivity != in.DyeSensitivity || c.FragileAreas != in.FragileAreas || c.HistoricalLuxHours != in.HistoricalLuxHours || c.AnnualLuxHourLimit != in.AnnualLuxHourLimit
		c.Title = in.Title
		c.MaterialProfile = in.MaterialProfile
		c.DyeSensitivity = in.DyeSensitivity
		c.FragileAreas = in.FragileAreas
		c.HistoricalLuxHours = in.HistoricalLuxHours
		c.AnnualLuxHourLimit = in.AnnualLuxHourLimit
		if e = domain.ValidateCase(c); e != nil {
			return nil, e
		}
		if impact {
			c.MeasurementStale = true
			if c.Status == domain.StatusReview {
				c.Status = domain.StatusRemediating
			}
		}
		old := c.Version
		c.Version++
		c.UpdatedAt = clock()
		return c, tx.UpdateCase(ctx, c, old)
	})
	if err != nil {
		return domain.ArtifactCase{}, err
	}
	return decodeResult[domain.ArtifactCase](raw)
}
