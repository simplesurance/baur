package baur

// App represents an application
type App struct {
	Name string
	Dir  string
}

// AppDiscover is an interface for discovering applications in a directory
type AppDiscover interface {
	Apps() ([]App, error)
}

// AppCfg is an interface for an application configuration
type AppCfg interface {
	GetName() string
	Validate() error
}

// AppConfigReader is an interface for an application config reader and parser.
type AppCfgReader interface {
	AppFromFile(path string) (AppCfg, error)
}
