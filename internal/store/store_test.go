package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"textilepermit/internal/domain"
	"time"
)

func TestPersistenceAndIdempotency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	c := domain.ArtifactCase{CaseID: "c1", AccessionCode: "A1", Title: "藏品", MaterialProfile: "丝", DyeSensitivity: "high", FragileAreas: "边", AnnualLuxHourLimit: 100, Status: domain.StatusDraft, Version: 1, CreatedBy: "u1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	first, _, e := s.Command(ctx, "key1", "case.created", c.CaseID, "u1", func(tx *CommandTx) (any, error) { return c, tx.InsertCase(ctx, c) })
	if e != nil {
		t.Fatal(e)
	}
	second, reused, e := s.Command(ctx, "key1", "case.created", c.CaseID, "u2", func(tx *CommandTx) (any, error) { t.Fatal("不应重复执行"); return nil, nil })
	if e != nil || !reused || string(first) != string(second) {
		t.Fatalf("幂等失败 reused=%v err=%v", reused, e)
	}
	if _, _, e = s.Command(ctx, "key1", "different", c.CaseID, "u2", func(tx *CommandTx) (any, error) { return nil, nil }); e == nil {
		t.Fatal("同一幂等键不得跨操作复用")
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e := s.Case(ctx, "c1")
	if e != nil || got.Title != "藏品" {
		t.Fatalf("重启持久化失败 %+v %v", got, e)
	}
	events, e := s.Audit(ctx, "c1")
	if e != nil || len(events) != 1 || events[0].Sequence < 1 {
		t.Fatalf("审计失败 %+v %v", events, e)
	}
	var decoded domain.ArtifactCase
	if json.Unmarshal(first, &decoded) != nil || decoded.CaseID != "c1" {
		t.Fatal("响应不可解码")
	}
}

func TestExpectedVersionConflict(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "v.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	c := domain.ArtifactCase{CaseID: "c", AccessionCode: "A", Title: "T", MaterialProfile: "M", DyeSensitivity: "low", FragileAreas: "F", AnnualLuxHourLimit: 1, Status: domain.StatusDraft, Version: 1, CreatedBy: "u", CreatedAt: now, UpdatedAt: now}
	_, _, e = s.Command(ctx, "k", "create", "c", "u", func(tx *CommandTx) (any, error) { return c, tx.InsertCase(ctx, c) })
	if e != nil {
		t.Fatal(e)
	}
	_, _, e = s.Command(ctx, "k2", "update", "c", "u", func(tx *CommandTx) (any, error) { c.Version = 2; return c, tx.UpdateCase(ctx, c, 99) })
	if e != domain.ErrConflict {
		t.Fatalf("期望版本冲突，得到 %v", e)
	}
}
