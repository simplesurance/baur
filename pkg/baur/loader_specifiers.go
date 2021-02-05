package baur

import (
	"fmt"
	"strings"

	"github.com/simplesurance/baur"
)

type taskSpec struct {
	appName  string
	taskName string
}

func (t *taskSpec) String() string {
	return fmt.Sprintf("%s.%s", t.appName, t.taskName)
}

type specs struct {
	// all is true if all tasks of all apps are matched
	all bool
	// allApps is true if >=1 spec matches all apps
	allApps   bool
	appDirs   []string
	appNames  []string
	taskSpecs []*taskSpec
}

// parseSpecs parses the task and app specifiers and returns a new *specs object.
// The following specifiers are supported:
// - '*' to match all apps and tasks,
// - <APP-DIR-PATH> path to an application directory containing an .app.toml file,
// <APP-SPEC>[.<TASK-SPEC>] where:
//     <APP-SPEC> is:
//       - <APP-NAME> or
//       - '*'
//     <TASK-SPEC> is:
//       - Task Name or
//       - '*'
func parseSpecs(specifiers []string) (*specs, error) {
	var result specs

	for _, spec := range specifiers {
		if spec == "*" {
			result.all = true
			result.allApps = true
			return &result, nil
		}

		if isAppDirectory(spec) {
			result.appDirs = append(result.appDirs, spec)
			continue
		}

		if spec == "." {
			return nil, fmt.Errorf("current directory does not contain an %q file", baur.AppCfgFile)
		}

		if !strings.Contains(spec, ".") {
			result.appNames = append(result.appNames, spec)
			continue
		}

		spl := strings.Split(spec, ".")
		switch len(spl) {
		case 0:
			// impossible condition
			panic(fmt.Sprintf("strings.Split(%q, \".\") returned empty slice", spec))
		case 1:
			result.appNames = append(result.appNames, spl[0])
		case 2:
			appName := spl[0]
			taskName := spl[1]

			if appName == "*" {
				result.allApps = true
				if taskName == "*" {
					result.all = true

					break
				}
			}

			if taskName == "*" {
				result.appNames = append(result.appNames, appName)
				continue
			}

			result.taskSpecs = append(result.taskSpecs, &taskSpec{appName: appName, taskName: taskName})

		default:
			return nil, fmt.Errorf("invalid specifier: %q is not a path to an existing directory and contains > 1 dots ", spec)
		}
	}

	return &result, nil
}
