package resolver

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/google/uuid"

	"github.com/simplesurance/baur/v1/internal/vcs"
)

const (
	rootVar       = "root"
	appnameVar    = "appname"
	gitcommitFunc = "gitcommit"
	envFunc       = "env"
	uuidFunc      = "uuid"
)

type GoTemplate struct {
	appName      string
	root         string
	gitCommitFn  func() (string, error)
	templateVars map[string]string
	funcMap      template.FuncMap
}

func newUUID() string {
	return uuid.NewString()
}

func lookupEnv(envVarName string) (string, error) {
	envVal, exist := os.LookupEnv(envVarName)
	if !exist {
		return "", fmt.Errorf("environment variable %q is undefined", envVarName)
	}

	return envVal, nil
}

func (s *GoTemplate) gitCommit() (string, error) {
	commit, err := s.gitCommitFn()
	if errors.Is(err, vcs.ErrVCSRepositoryNotExist) {
		return "", errors.New("baur repository is not part of a git repository")
	}

	return commit, err
}

func NewGoTemplate(appName, root string, gitCommitFn func() (string, error)) *GoTemplate {
	result := &GoTemplate{
		appName:     appName,
		root:        root,
		gitCommitFn: gitCommitFn,
		templateVars: map[string]string{
			rootVar:    root,
			appnameVar: appName,
		},
	}

	result.funcMap = template.FuncMap{
		gitcommitFunc: result.gitCommit,
		envFunc:       lookupEnv,
		uuidFunc:      newUUID,
	}

	return result
}

func (s *GoTemplate) Resolve(in string) (string, error) {
	t, err := template.New("baur").Funcs(s.funcMap).Parse(in)
	if err != nil {
		return "", fmt.Errorf("failed parsing go template: %w", err)
	}

	output := new(bytes.Buffer)
	if err = t.Execute(output, s.templateVars); err != nil {
		return "", fmt.Errorf("failed evaluating template: %w", err)
	}

	return output.String(), nil
}
