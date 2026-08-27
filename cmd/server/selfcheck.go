package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type selfCase struct {
	CaseID  string `json:"caseId"`
	Version int64  `json:"version"`
}
type selfPlan struct {
	Case selfCase `json:"case"`
}
type selfReview struct {
	Case   selfCase `json:"case"`
	Permit *struct {
		VerificationCode string `json:"verificationCode"`
	} `json:"permit"`
}
type selfVerify struct {
	Valid  bool   `json:"valid"`
	Status string `json:"status"`
}

func runSelfcheck(ctx context.Context, base string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	if err := waitReady(ctx, client, base); err != nil {
		return err
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	var c selfCase
	if err := post(ctx, client, base+"/api/cases", map[string]any{"idempotencyKey": "self-create-" + suffix, "accessionCode": "SELF-" + suffix, "title": "自检纺织藏品", "materialProfile": "丝织物与矿物染料", "dyeSensitivity": "medium", "fragileAreas": "右下角轻微纤维松动，已托衬", "historicalLuxHours": 1000, "annualLuxHourLimit": 30000, "actor": "self-protector"}, &c); err != nil {
		return fmt.Errorf("自检建档: %w", err)
	}
	today := time.Now().UTC()
	var p selfPlan
	if err := post(ctx, client, base+"/api/cases/"+c.CaseID+"/plans", map[string]any{"idempotencyKey": "self-plan-" + suffix, "expectedVersion": c.Version, "cabinetCode": "SELF-CABINET", "lightingSlots": []map[string]any{{"name": "开放时段", "lux": 40, "minutes": 360}}, "dailyOpenMinutes": 360, "displayStartDate": today.Format(time.DateOnly), "displayEndDate": today.AddDate(0, 0, 9).Format(time.DateOnly), "restRotationDays": 1, "uvProtection": true, "actor": "self-designer"}, &p); err != nil {
		return fmt.Errorf("自检测算: %w", err)
	}
	var submitted selfCase
	if err := post(ctx, client, base+"/api/cases/"+c.CaseID+"/submit-review", map[string]any{"idempotencyKey": "self-submit-" + suffix, "expectedVersion": p.Case.Version, "actor": "self-protector"}, &submitted); err != nil {
		return fmt.Errorf("自检送审: %w", err)
	}
	var reviewed selfReview
	if err := post(ctx, client, base+"/api/cases/"+c.CaseID+"/review", map[string]any{"idempotencyKey": "self-review-" + suffix, "expectedVersion": submitted.Version, "reviewerId": "self-reviewer", "outcome": "approved", "reason": "自检证据满足保护要求", "validDays": 10}, &reviewed); err != nil {
		return fmt.Errorf("自检批准: %w", err)
	}
	if reviewed.Permit == nil {
		return fmt.Errorf("自检未签发凭据")
	}
	var verified selfVerify
	if err := getJSON(ctx, client, base+"/api/permits/verify?code="+reviewed.Permit.VerificationCode, &verified); err != nil {
		return fmt.Errorf("自检验真: %w", err)
	}
	if !verified.Valid {
		return fmt.Errorf("自检凭据无效: %s", verified.Status)
	}
	return nil
}

func waitReady(ctx context.Context, c *http.Client, base string) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		var v map[string]any
		if getJSON(ctx, c, base+"/api/ready", &v) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
func post(ctx context.Context, c *http.Client, url string, value, out any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	return execute(c, req, out)
}
func getJSON(ctx context.Context, c *http.Client, url string, out any) error {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if e != nil {
		return e
	}
	return execute(c, req, out)
}
func execute(c *http.Client, req *http.Request, out any) error {
	r, e := c.Do(req)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", r.StatusCode, string(b))
	}
	return json.NewDecoder(r.Body).Decode(out)
}
