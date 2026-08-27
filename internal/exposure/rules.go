package exposure

import (
	"fmt"
	"textilepermit/internal/domain"
)

type findingSpec struct{ code, severity, summary, evidence string }

func Evaluate(c domain.ArtifactCase, p domain.DisplayPlanRevision, a domain.ExposureAssessment) []domain.RiskFinding {
	var specs []findingSpec
	threshold := map[string]float64{"high": 50, "medium": 100, "low": 200}[c.DyeSensitivity]
	if a.PeakLux > threshold {
		specs = append(specs, findingSpec{"PEAK_LUX", "blocking", "峰值照度超过该敏感等级建议阈值", "提供降低照度后的灯具测量记录"})
	}
	if a.AnnualRemainingLuxHours < 0 {
		summary := "预计年度累计光照剂量超过上限"
		for _, y := range a.AnnualBreakdown {
			if y.RemainingLuxHours < 0 {
				summary = fmt.Sprintf("%d年度累计光照剂量超过上限", y.Year)
				break
			}
		}
		specs = append(specs, findingSpec{"ANNUAL_DOSE", "blocking", summary, "调整展期、照度或轮换，并提供新方案"})
	}
	if !p.UVProtection {
		specs = append(specs, findingSpec{"UV_PROTECTION", "blocking", "未登记紫外防护措施", "提供滤紫外膜或灯具紫外控制证明"})
	}
	if c.DyeSensitivity == "high" && p.RestRotationDays < 1 {
		specs = append(specs, findingSpec{"ROTATION", "blocking", "高敏染料未安排轮换休息", "设置至少一天轮换休息并说明替换安排"})
	}
	if a.SafetyMarginPercent >= 0 && a.SafetyMarginPercent < 10 {
		specs = append(specs, findingSpec{"LOW_MARGIN", "warning", "年度安全裕度低于 10%", "复核负责人确认监测频率"})
	}
	if c.FragileAreas == "" {
		specs = append(specs, findingSpec{"FRAGILE_EVIDENCE", "blocking", "未登记脆弱部位检查结果", "补充脆弱部位或明确记录无可见脆弱部位"})
	}
	out := make([]domain.RiskFinding, 0, len(specs))
	for _, s := range specs {
		out = append(out, domain.RiskFinding{FindingID: stableID("finding", a.AssessmentID, s.code), CaseID: c.CaseID, AssessmentID: a.AssessmentID, RuleCode: s.code, Severity: s.severity, Summary: s.summary, EvidenceRequirement: s.evidence, Status: "open"})
	}
	return out
}
