package term

import (
	"github.com/fatih/color"

	"github.com/simplesurance/baur/v3/pkg/baur"
)

var (
	GreenHighlight  = color.New(color.FgGreen).SprintFunc()
	RedHighlight    = color.New(color.FgRed).SprintFunc()
	YellowHighlight = color.New(color.FgYellow).SprintFunc()

	MagentaHighlight = color.New(color.FgMagenta).SprintFunc()

	Underline = color.New(color.Underline).SprintFunc()

	Highlight = MagentaHighlight
)

func ColoredTaskStatus(status baur.TaskStatus) string {
	switch status {
	case baur.TaskStatusRunExist:
		return GreenHighlight(status.String())
	case baur.TaskStatusExecutionPending:
		return RedHighlight(status.String())
	default:
		return status.String()
	}
}
