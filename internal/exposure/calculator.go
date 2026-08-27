package exposure

import (
	"fmt"
	"math"
	"sort"
	"textilepermit/internal/domain"
	"time"
)

const RuleSetVersion = "textile-light-v1"

type Result struct {
	Assessment domain.ExposureAssessment
	Findings   []domain.RiskFinding
}

func Calculate(c domain.ArtifactCase, p domain.DisplayPlanRevision, now time.Time) (Result, error) {
	if err := domain.ValidateCase(c); err != nil {
		return Result{}, err
	}
	if err := domain.ValidatePlan(p); err != nil {
		return Result{}, err
	}
	start, _ := time.Parse(time.DateOnly, p.DisplayStartDate)
	end, _ := time.Parse(time.DateOnly, p.DisplayEndDate)
	calendarDays := int(end.Sub(start).Hours()/24) + 1
	cycle := p.RestRotationDays + 1
	displayDays := 0
	dailyDose, peak := 0.0, 0.0
	items := make([]domain.CalculationItem, 0, len(p.LightingSlots)+4)
	for _, slot := range p.LightingSlots {
		dose := slot.Lux * float64(slot.Minutes) / 60
		dailyDose += dose
		if slot.Lux > peak {
			peak = slot.Lux
		}
		items = append(items, domain.CalculationItem{Label: slot.Name + "每日剂量", Formula: fmt.Sprintf("%.2f lux × %d/60 h", slot.Lux, slot.Minutes), Value: round(dose), Unit: "lux·h"})
	}
	for i := 0; i < calendarDays; i++ {
		if i%cycle == 0 {
			displayDays++
		}
	}
	projected := dailyDose * float64(displayDays)
	annual := annualBreakdown(start, end, cycle, dailyDose, c)
	if len(annual) > 0 {
		totalAnnual := 0.0
		for _, y := range annual {
			totalAnnual += y.ProjectedLuxHours
		}
		delta := round(projected) - round(totalAnnual)
		if delta != 0 {
			annual[len(annual)-1].ProjectedLuxHours = round(annual[len(annual)-1].ProjectedLuxHours + delta)
			annual[len(annual)-1].RemainingLuxHours = round(c.AnnualLuxHourLimit - annual[len(annual)-1].HistoricalLuxHours - annual[len(annual)-1].ProjectedLuxHours)
			annual[len(annual)-1].SafetyMarginPercent = round(annual[len(annual)-1].RemainingLuxHours / c.AnnualLuxHourLimit * 100)
		}
	}
	remaining, margin := c.AnnualLuxHourLimit-c.HistoricalLuxHours-projected, c.AnnualLuxHourLimit
	if len(annual) > 0 {
		remaining = annual[0].RemainingLuxHours
		margin = annual[0].SafetyMarginPercent
	}
	for _, y := range annual {
		if y.RemainingLuxHours < remaining {
			remaining = y.RemainingLuxHours
		}
		if y.SafetyMarginPercent < margin {
			margin = y.SafetyMarginPercent
		}
	}
	items = append(items,
		domain.CalculationItem{Label: "日历展期", Formula: "结束日期-开始日期+1", Value: float64(calendarDays), Unit: "天"},
		domain.CalculationItem{Label: "实际照明日", Formula: fmt.Sprintf("按 %d 天周期轮换", cycle), Value: float64(displayDays), Unit: "天"},
		domain.CalculationItem{Label: "方案累计剂量", Formula: "每日剂量 × 实际照明日", Value: round(projected), Unit: "lux·h"},
		domain.CalculationItem{Label: "年度余量", Formula: "年度上限 - 历史剂量 - 方案剂量", Value: round(remaining), Unit: "lux·h"},
	)
	for _, y := range annual {
		items = append(items, domain.CalculationItem{Label: fmt.Sprintf("%d年度剂量", y.Year), Formula: "年度实际照明日 × 每日剂量", Value: round(y.ProjectedLuxHours), Unit: "lux·h"})
	}
	a := domain.ExposureAssessment{AssessmentID: stableID("assessment", p.RevisionID, RuleSetVersion), CaseID: c.CaseID, RevisionID: p.RevisionID, RuleSetVersion: RuleSetVersion, ProjectedLuxHours: round(projected), PeakLux: round(peak), AnnualRemainingLuxHours: round(remaining), SafetyMarginPercent: round(margin), CalculationBreakdown: items, CalculatedAt: now.UTC(), AnnualBreakdown: annual}
	return Result{Assessment: a, Findings: Evaluate(c, p, a)}, nil
}

func annualBreakdown(start, end time.Time, cycle int, daily float64, c domain.ArtifactCase) []domain.AnnualExposure {
	type bucket struct {
		days int
		dose float64
	}
	m := map[int]*bucket{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		idx := int(d.Sub(start).Hours() / 24)
		if idx%cycle != 0 {
			continue
		}
		y := d.Year()
		if m[y] == nil {
			m[y] = &bucket{}
		}
		m[y].days++
		m[y].dose += daily
	}
	years := make([]int, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Ints(years)
	out := make([]domain.AnnualExposure, 0, len(years))
	for i, y := range years {
		hist := 0.0
		if i == 0 {
			hist = c.HistoricalLuxHours
		}
		rem := c.AnnualLuxHourLimit - hist - m[y].dose
		out = append(out, domain.AnnualExposure{Year: y, LightingDays: m[y].days, ProjectedLuxHours: round(m[y].dose), HistoricalLuxHours: round(hist), RemainingLuxHours: round(rem), SafetyMarginPercent: round(rem / c.AnnualLuxHourLimit * 100)})
	}
	return out
}

func round(v float64) float64 { return math.Round(v*100) / 100 }
