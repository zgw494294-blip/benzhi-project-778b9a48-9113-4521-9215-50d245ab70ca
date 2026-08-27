package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type CommandTx struct {
	tx  *sql.Tx
	now time.Time
}

func (s *Store) Command(ctx context.Context, key, operation, caseID, actor string, fn func(*CommandTx) (any, error)) (json.RawMessage, bool, error) {
	if key == "" {
		return nil, false, errors.New("idempotencyKey 不能为空")
	}
	var saved, savedOperation string
	err := s.db.QueryRowContext(ctx, "SELECT operation,response FROM idempotency WHERE idempotency_key=?", key).Scan(&savedOperation, &saved)
	if err == nil {
		if savedOperation != operation {
			return nil, false, errors.New("idempotencyKey 已被其他操作使用")
		}
		return json.RawMessage(saved), true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	ct := &CommandTx{tx: tx, now: time.Now().UTC()}
	value, err := fn(ct)
	if err != nil {
		return nil, false, err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO idempotency(idempotency_key,operation,case_id,response,created_at) VALUES(?,?,?,?,?)", key, operation, nullable(caseID), string(body), ct.now.Format(time.RFC3339Nano)); err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events(case_id,event_type,actor,detail,occurred_at) VALUES(?,?,?,?,?)", caseID, operation, actor, string(body), ct.now.Format(time.RFC3339Nano)); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return body, false, nil
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}
