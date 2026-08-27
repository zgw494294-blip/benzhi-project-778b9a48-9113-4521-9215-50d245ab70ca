package workflow

import (
	"context"
	"strings"
	"textilepermit/internal/domain"
	"textilepermit/internal/store"
)

type ReadinessItem struct {
	Code    string `json:"code"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}
type Readiness struct {
	Ready bool            `json:"ready"`
	Items []ReadinessItem `json:"items"`
}

func (s *Service) CheckReadiness(ctx context.Context, id, reviewer string) (Readiness, error) {
	v, e := s.store.Evidence(ctx, id)
	if e != nil {
		return Readiness{}, e
	}
	items := []ReadinessItem{}
	rev, hasRev := store.LatestRevision(v)
	ass, hasAss := store.LatestAssessment(v)
	items = append(items, ReadinessItem{"PLAN_EXISTS", hasRev, "已存在展陈方案"})
	items = append(items, ReadinessItem{"ASSESSMENT_CURRENT", hasAss && hasRev && ass.RevisionID == rev.RevisionID && !v.Case.MeasurementStale, "最新方案已有有效测算"})
	blocking := 0
	parameterStale := false
	for _, f := range store.LatestFindings(v) {
		if f.Severity == "blocking" && f.Status != "resolved" {
			blocking++
		}
		if f.Status == "resolved" && (f.RuleCode == "PEAK_LUX" || f.RuleCode == "ANNUAL_DOSE" || f.RuleCode == "ROTATION") && hasRev && f.AssessmentID == ass.AssessmentID && rev.RevisionID == ass.RevisionID {
			parameterStale = true
		}
	}
	items = append(items, ReadinessItem{"BLOCKING_RESOLVED", blocking == 0 && !parameterStale, "所有阻断风险已闭环且参数风险已重新测算"})
	returnedOK := true
	if d, ok := latestReturned(v); ok {
		returnedOK = false
		for _, r := range v.Responses {
			if r.DecisionID == d.DecisionID {
				returnedOK = true
			}
		}
		if returnedOK {
			returnedOK = hasRev && rev.SubmittedAt.After(d.DecidedAt) && hasAss && ass.RevisionID == rev.RevisionID
		}
	}
	items = append(items, ReadinessItem{"RETURN_RESPONSE", returnedOK, "退回意见已回应并完成新修订"})
	sep := domain.CheckReviewer(v.Case.CreatedBy, reviewer) == nil
	if strings.TrimSpace(reviewer) == "" {
		sep = false
	}
	items = append(items, ReadinessItem{"REVIEWER_SEPARATION", sep, "复核人与申请人职责分离"})
	ready := true
	for _, i := range items {
		ready = ready && i.Passed
	}
	return Readiness{Ready: ready, Items: items}, nil
}

type CaseSearchQuery struct {
	Keyword, Status, Dye string
	HasBlocking          *bool
	Page, PageSize       int
}
type CaseSummary struct {
	Case              domain.ArtifactCase `json:"case"`
	CaseID            string              `json:"caseId"`
	AccessionCode     string              `json:"accessionCode"`
	Title             string              `json:"title"`
	Status            domain.CaseStatus   `json:"status"`
	DyeSensitivity    string              `json:"dyeSensitivity"`
	Version           int64               `json:"version"`
	LatestRevision    int                 `json:"latestRevision"`
	ProjectedLuxHours float64             `json:"projectedLuxHours"`
	OpenBlocking      int                 `json:"openBlocking"`
	PermitValid       bool                `json:"permitValid"`
	UpdatedAt         string              `json:"updatedAt"`
}
type CaseSearchResult struct {
	Cases        []CaseSummary  `json:"cases"`
	Total        int            `json:"total"`
	Page         int            `json:"page"`
	PageSize     int            `json:"pageSize"`
	StatusCounts map[string]int `json:"statusCounts"`
}

func (s *Service) SearchCases(ctx context.Context, q CaseSearchQuery) (CaseSearchResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		return CaseSearchResult{}, &domain.RuleError{Field: "pageSize", Message: "页大小不能超过100"}
	}
	cs, e := s.store.ListCases(ctx)
	if e != nil {
		return CaseSearchResult{}, e
	}
	out := []CaseSummary{}
	counts := map[string]int{}
	for _, c := range cs {
		if q.Keyword != "" && !strings.Contains(c.AccessionCode, q.Keyword) && !strings.Contains(c.Title, q.Keyword) {
			continue
		}
		if q.Status != "" && string(c.Status) != q.Status {
			continue
		}
		if q.Dye != "" && c.DyeSensitivity != q.Dye {
			continue
		}
		v, e := s.store.Evidence(ctx, c.CaseID)
		if e != nil {
			return CaseSearchResult{}, e
		}
		n := 0
		for _, f := range store.LatestFindings(v) {
			if f.Severity == "blocking" && f.Status != "resolved" {
				n++
			}
		}
		if q.HasBlocking != nil && *q.HasBlocking != (n > 0) {
			continue
		}
		counts[string(c.Status)]++
		sum := CaseSummary{Case: c, CaseID: c.CaseID, AccessionCode: c.AccessionCode, Title: c.Title, Status: c.Status, DyeSensitivity: c.DyeSensitivity, Version: c.Version, OpenBlocking: n, UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
		if r, ok := store.LatestRevision(v); ok {
			sum.LatestRevision = r.RevisionNumber
		}
		if a, ok := store.LatestAssessment(v); ok {
			sum.ProjectedLuxHours = a.ProjectedLuxHours
		}
		if v.Permit != nil {
			sum.PermitValid = clock().Before(v.Permit.ValidUntil) && clock().After(v.Permit.ValidFrom)
		}
		out = append(out, sum)
	}
	total := len(out)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return CaseSearchResult{Cases: out[start:end], Total: total, Page: q.Page, PageSize: q.PageSize, StatusCounts: counts}, nil
}
