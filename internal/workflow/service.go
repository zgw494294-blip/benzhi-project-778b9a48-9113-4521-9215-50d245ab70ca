package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"textilepermit/internal/domain"
	"textilepermit/internal/store"
)

type Service struct{ store *store.Store }

func New(s *store.Store) *Service { return &Service{store: s} }

func decodeResult[T any](raw json.RawMessage) (T, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}
func (s *Service) command(ctx context.Context, key, op, caseID, actor string, fn func(*store.CommandTx) (any, error)) (json.RawMessage, error) {
	raw, _, err := s.store.Command(ctx, key, op, caseID, actor, fn)
	return raw, err
}
func (s *Service) Evidence(ctx context.Context, id string) (domain.EvidenceView, error) {
	return s.store.Evidence(ctx, id)
}
func (s *Service) Cases(ctx context.Context) ([]domain.ArtifactCase, error) {
	return s.store.ListCases(ctx)
}

type Verification struct {
	Valid             bool             `json:"valid"`
	Status            string           `json:"status"`
	PermitID          string           `json:"permitId,omitempty"`
	CaseID            string           `json:"caseId,omitempty"`
	Conditions        []string         `json:"conditions,omitempty"`
	ValidFrom         string           `json:"validFrom,omitempty"`
	ValidUntil        string           `json:"validUntil,omitempty"`
	EvidenceDigest    string           `json:"evidenceDigest,omitempty"`
	ConditionChecks   []ConditionCheck `json:"conditionChecks,omitempty"`
	ConditionsOverall string           `json:"conditionsOverall,omitempty"`
}
type ConditionCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Service) Verify(ctx context.Context, code string) (Verification, error) {
	p, err := s.store.PermitByCode(ctx, code)
	if err != nil {
		return Verification{}, err
	}
	digest, err := s.store.ValidateFrozenEvidence(ctx, p.CaseID)
	if err != nil {
		return Verification{}, err
	}
	status := "valid"
	valid := digest == p.EvidenceDigest
	if !valid {
		status = "digest_mismatch"
	} else if clock().Before(p.ValidFrom) {
		valid = false
		status = "not_started"
	} else if clock().After(p.ValidUntil) {
		valid = false
		status = "expired"
	}
	return Verification{Valid: valid, Status: status, PermitID: p.PermitID, CaseID: p.CaseID, Conditions: p.Conditions, ValidFrom: p.ValidFrom.Format("2006-01-02"), ValidUntil: p.ValidUntil.Format("2006-01-02"), EvidenceDigest: p.EvidenceDigest}, nil
}

func (s *Service) VerifyWithConditions(ctx context.Context, code, checkDate, cabinet string, measuredPeak float64, rotationDays int) (Verification, error) {
	v, err := s.Verify(ctx, code)
	if err != nil {
		return v, err
	}
	checks := []ConditionCheck{}
	overall := v.Valid
	if checkDate != "" {
		pass := checkDate >= v.ValidFrom && checkDate <= v.ValidUntil
		checks = append(checks, ConditionCheck{"核验日期", map[bool]string{true: "pass", false: "fail"}[pass], map[bool]string{true: "日期在有效期内", false: "日期不在有效期内"}[pass]})
		overall = overall && pass
	}
	if cabinet != "" {
		pass := false
		for _, c := range v.Conditions {
			if strings.Contains(c, cabinet) {
				pass = true
			}
		}
		checks = append(checks, ConditionCheck{"现场展柜", map[bool]string{true: "pass", false: "fail"}[pass], map[bool]string{true: "展柜符合批准条件", false: "展柜不符合批准条件"}[pass]})
		overall = overall && pass
	}
	var approvedPeak float64
	var approvedRotation int
	if fr, e := s.store.Frozen(ctx, v.CaseID); e == nil {
		var m struct {
			Assessment domain.ExposureAssessment  `json:"assessment"`
			Revision   domain.DisplayPlanRevision `json:"revision"`
		}
		_ = json.Unmarshal(fr.Manifest, &m)
		approvedPeak = m.Assessment.PeakLux
		approvedRotation = m.Revision.RestRotationDays
	}
	if measuredPeak > 0 {
		pass := approvedPeak <= 0 || measuredPeak <= approvedPeak
		checks = append(checks, ConditionCheck{"实测峰值照度", map[bool]string{true: "pass", false: "fail"}[pass], map[bool]string{true: "实测峰值不超过批准值", false: "实测峰值超过批准值"}[pass]})
		overall = overall && pass
	}
	if rotationDays >= 0 {
		pass := rotationDays >= approvedRotation
		checks = append(checks, ConditionCheck{"轮换执行天数", map[bool]string{true: "pass", false: "fail"}[pass], map[bool]string{true: "轮换执行满足批准周期", false: "轮换执行少于批准周期"}[pass]})
		overall = overall && pass
	}
	v.ConditionChecks = checks
	if len(checks) > 0 {
		v.ConditionsOverall = map[bool]string{true: "符合", false: "不符合"}[overall]
		if v.Valid && !overall {
			v.Status = "conditions_mismatch"
		}
	}
	return v, nil
}

func requireVersion(got, want int64) error {
	if got != want {
		return fmt.Errorf("%w：当前版本 %d，请求版本 %d", domain.ErrConflict, got, want)
	}
	return nil
}
