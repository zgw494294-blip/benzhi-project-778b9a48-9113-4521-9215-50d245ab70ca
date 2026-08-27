package httpui

import (
	"net/http"
	"strconv"
	"textilepermit/internal/workflow"
)

func (s *Server) SubmitPlan(w http.ResponseWriter, r *http.Request) {
	var in workflow.SubmitPlanInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.SubmitPlan(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (s *Server) PreviewPlan(w http.ResponseWriter, r *http.Request) {
	var in workflow.SubmitPlanInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.PreviewPlan(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": true, "result": v})
}
func (s *Server) GetDifferences(w http.ResponseWriter, r *http.Request) {
	v, e := s.service.RevisionDifferences(r.Context(), r.PathValue("caseID"))
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"differences": v})
}
func (s *Server) ResolveRisk(w http.ResponseWriter, r *http.Request) {
	var in workflow.ResolveRiskInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.ResolveRisk(r.Context(), r.PathValue("caseID"), r.PathValue("findingID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) BatchResolveRisk(w http.ResponseWriter, r *http.Request) {
	var in workflow.BatchResolveInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.BatchResolveRisk(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"findings": v})
}
func (s *Server) RespondReview(w http.ResponseWriter, r *http.Request) {
	var in workflow.ReviewResponseInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.RespondReview(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) SubmitReview(w http.ResponseWriter, r *http.Request) {
	var in workflow.SubmitReviewInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.SubmitReview(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) Review(w http.ResponseWriter, r *http.Request) {
	var in workflow.ReviewInput
	if e := readJSON(w, r, &in); e != nil {
		badJSON(w, e)
		return
	}
	v, e := s.service.Review(r.Context(), r.PathValue("caseID"), in)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) VerifyPermit(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		badJSON(w, &queryError{})
		return
	}
	peak, _ := strconv.ParseFloat(r.URL.Query().Get("measuredPeakLux"), 64)
	rot := -1
	if x := r.URL.Query().Get("rotationDays"); x != "" {
		rot, _ = strconv.Atoi(x)
	}
	v, e := s.service.VerifyWithConditions(r.Context(), code, r.URL.Query().Get("checkDate"), r.URL.Query().Get("cabinetCode"), peak, rot)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, v)
}

type queryError struct{}

func (*queryError) Error() string { return "code 参数不能为空" }
