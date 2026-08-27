package domain

import "testing"

func TestPlanSlotMinutesMustMatch(t *testing.T) {
	p := DisplayPlanRevision{CabinetCode: "C", LightingSlots: []LightingSlot{{Name: "上午", Lux: 50, Minutes: 60}}, DailyOpenMinutes: 120, DisplayStartDate: "2026-01-01", DisplayEndDate: "2026-01-02"}
	if ValidatePlan(p) == nil {
		t.Fatal("应拒绝不一致时长")
	}
}
func TestFrozenCaseCannotChange(t *testing.T) {
	c := ArtifactCase{Status: StatusFrozen}
	if c.EnsureMutable() == nil {
		t.Fatal("冻结案卷应不可修改")
	}
}
func TestReviewerSeparation(t *testing.T) {
	if CheckReviewer("u1", "u1") != ErrSeparation {
		t.Fatal("应执行职责分离")
	}
}
