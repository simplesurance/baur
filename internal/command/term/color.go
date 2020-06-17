package term

import (
	"github.com/fatih/color"

	"github.com/simplesurance/baur"
)

var (
	GreenHighlight  = color.New(color.FgGreen).SprintFunc()
	RedHighlight    = color.New(color.FgRed).SprintFunc()
	YellowHighlight = color.New(color.FgYellow).SprintFunc()

	Underline = color.New(color.Underline).SprintFunc()

	Highlight = GreenHighlight
)

func ColoredTaskStatus(status baur.TaskStatus) string {
	switch status {
	case baur.TaskStatusInputsUndefined:
		return YellowHighlight(status.String())
	case baur.TaskStatusRunExist:
		return GreenHighlight(status.String())
	case baur.TaskStatusExecutionPending:
		return RedHighlight(status.String())
	default:
		return status.String()
	}
}
