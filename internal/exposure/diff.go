package exposure

import "textilepermit/internal/domain"

type Difference struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

func Compare(before, after domain.DisplayPlanRevision) []Difference {
	var d []Difference
	if before.CabinetCode != after.CabinetCode {
		d = append(d, Difference{"cabinetCode", before.CabinetCode, after.CabinetCode})
	}
	if before.DailyOpenMinutes != after.DailyOpenMinutes {
		d = append(d, Difference{"dailyOpenMinutes", before.DailyOpenMinutes, after.DailyOpenMinutes})
	}
	if before.DisplayStartDate != after.DisplayStartDate {
		d = append(d, Difference{"displayStartDate", before.DisplayStartDate, after.DisplayStartDate})
	}
	if before.DisplayEndDate != after.DisplayEndDate {
		d = append(d, Difference{"displayEndDate", before.DisplayEndDate, after.DisplayEndDate})
	}
	if before.RestRotationDays != after.RestRotationDays {
		d = append(d, Difference{"restRotationDays", before.RestRotationDays, after.RestRotationDays})
	}
	if before.UVProtection != after.UVProtection {
		d = append(d, Difference{"uvProtection", before.UVProtection, after.UVProtection})
	}
	if digestSlots(before.LightingSlots) != digestSlots(after.LightingSlots) {
		d = append(d, Difference{"lightingSlots", before.LightingSlots, after.LightingSlots})
	}
	return d
}

func digestSlots(v []domain.LightingSlot) string { s, _ := domain.Digest(v); return s }
