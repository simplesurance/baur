// Package resolver provides templating for baur configuration files.
package resolver

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/google/uuid"
)

const (
	gitCommitFuncName = "gitCommit"
	envFuncName       = "env"
	uuidFuncName      = "uuid"
)

// GoTemplate parses template strings and executes the template statements.
type GoTemplate struct {
	template     *template.Template
	templateVars *vars
}

// vars defines the fields that are available in the template.
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

// NewGoTemplate returns a GoTemplate instance.
// The .AppName and .Root variables are initialized with appName and root.
// gitCommitFn is the function that is called via {{ gitCommit }} in a template.
func NewGoTemplate(appName, root string, gitCommitFn func() (string, error)) *GoTemplate {
	templateVars := vars{
		Root:    root,
		AppName: appName,
	}

	funcMap := template.FuncMap{
		gitCommitFuncName: gitCommitFn,
		envFuncName:       lookupEnv,
		uuidFuncName:      uuid.NewString,
	}

	return &GoTemplate{
		templateVars: &templateVars,
		template:     template.New("baur").Funcs(funcMap).Option("missingkey=error"),
	}
}

// Resolve parses the parameter "in" as Go template, executes it and returns
// the result.
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
