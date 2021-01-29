package baur

import (
	"errors"

	"github.com/simplesurance/baur/v1/internal/vcs"
	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

const (
	rootVarName      = "{{ .root }}"
	appVarName       = "{{ .appname }}"
	uuidVarname      = "{{ .uuid }}"
	gitCommitVarname = "{{ .gitcommit }}"
)

// DefaultAppCfgResolvers returns the default set of resolvers that is applied on application configs.
func DefaultAppCfgResolvers(rootPath, appName string, gitCommitFn func() (string, error)) resolver.Resolver {
	return resolver.List{
		&resolver.StrReplacement{Old: appVarName, New: appName},
		&resolver.StrReplacement{Old: rootVarName, New: rootPath},
		&resolver.UUIDVar{Old: uuidVarname},
		&resolver.CallbackReplacement{
			Old: gitCommitVarname,
			NewFunc: func() (string, error) {
				commit, err := gitCommitFn()
				if errors.Is(err, vcs.ErrVCSRepositoryNotExist) {
					return "", errors.New("baur repository is not part of a git repository")
				}

				return commit, err

			},
		},
		&resolver.EnvVar{},
	}
}

// IncludeCfgVarResolvers returns the default resolvers for variables in the
// Includes field in config files.
func IncludeCfgVarResolvers(rootPath, appName string) resolver.Resolver {
	// TODO: do we really need to distinguish between resolvers for include directives and all other fields?
	// We should be able to use the the same set of resolvers for all
	// fields. If somebody wants to use {{ .gitcommit }} in their include
	// path, they have to cope with it. :-)
	return resolver.List{
		&resolver.StrReplacement{Old: appVarName, New: appName},
		&resolver.StrReplacement{Old: rootVarName, New: rootPath},
		&resolver.EnvVar{},
	}
}
