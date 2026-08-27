package workflow

import (
	"context"
	"path/filepath"
	"testing"
	"textilepermit/internal/domain"
	"textilepermit/internal/store"
	"time"
)

func TestCompletePermitFlow(t *testing.T) {
	repo, e := store.Open(filepath.Join(t.TempDir(), "flow.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer repo.Close()
	s := New(repo)
	ctx := context.Background()
	c, e := s.CreateCase(ctx, CreateCaseInput{IdempotencyKey: "1", AccessionCode: "T1", Title: "丝织品", MaterialProfile: "丝", DyeSensitivity: "medium", FragileAreas: "无可见脆弱部位", HistoricalLuxHours: 10, AnnualLuxHourLimit: 10000, Actor: "applicant"})
	if e != nil {
		t.Fatal(e)
	}
	today := time.Now().UTC()
	p, e := s.SubmitPlan(ctx, c.CaseID, SubmitPlanInput{IdempotencyKey: "2", ExpectedVersion: c.Version, CabinetCode: "C1", LightingSlots: []domain.LightingSlot{{Name: "开放", Lux: 40, Minutes: 60}}, DailyOpenMinutes: 60, DisplayStartDate: today.Format(time.DateOnly), DisplayEndDate: today.AddDate(0, 0, 2).Format(time.DateOnly), RestRotationDays: 1, UVProtection: true, Actor: "designer"})
	if e != nil {
		t.Fatal(e)
	}
	submitted, e := s.SubmitReview(ctx, c.CaseID, SubmitReviewInput{IdempotencyKey: "3", ExpectedVersion: p.Case.Version, Actor: "applicant"})
	if e != nil {
		t.Fatal(e)
	}
	review, e := s.Review(ctx, c.CaseID, ReviewInput{IdempotencyKey: "4", ExpectedVersion: submitted.Version, ReviewerID: "reviewer", Outcome: "approved", Reason: "符合要求", ValidDays: 3})
	if e != nil {
		t.Fatal(e)
	}
	if review.Permit == nil || review.Case.Status != domain.StatusPermitted {
		t.Fatalf("未签发 %+v", review)
	}
	verified, e := s.Verify(ctx, review.Permit.VerificationCode)
	if e != nil || !verified.Valid {
		t.Fatalf("验真失败 %+v %v", verified, e)
	}
	_, e = s.UpdateCase(ctx, c.CaseID, UpdateCaseInput{IdempotencyKey: "5", ExpectedVersion: review.Case.Version, Title: "改名", Actor: "x"})
	if e != domain.ErrForbidden {
		t.Fatalf("冻结后应禁止修改 %v", e)
	}
}

func TestSameApplicantCannotApprove(t *testing.T) {
	repo, _ := store.Open(filepath.Join(t.TempDir(), "sep.db"))
	defer repo.Close()
	s := New(repo)
	ctx := context.Background()
	c, _ := s.CreateCase(ctx, CreateCaseInput{IdempotencyKey: "1", AccessionCode: "A", Title: "T", MaterialProfile: "丝", DyeSensitivity: "low", FragileAreas: "无", AnnualLuxHourLimit: 10000, Actor: "same"})
	now := time.Now()
	p, _ := s.SubmitPlan(ctx, c.CaseID, SubmitPlanInput{IdempotencyKey: "2", ExpectedVersion: c.Version, CabinetCode: "C", LightingSlots: []domain.LightingSlot{{Name: "L", Lux: 10, Minutes: 60}}, DailyOpenMinutes: 60, DisplayStartDate: now.Format(time.DateOnly), DisplayEndDate: now.Format(time.DateOnly), RestRotationDays: 0, UVProtection: true, Actor: "d"})
	submitted, _ := s.SubmitReview(ctx, c.CaseID, SubmitReviewInput{IdempotencyKey: "3", ExpectedVersion: p.Case.Version, Actor: "same"})
	_, e := s.Review(ctx, c.CaseID, ReviewInput{IdempotencyKey: "4", ExpectedVersion: submitted.Version, ReviewerID: "same", Outcome: "approved"})
	if e != domain.ErrSeparation {
		t.Fatalf("未执行职责分离 %v", e)
	}
}
