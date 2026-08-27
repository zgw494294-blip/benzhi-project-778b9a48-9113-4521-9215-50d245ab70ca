package domain

import "errors"

var (
	ErrNotFound   = errors.New("记录不存在")
	ErrConflict   = errors.New("案卷版本冲突")
	ErrInvalid    = errors.New("输入不符合业务规则")
	ErrForbidden  = errors.New("当前状态不允许此操作")
	ErrSeparation = errors.New("申请人与最终复核人必须不同")
)

type RuleError struct{ Field, Message string }

func (e *RuleError) Error() string { return e.Field + ": " + e.Message }
