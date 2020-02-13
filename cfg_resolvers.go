package baur

import "github.com/simplesurance/baur/cfg/resolver"

const (
	rootVarName      = "$ROOT"
	appVarName       = "$APPNAME"
	uuidVarname      = "$UUID"
	gitCommitVarname = "$GITCOMMIT"
)

// DefaultAppCfgResolvers returns the default set of resolvers that is applied on application configs.
func DefaultAppCfgResolvers(rootPath, appName string, gitCommitFn func() (string, error)) resolver.Resolver {
	return resolver.List{
		&resolver.StrReplacement{Old: appVarName, New: appName},
		&resolver.StrReplacement{Old: rootVarName, New: rootPath},
		&resolver.UUIDVar{Old: uuidVarname},
		&resolver.CallbackReplacement{Old: gitCommitVarname, NewFunc: gitCommitFn},
	}
}

// IncludeCfgVarResolvers returns the default resolvers for variables in the
// Includes field in config files.
func IncludeCfgVarResolvers(rootPath, appName string) resolver.Resolver {
	return resolver.List{
		&resolver.StrReplacement{Old: appVarName, New: appName},
		&resolver.StrReplacement{Old: rootVarName, New: rootPath},
	}
}
