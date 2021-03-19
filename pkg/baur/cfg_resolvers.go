package baur

import (
	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

const (
	// bc convertions - in gotemplate functions are called without dot
	uuidOldVarname = "{{ .uuid }}"
	uuidNewVarname = "{{ uuid }}"

	gitCommitOldVarname = "{{ .gitcommit }}"
	gitCommitNewVarname = "{{ gitcommit }}"
)

// defaultAppCfgResolvers returns the default set of resolvers that is applied on application configs.
func defaultAppCfgResolvers(rootPath, appName string, gitCommitFn func() (string, error)) resolver.Resolver {
	return resolver.List{
		&resolver.StrReplacement{Old: uuidOldVarname, New: uuidNewVarname},
		&resolver.StrReplacement{Old: gitCommitOldVarname, New: gitCommitNewVarname},
		resolver.NewGoTemplate(appName, rootPath, gitCommitFn),
	}
}
