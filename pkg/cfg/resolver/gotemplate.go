package resolver

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"

	"github.com/google/uuid"
	"github.com/simplesurance/baur/v1/internal/vcs"
)

type GoTemplate struct {
	appName     string
	root        string
	gitCommitFn func() (string, error)
}

func NewGoTemplate(appName, root string, gitCommitFn func() (string, error)) Resolver {
	return &GoTemplate{
		appName:     appName,
		root:        root,
		gitCommitFn: gitCommitFn,
	}
}

func (s *GoTemplate) Resolve(in string) (string, error) {
	templateVars := map[string]string{
		"root":    s.root,
		"appname": s.appName,
	}

	funcMap := template.FuncMap{
		"gitcommit": func() (string, error) {
			commit, err := s.gitCommitFn()
			if errors.Is(err, vcs.ErrVCSRepositoryNotExist) {
				return "", errors.New("baur repository is not part of a git repository")
			}

			return commit, err

		},
		"env": func(envVarName string) (string, error) {
			envVal, exist := os.LookupEnv(envVarName)
			if !exist {
				return "", fmt.Errorf("environment variable %q is undefined", envVarName)
			}

			return envVal, nil
		},
		"uuid": func() string {
			return uuid.NewString()
		},
	}

	t, err := template.New("baur").Funcs(funcMap).Parse(in)
	if err != nil {
		return "", fmt.Errorf("failed parsing go template: %w", err)
	}

	output := new(bytes.Buffer)
	if err = t.Execute(output, templateVars); err != nil {
		return "", fmt.Errorf("failed evaluating template: %w", err)
	}

	return output.String(), nil
}
