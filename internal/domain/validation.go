package domain

import (
	"fmt"
	"strings"
	"time"
)

func ValidateCase(c ArtifactCase) error {
	if strings.TrimSpace(c.AccessionCode) == "" {
		return &RuleError{"accessionCode", "藏品编号不能为空"}
	}
	if strings.TrimSpace(c.Title) == "" {
		return &RuleError{"title", "藏品名称不能为空"}
	}
	if strings.TrimSpace(c.MaterialProfile) == "" {
		return &RuleError{"materialProfile", "必须记录材质构成"}
	}
	switch c.DyeSensitivity {
	case "high", "medium", "low":
	default:
		return &RuleError{"dyeSensitivity", "必须为 high、medium 或 low"}
	}
	if c.HistoricalLuxHours < 0 {
		return &RuleError{"historicalLuxHours", "历史曝光量不能为负数"}
	}
	if c.AnnualLuxHourLimit <= 0 {
		return &RuleError{"annualLuxHourLimit", "年度剂量上限必须大于零"}
	}
	if c.HistoricalLuxHours > c.AnnualLuxHourLimit*5 {
		return &RuleError{"historicalLuxHours", "历史曝光量异常偏高，请复核单位"}
	}
	return nil
}

func ValidatePlan(p DisplayPlanRevision) error {
	if strings.TrimSpace(p.CabinetCode) == "" {
		return &RuleError{"cabinetCode", "展柜编号不能为空"}
	}
	if len(p.LightingSlots) == 0 {
		return &RuleError{"lightingSlots", "至少登记一个照明时段"}
	}
	totalMinutes := 0
	names := map[string]bool{}
	for i, slot := range p.LightingSlots {
		if strings.TrimSpace(slot.Name) == "" {
			return &RuleError{fmt.Sprintf("lightingSlots[%d].name", i), "时段名称不能为空"}
		}
		name := strings.TrimSpace(slot.Name)
		if names[name] {
			return &RuleError{fmt.Sprintf("lightingSlots[%d].name", i), "同一方案内时段名称不能重复"}
		}
		names[name] = true
		if slot.Lux <= 0 || slot.Lux > 2000 {
			return &RuleError{fmt.Sprintf("lightingSlots[%d].lux", i), "照度必须在 0 到 2000 lux 之间"}
		}
		if slot.Minutes <= 0 {
			return &RuleError{fmt.Sprintf("lightingSlots[%d].minutes", i), "时段分钟数必须大于零"}
		}
		totalMinutes += slot.Minutes
	}
	if p.DailyOpenMinutes <= 0 || p.DailyOpenMinutes > 1440 {
		return &RuleError{"dailyOpenMinutes", "每日开放分钟数必须在 1 到 1440 之间"}
	}
	if totalMinutes != p.DailyOpenMinutes {
		return &RuleError{"lightingSlots", "各时段分钟数之和必须等于每日开放分钟数"}
	}
	start, err := time.Parse(time.DateOnly, p.DisplayStartDate)
	if err != nil {
		return &RuleError{"displayStartDate", "日期格式必须为 YYYY-MM-DD"}
	}
	end, err := time.Parse(time.DateOnly, p.DisplayEndDate)
	if err != nil {
		return &RuleError{"displayEndDate", "日期格式必须为 YYYY-MM-DD"}
	}
	if end.Before(start) {
		return &RuleError{"displayEndDate", "结束日期不得早于开始日期"}
	}
	if end.Sub(start).Hours()/24 > 730 {
		return &RuleError{"displayEndDate", "单次方案展期不得超过两年"}
	}
	if p.RestRotationDays < 0 {
		return &RuleError{"restRotationDays", "轮换休息天数不能为负数"}
	}
	return nil
}
