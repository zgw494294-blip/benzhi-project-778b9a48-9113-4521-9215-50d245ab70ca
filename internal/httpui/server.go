package httpui

import (
	"embed"
	"net/http"
	"textilepermit/internal/workflow"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	service   *workflow.Service
	mux       *http.ServeMux
	selfcheck bool
}

func New(service *workflow.Service, selfcheck bool) *Server {
	s := &Server{service: service, mux: http.NewServeMux(), selfcheck: selfcheck}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return securityHeaders(requestLog(s.mux)) }
func (s *Server) routes() {
	s.mux.HandleFunc("GET /workspace", s.Workspace)
	s.mux.HandleFunc("GET /", s.Root)
	s.mux.HandleFunc("GET /assets/app.css", s.AssetCSS)
	s.mux.HandleFunc("GET /assets/app.js", s.AssetJS)
	s.mux.HandleFunc("GET /api/ready", s.Ready)
	s.mux.HandleFunc("GET /api/cases", s.ListCases)
	s.mux.HandleFunc("POST /api/cases", s.CreateCase)
	s.mux.HandleFunc("GET /api/cases/{caseID}", s.GetCase)
	s.mux.HandleFunc("PATCH /api/cases/{caseID}", s.UpdateCase)
	s.mux.HandleFunc("POST /api/cases/{caseID}/profile/preview", s.PreviewCaseUpdate)
	s.mux.HandleFunc("POST /api/cases/{caseID}/impact-preview", s.PreviewCaseUpdate)
	s.mux.HandleFunc("POST /api/cases/{caseID}/archive/preview", s.PreviewCaseUpdate)
	s.mux.HandleFunc("POST /api/cases/{caseID}/plans", s.SubmitPlan)
	s.mux.HandleFunc("POST /api/cases/{caseID}/plans/preview", s.PreviewPlan)
	s.mux.HandleFunc("GET /api/cases/{caseID}/differences", s.GetDifferences)
	s.mux.HandleFunc("POST /api/cases/{caseID}/risks/{findingID}/resolve", s.ResolveRisk)
	s.mux.HandleFunc("POST /api/cases/{caseID}/risks/batch-resolve", s.BatchResolveRisk)
	s.mux.HandleFunc("POST /api/cases/{caseID}/risks/batch", s.BatchResolveRisk)
	s.mux.HandleFunc("POST /api/cases/{caseID}/submit-review", s.SubmitReview)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review", s.Review)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review-response", s.RespondReview)
	s.mux.HandleFunc("POST /api/cases/{caseID}/respond-review", s.RespondReview)
	s.mux.HandleFunc("GET /api/cases/{caseID}/readiness", s.Readiness)
	s.mux.HandleFunc("GET /api/cases/{caseID}/review/readiness", s.Readiness)
	s.mux.HandleFunc("GET /api/cases/{caseID}/audit", s.GetAudit)
	s.mux.HandleFunc("GET /api/permits/verify", s.VerifyPermit)
}
