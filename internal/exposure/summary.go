package exposure

import "textilepermit/internal/domain"

type Summary struct {
	ProjectedLuxHours float64 `json:"projectedLuxHours"`
	PeakLux           float64 `json:"peakLux"`
	Remaining         float64 `json:"remaining"`
	Blocking          int     `json:"blocking"`
	Warnings          int     `json:"warnings"`
}

func Summarize(a domain.ExposureAssessment, f []domain.RiskFinding) Summary {
	s := Summary{ProjectedLuxHours: a.ProjectedLuxHours, PeakLux: a.PeakLux, Remaining: a.AnnualRemainingLuxHours}
	for _, item := range f {
		if item.Severity == "blocking" && item.Status != "resolved" {
			s.Blocking++
		} else if item.Severity == "warning" {
			s.Warnings++
		}
	}
	return s
}
