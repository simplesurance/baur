package baur

import "github.com/simplesurance/baur/cfg/resolver"

const (
	rootVarName      = "$ROOT"
	appVarName       = "$APPNAME"
	uuidVarname      = "$UUID"
	gitCommitVarname = "$GITCOMMIT"
)

// DefaultAppCfgResolvers returns the default set of resolvers that is applied on application configs.
func DefaultAppCfgResolvers(rootPath, appName, gitCommit string) resolver.Resolver {
	return resolver.List{
		&resolver.StrReplacement{Old: appVarName, New: appName},
		&resolver.StrReplacement{Old: rootVarName, New: rootPath},
		&resolver.UUIDVar{Old: uuidVarname},
		&resolver.StrReplacement{Old: gitCommitVarname, New: gitCommit},
	}
}
