package workflow

import (
	"context"
	"textilepermit/internal/domain"
	"textilepermit/internal/exposure"
	"textilepermit/internal/store"
)

// PreviewPlan performs the same domain validation and exposure rules as a
// submission without allocating identifiers or writing evidence. The result
// is explicitly marked as a draft by the HTTP representation.
func (s *Service) PreviewPlan(ctx context.Context, caseID string, in SubmitPlanInput) (PlanResult, error) {
	view, err := s.store.Evidence(ctx, caseID)
	if err != nil {
		return PlanResult{}, err
	}
	if err = requireVersion(view.Case.Version, in.ExpectedVersion); err != nil {
		return PlanResult{}, err
	}
	if err = view.Case.EnsureMutable(); err != nil {
		return PlanResult{}, err
	}
	previous, hasPrevious := store.LatestRevision(view)
	plan := domain.DisplayPlanRevision{
		RevisionID:       "draft-preview",
		CaseID:           caseID,
		RevisionNumber:   len(view.Revisions) + 1,
		CabinetCode:      in.CabinetCode,
		LightingSlots:    in.LightingSlots,
		DailyOpenMinutes: in.DailyOpenMinutes,
		DisplayStartDate: in.DisplayStartDate,
		DisplayEndDate:   in.DisplayEndDate,
		RestRotationDays: in.RestRotationDays,
		UVProtection:     in.UVProtection,
		SubmittedBy:      in.Actor,
	}
	if hasPrevious {
		plan.SupersedesRevisionID = previous.RevisionID
	}
	if err = domain.ValidatePlan(plan); err != nil {
		return PlanResult{}, err
	}
	result, err := exposure.Calculate(view.Case, plan, clock())
	if err != nil {
		return PlanResult{}, err
	}
	differences := []exposure.Difference{}
	if hasPrevious {
		differences = exposure.Compare(previous, plan)
	}
	return PlanResult{Case: view.Case, Revision: plan, Assessment: result.Assessment, Findings: result.Findings, Differences: differences}, nil
}

type RevisionDifference struct {
	FromRevision int                   `json:"fromRevision"`
	ToRevision   int                   `json:"toRevision"`
	Changes      []exposure.Difference `json:"changes"`
}

func (s *Service) RevisionDifferences(ctx context.Context, caseID string) ([]RevisionDifference, error) {
	view, err := s.store.Evidence(ctx, caseID)
	if err != nil {
		return nil, err
	}
	result := make([]RevisionDifference, 0, len(view.Revisions))
	for i := 1; i < len(view.Revisions); i++ {
		before, after := view.Revisions[i-1], view.Revisions[i]
		result = append(result, RevisionDifference{FromRevision: before.RevisionNumber, ToRevision: after.RevisionNumber, Changes: exposure.Compare(before, after)})
	}
	return result, nil
}
