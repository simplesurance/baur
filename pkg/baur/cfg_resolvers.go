package baur

import (
	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

// defaultAppCfgResolvers returns the default set of resolvers that is applied on application configs.
func defaultAppCfgResolvers(rootPath, appName string, gitCommitFn func() (string, error)) resolver.Resolver {
	return resolver.List{
		resolver.NewGoTemplate(appName, rootPath, gitCommitFn),
	}
}
