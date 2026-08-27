package store

import (
	"context"
	"encoding/json"
	"fmt"
	"textilepermit/internal/domain"
)

type FrozenManifest struct {
	Case       domain.ArtifactCase        `json:"case"`
	Revision   domain.DisplayPlanRevision `json:"revision"`
	Assessment domain.ExposureAssessment  `json:"assessment"`
	Findings   []domain.RiskFinding       `json:"findings"`
	Decision   domain.ReviewDecision      `json:"decision"`
}

// ValidateFrozenEvidence checks the canonical digest and internal references
// represented by a freeze before public verification.
func (s *Store) ValidateFrozenEvidence(ctx context.Context, caseID string) (string, error) {
	record, err := s.Frozen(ctx, caseID)
	if err != nil {
		return "", err
	}
	var manifest FrozenManifest
	if err = json.Unmarshal(record.Manifest, &manifest); err != nil {
		return "", fmt.Errorf("冻结清单无法解析: %w", err)
	}
	calculated, err := domain.Digest(manifest)
	if err != nil || calculated != record.Digest {
		return "", fmt.Errorf("冻结清单摘要不一致")
	}
	if manifest.Case.CaseID != caseID || manifest.Revision.CaseID != caseID || manifest.Assessment.CaseID != caseID || manifest.Decision.CaseID != caseID {
		return "", fmt.Errorf("冻结清单包含其他案卷的记录")
	}
	if manifest.Assessment.RevisionID != manifest.Revision.RevisionID || manifest.Decision.RevisionID != manifest.Revision.RevisionID {
		return "", fmt.Errorf("冻结清单的方案引用不一致")
	}
	return record.Digest, nil
}

func hasRevision(all []domain.DisplayPlanRevision, id string) bool {
	for _, item := range all {
		if item.RevisionID == id {
			return true
		}
	}
	return false
}

func hasAssessment(all []domain.ExposureAssessment, id string) bool {
	for _, item := range all {
		if item.AssessmentID == id {
			return true
		}
	}
	return false
}

func hasDecision(all []domain.ReviewDecision, id string) bool {
	for _, item := range all {
		if item.DecisionID == id {
			return true
		}
	}
	return false
}

func hasFinding(all []domain.RiskFinding, id string) bool {
	for _, item := range all {
		if item.FindingID == id {
			return true
		}
	}
	return false
}
