package resolver

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/google/uuid"
)

const (
	gitcommitFunc = "gitCommit"
	envFunc       = "env"
	uuidFunc      = "uuid"
)

// GoTemplate parses a string as go-template and executes it.
type GoTemplate struct {
	template     *template.Template
	templateVars *vars
}

type vars struct {
	Root    string
	AppName string
}

func lookupEnv(envVarName string) (string, error) {
	envVal, exist := os.LookupEnv(envVarName)
	if !exist {
		return "", fmt.Errorf("environment variable %q is undefined", envVarName)
	}

	return envVal, nil
}

func NewGoTemplate(appName, root string, gitCommitFn func() (string, error)) *GoTemplate {
	templateVars := vars{
		Root:    root,
		AppName: appName,
	}

	funcMap := template.FuncMap{
		gitcommitFunc: gitCommitFn,
		envFunc:       lookupEnv,
		uuidFunc:      uuid.NewString,
	}

	return &GoTemplate{
		templateVars: &templateVars,
		template:     template.New("baur").Funcs(funcMap).Option("missingkey=error"),
	}
}

// Resolve parses in as go template, executes it and returns the resulting text.
func (s *GoTemplate) Resolve(in string) (string, error) {
	t, err := s.template.Parse(in)
	if err != nil {
		return "", fmt.Errorf("parsing as go template failed: %w", err)
	}

	output := new(bytes.Buffer)
	if err = t.Execute(output, s.templateVars); err != nil {
		return "", fmt.Errorf("templating failed: %w", err)
	}

	return output.String(), nil
}
