package exposure

import (
	"testing"
	"textilepermit/internal/domain"
	"time"
)

func TestCalculateRotationAndFindings(t *testing.T) {
	c := domain.ArtifactCase{CaseID: "c1", AccessionCode: "A", Title: "丝巾", MaterialProfile: "丝", DyeSensitivity: "high", FragileAreas: "边缘", HistoricalLuxHours: 100, AnnualLuxHourLimit: 10000}
	p := domain.DisplayPlanRevision{RevisionID: "r1", CaseID: "c1", CabinetCode: "柜1", LightingSlots: []domain.LightingSlot{{Name: "上午", Lux: 60, Minutes: 120}, {Name: "下午", Lux: 30, Minutes: 120}}, DailyOpenMinutes: 240, DisplayStartDate: "2026-01-01", DisplayEndDate: "2026-01-10", RestRotationDays: 1, UVProtection: true}
	r, e := Calculate(c, p, time.Unix(1, 0))
	if e != nil {
		t.Fatal(e)
	}
	if r.Assessment.ProjectedLuxHours != 900 {
		t.Fatalf("剂量=%v", r.Assessment.ProjectedLuxHours)
	}
	if len(r.Findings) != 1 || r.Findings[0].RuleCode != "PEAK_LUX" {
		t.Fatalf("风险=%+v", r.Findings)
	}
}

func TestDeterministicIdentifiers(t *testing.T) {
	c := domain.ArtifactCase{CaseID: "c1", AccessionCode: "A", Title: "丝巾", MaterialProfile: "丝", DyeSensitivity: "medium", FragileAreas: "无", AnnualLuxHourLimit: 10000}
	p := domain.DisplayPlanRevision{RevisionID: "r1", CabinetCode: "柜", LightingSlots: []domain.LightingSlot{{Name: "日", Lux: 40, Minutes: 60}}, DailyOpenMinutes: 60, DisplayStartDate: "2026-01-01", DisplayEndDate: "2026-01-01", UVProtection: false}
	a, _ := Calculate(c, p, time.Now())
	b, _ := Calculate(c, p, time.Now().Add(time.Hour))
	if a.Assessment.AssessmentID != b.Assessment.AssessmentID || a.Findings[0].FindingID != b.Findings[0].FindingID {
		t.Fatal("相同输入未生成稳定编号")
	}
}
