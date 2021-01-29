package baur

import (
	"fmt"
)

// TaskStatus describes the status of the task
type TaskStatus int

const (
	_ TaskStatus = iota
	TaskStatusUndefined
	TaskStatusRunExist
	TaskStatusExecutionPending
)

func (b TaskStatus) String() string {
	switch b {
	case TaskStatusUndefined:
		return "Undefined"
	case TaskStatusRunExist:
		return "Exist"
	case TaskStatusExecutionPending:
		return "Pending"

	default:
		panic(fmt.Sprintf("undefined TaskStatus value: %d", b))
	}
}
