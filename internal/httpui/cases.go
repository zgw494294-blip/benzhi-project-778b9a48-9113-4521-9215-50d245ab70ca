package httpui

import (
	"context"
	"net/http"
	"strconv"
	"textilepermit/internal/domain"
	"textilepermit/internal/workflow"
)

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ready": true, "selfcheck": s.selfcheck})
}
func (s *Server) ListCases(w http.ResponseWriter, r *http.Request) {
	q := workflow.CaseSearchQuery{Keyword: r.URL.Query().Get("keyword"), Status: r.URL.Query().Get("status"), Dye: r.URL.Query().Get("dyeSensitivity"), Page: 1, PageSize: 20}
	if x := r.URL.Query().Get("page"); x != "" {
		n, e := strconv.Atoi(x)
		if e != nil || n < 1 {
			writeError(w, &domain.RuleError{Field: "page", Message: "页码必须为正整数"})
			return
		}
		q.Page = n
	}
	if x := r.URL.Query().Get("pageSize"); x != "" {
		n, e := strconv.Atoi(x)
		if e != nil || n < 1 {
			writeError(w, &domain.RuleError{Field: "pageSize", Message: "页大小必须为正整数"})
			return
		}
		q.PageSize = n
	}
	blockParam := r.URL.Query().Get("hasBlocking")
	if blockParam == "" {
		blockParam = r.URL.Query().Get("hasOpenBlocking")
	}
	if x := blockParam; x != "" {
		b := x == "true"
		q.HasBlocking = &b
	}
	if q.Status != "" && q.Status != "draft" && q.Status != "remediating" && q.Status != "review" && q.Status != "approved" && q.Status != "frozen" && q.Status != "permitted" {
		writeError(w, &domain.RuleError{Field: "status", Message: "状态筛选值无效"})
		return
	}
	if q.Dye != "" && q.Dye != "high" && q.Dye != "medium" && q.Dye != "low" {
		writeError(w, &domain.RuleError{Field: "dyeSensitivity", Message: "染料敏感等级无效"})
		return
	}
	v, e := s.service.SearchCases(r.Context(), q)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) PreviewCaseUpdate(w http.ResponseWriter, r *http.Request) {
	var in workflow.UpdateCaseInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.PreviewCaseUpdate(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) Readiness(w http.ResponseWriter, r *http.Request) {
	v, e := s.service.CheckReadiness(r.Context(), r.PathValue("caseID"), r.URL.Query().Get("reviewerId"))
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) CreateCase(w http.ResponseWriter, r *http.Request) {
	var in workflow.CreateCaseInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.CreateCase(context.WithoutCancel(r.Context()), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (s *Server) GetCase(w http.ResponseWriter, r *http.Request) {
	v, e := s.service.Evidence(r.Context(), r.PathValue("caseID"))
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) UpdateCase(w http.ResponseWriter, r *http.Request) {
	var in workflow.UpdateCaseInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.UpdateCase(context.WithoutCancel(r.Context()), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) GetAudit(w http.ResponseWriter, r *http.Request) {
	v, e := s.service.Evidence(r.Context(), r.PathValue("caseID"))
	if e != nil {
		writeError(w, e)
		return
	}
	events := v.Audit
	if typ := r.URL.Query().Get("eventType"); typ != "" {
		filtered := make([]domain.AuditEvent, 0)
		for _, a := range events {
			if a.EventType == typ {
				filtered = append(filtered, a)
			}
		}
		events = filtered
	}
	if seq := r.URL.Query().Get("fromSequence"); seq != "" {
		if n, e := strconv.ParseInt(seq, 10, 64); e == nil {
			filtered := make([]domain.AuditEvent, 0)
			for _, a := range events {
				if a.Sequence >= n {
					filtered = append(filtered, a)
				}
			}
			events = filtered
		}
	}
	continuous := true
	for i := 1; i < len(events); i++ {
		if events[i].Sequence <= events[i-1].Sequence {
			continuous = false
		}
	}
	writeJSON(w, 200, map[string]any{"caseId": v.Case.CaseID, "audit": events, "continuous": continuous, "frozenDigest": v.FrozenDigest})
}
