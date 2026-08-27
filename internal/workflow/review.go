package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"textilepermit/internal/domain"
	"textilepermit/internal/store"
	"time"
)

type SubmitReviewInput struct {
	IdempotencyKey  string `json:"idempotencyKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Actor           string `json:"actor"`
	ReviewerID      string `json:"reviewerId,omitempty"`
}

func (s *Service) SubmitReview(ctx context.Context, id string, in SubmitReviewInput) (domain.ArtifactCase, error) {
	view, err := s.store.Evidence(ctx, id)
	if err != nil {
		return domain.ArtifactCase{}, err
	}
	if view.Case.MeasurementStale {
		return domain.ArtifactCase{}, &domain.RuleError{Field: "assessment", Message: "档案变更后必须重新测算"}
	}
	if in.ReviewerID != "" {
		ready, e := s.CheckReadiness(ctx, id, in.ReviewerID)
		if e != nil {
			return domain.ArtifactCase{}, e
		}
		if !ready.Ready {
			for _, it := range ready.Items {
				if !it.Passed {
					return domain.ArtifactCase{}, &domain.RuleError{Field: it.Code, Message: it.Message}
				}
			}
		}
	}
	latest, ok := store.LatestAssessment(view)
	if !ok {
		return domain.ArtifactCase{}, &domain.RuleError{Field: "assessment", Message: "尚未完成测算"}
	}
	for _, f := range view.Findings {
		if f.AssessmentID == latest.AssessmentID && f.Severity == "blocking" && f.Status != "resolved" {
			return domain.ArtifactCase{}, &domain.RuleError{Field: "findings", Message: "仍有未闭环的阻断风险"}
		}
		if f.AssessmentID == latest.AssessmentID && f.Status == "resolved" && (f.RuleCode == "PEAK_LUX" || f.RuleCode == "ANNUAL_DOSE" || f.RuleCode == "ROTATION") {
			if rev, ok := store.LatestRevision(view); !ok || rev.RevisionID == latest.RevisionID {
				return domain.ArtifactCase{}, &domain.RuleError{Field: "revision", Message: "参数类阻断风险必须创建新方案修订并重新测算"}
			}
		}
	}
	if d, ok := latestReturned(view); ok {
		responded := false
		for _, r := range view.Responses {
			if r.DecisionID == d.DecisionID {
				responded = true
			}
		}
		latestRev, hasRev := store.LatestRevision(view)
		latestAssessment, hasAssessment := store.LatestAssessment(view)
		if !responded || !hasRev || !hasAssessment || latestRev.SubmittedAt.Before(d.DecidedAt) || latestAssessment.RevisionID != latestRev.RevisionID {
			return domain.ArtifactCase{}, &domain.RuleError{Field: "reviewResponse", Message: "退回意见必须回应并完成新的方案修订与测算"}
		}
	}
	raw, err := s.command(ctx, in.IdempotencyKey, "review.submitted", id, in.Actor, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, id)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if c.Status != domain.StatusRemediating && c.Status != domain.StatusDraft && c.Status != domain.StatusReview {
			return nil, domain.ErrForbidden
		}
		if c.Status != domain.StatusReview {
			if e = c.Move(domain.StatusReview); e != nil {
				return nil, e
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

func latestReturned(v domain.EvidenceView) (domain.ReviewDecision, bool) {
	for i := len(v.Decisions) - 1; i >= 0; i-- {
		if v.Decisions[i].Outcome == "returned" {
			return v.Decisions[i], true
		}
	}
	return domain.ReviewDecision{}, false
}

type ReviewResponseInput struct {
	IdempotencyKey  string `json:"idempotencyKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	DecisionID      string `json:"decisionId"`
	Explanation     string `json:"explanation"`
	Actor           string `json:"actor"`
}

func (s *Service) RespondReview(ctx context.Context, id string, in ReviewResponseInput) (domain.ReviewResponse, error) {
	if strings.TrimSpace(in.Explanation) == "" {
		return domain.ReviewResponse{}, &domain.RuleError{Field: "explanation", Message: "回应说明不能为空"}
	}
	v, err := s.store.Evidence(ctx, id)
	if err != nil {
		return domain.ReviewResponse{}, err
	}
	if err = requireVersion(v.Case.Version, in.ExpectedVersion); err != nil {
		return domain.ReviewResponse{}, err
	}
	if err = v.Case.EnsureMutable(); err != nil {
		return domain.ReviewResponse{}, err
	}
	d, ok := latestReturned(v)
	if !ok || d.DecisionID != in.DecisionID {
		return domain.ReviewResponse{}, &domain.RuleError{Field: "decisionId", Message: "只能回应当前最新退回意见"}
	}
	for _, r := range v.Responses {
		if r.DecisionID == d.DecisionID {
			return domain.ReviewResponse{}, &domain.RuleError{Field: "decisionId", Message: "该退回意见已回应"}
		}
	}
	rev, ok := store.LatestRevision(v)
	if !ok || rev.SubmittedAt.Before(d.DecidedAt) {
		return domain.ReviewResponse{}, &domain.RuleError{Field: "revisionId", Message: "请先创建退回后的新方案修订"}
	}
	r := domain.ReviewResponse{ResponseID: newID("response"), CaseID: id, DecisionID: d.DecisionID, RevisionID: rev.RevisionID, Explanation: in.Explanation, Actor: in.Actor, CreatedAt: clock()}
	raw, err := s.command(ctx, in.IdempotencyKey, "review.responded", id, in.Actor, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, id)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if e = tx.InsertResponse(ctx, r); e != nil {
			return nil, e
		}
		old := c.Version
		c.Version++
		c.UpdatedAt = clock()
		if e = tx.UpdateCase(ctx, c, old); e != nil {
			return nil, e
		}
		return r, nil
	})
	if err != nil {
		return domain.ReviewResponse{}, err
	}
	return decodeResult[domain.ReviewResponse](raw)
}

type ReviewInput struct {
	IdempotencyKey  string `json:"idempotencyKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	ReviewerID      string `json:"reviewerId"`
	Outcome         string `json:"outcome"`
	Reason          string `json:"reason"`
	ValidDays       int    `json:"validDays"`
}
type ReviewResult struct {
	Case           domain.ArtifactCase   `json:"case"`
	Decision       domain.ReviewDecision `json:"decision"`
	Permit         *domain.DisplayPermit `json:"permit,omitempty"`
	EvidenceDigest string                `json:"evidenceDigest,omitempty"`
}

func (s *Service) Review(ctx context.Context, id string, in ReviewInput) (ReviewResult, error) {
	view, err := s.store.Evidence(ctx, id)
	if err != nil {
		return ReviewResult{}, err
	}
	if err = requireVersion(view.Case.Version, in.ExpectedVersion); err != nil {
		return ReviewResult{}, err
	}
	if view.Case.Status != domain.StatusReview {
		return ReviewResult{}, domain.ErrForbidden
	}
	if err = domain.CheckReviewer(view.Case.CreatedBy, in.ReviewerID); err != nil {
		return ReviewResult{}, err
	}
	if in.Outcome != "approved" && in.Outcome != "returned" {
		return ReviewResult{}, &domain.RuleError{Field: "outcome", Message: "必须为 approved 或 returned"}
	}
	if in.Outcome == "returned" && strings.TrimSpace(in.Reason) == "" {
		return ReviewResult{}, &domain.RuleError{Field: "reason", Message: "退回必须填写理由"}
	}
	rev, ok := store.LatestRevision(view)
	if !ok {
		return ReviewResult{}, domain.ErrForbidden
	}
	now := clock()
	decision := domain.ReviewDecision{DecisionID: newID("decision"), CaseID: id, RevisionID: rev.RevisionID, ReviewerID: in.ReviewerID, Outcome: in.Outcome, Reason: in.Reason, DecidedAt: now, CaseVersion: in.ExpectedVersion}
	manifest := struct {
		Case       domain.ArtifactCase        `json:"case"`
		Revision   domain.DisplayPlanRevision `json:"revision"`
		Assessment domain.ExposureAssessment  `json:"assessment"`
		Findings   []domain.RiskFinding       `json:"findings"`
		Decision   domain.ReviewDecision      `json:"decision"`
	}{Case: view.Case, Revision: rev, Findings: store.LatestFindings(view), Decision: decision}
	manifest.Assessment, _ = store.LatestAssessment(view)
	digest, _ := domain.Digest(manifest)
	raw, err := s.command(ctx, in.IdempotencyKey, "review.decided", id, in.ReviewerID, func(tx *store.CommandTx) (any, error) {
		c, e := tx.Case(ctx, id)
		if e != nil {
			return nil, e
		}
		if e = requireVersion(c.Version, in.ExpectedVersion); e != nil {
			return nil, e
		}
		if c.Status != domain.StatusReview {
			return nil, domain.ErrForbidden
		}
		if e = tx.InsertDecision(ctx, decision); e != nil {
			return nil, e
		}
		old := c.Version
		if in.Outcome == "returned" {
			if e = c.Move(domain.StatusRemediating); e != nil {
				return nil, e
			}
			c.Version++
			c.UpdatedAt = now
			if e = tx.UpdateCase(ctx, c, old); e != nil {
				return nil, e
			}
			return ReviewResult{Case: c, Decision: decision}, nil
		}
		if e = c.Move(domain.StatusApproved); e != nil {
			return nil, e
		}
		if e = c.Move(domain.StatusFrozen); e != nil {
			return nil, e
		}
		if e = tx.InsertFreeze(ctx, id, digest, manifest); e != nil {
			return nil, e
		}
		days := in.ValidDays
		if days <= 0 || days > 365 {
			days = 90
		}
		validFrom, _ := time.Parse(time.DateOnly, rev.DisplayStartDate)
		validUntil := validFrom.AddDate(0, 0, days-1)
		planEnd, _ := time.Parse(time.DateOnly, rev.DisplayEndDate)
		if validUntil.After(planEnd) {
			validUntil = planEnd
		}
		code := verificationCode(id, digest)
		permit := domain.DisplayPermit{PermitID: newID("permit"), CaseID: id, FrozenRevisionID: rev.RevisionID, EvidenceDigest: digest, Conditions: []string{"仅限展柜 " + rev.CabinetCode, "峰值照度不得超过已批准方案", "按批准轮换周期执行"}, ValidFrom: validFrom, ValidUntil: validUntil, IssuedBy: in.ReviewerID, IssuedAt: now, VerificationCode: code}
		if e = tx.InsertPermit(ctx, permit); e != nil {
			return nil, e
		}
		if e = c.Move(domain.StatusPermitted); e != nil {
			return nil, e
		}
		c.Version++
		c.UpdatedAt = now
		if e = tx.UpdateCase(ctx, c, old); e != nil {
			return nil, e
		}
		return ReviewResult{Case: c, Decision: decision, Permit: &permit, EvidenceDigest: digest}, nil
	})
	if err != nil {
		return ReviewResult{}, err
	}
	return decodeResult[ReviewResult](raw)
}

func verificationCode(caseID, digest string) string {
	h := sha256.Sum256([]byte(caseID + ":" + digest))
	return strings.ToUpper(hex.EncodeToString(h[:8]))
}
