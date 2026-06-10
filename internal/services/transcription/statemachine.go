//go:build completion

package transcription

import (
	"fmt"
)

// ValidTransitions 定义状态机允许的转换映射（对齐文档 §8.6）。
//
// 当前状态 → 允许的目标状态集合。
var ValidTransitions = map[TaskStatus][]TaskStatus{
	StatusQueued:    {StatusRunning, StatusCancelled},
	StatusRunning:   {StatusCancelling, StatusCompleted, StatusFailed},
	StatusCancelling: {StatusCancelled, StatusFailed},
	StatusCompleted: {}, // 终态：只允许 ImportResultAsMidiProject（不改变任务状态）
	StatusFailed:    {}, // 终态
	StatusCancelled: {}, // 终态
}

// IsTerminal 返回该状态是否为终态。
func IsTerminal(s TaskStatus) bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

// CanCancel 返回该状态是否允许取消。
func CanCancel(s TaskStatus) bool {
	switch s {
	case StatusQueued, StatusRunning:
		return true
	}
	return false
}

// ValidateTransition 校验状态转换是否合法。
func ValidateTransition(current, target TaskStatus) error {
	if current == target {
		return nil
	}
	allowed, ok := ValidTransitions[current]
	if !ok {
		return fmt.Errorf("unknown current status: %s", current)
	}
	for _, a := range allowed {
		if a == target {
			return nil
		}
	}
	return &InvalidStateError{Current: string(current), Requested: string(target)}
}

// InvalidStateError 表示非法的状态转换。
type InvalidStateError struct {
	Current   string
	Requested string
}

func (e *InvalidStateError) Error() string {
	return fmt.Sprintf("transcription.invalid_state: cannot transition from %s to %s", e.Current, e.Requested)
}
