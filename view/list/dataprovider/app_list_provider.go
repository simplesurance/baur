package dataprovider

import (
	"fmt"
	"strconv"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

// AppListProvider is the provider for the list of applications
type AppListProvider struct {
	apps   []*baur.App
	storer storage.Storer
	data   [][]string
}

// NewAppListProvider is the constructor of the app list provider
func NewAppListProvider(apps []*baur.App, storer storage.Storer) *AppListProvider {
	return &AppListProvider{
		apps:   apps,
		storer: storer,
	}
}

// FetchData fetches data
func (p *AppListProvider) FetchData() {
	for _, app := range p.apps {
		status, build, err := baur.GetBuildStatus(p.storer, app)
		if err != nil {
			log.Fatalf("evaluating build status of app '%s' failed: %s\n", app, err)
		}

		// Show by default the following columns: Name, Build-Status, Build-id, Git commit
		p.data = append(p.data, []string{
			app.Name,
			status.String(),
			getBuildIDStr(build),
			vcsStr(build),
		})
	}

	return
}

func getBuildIDStr(build *storage.Build) string {
	if build == nil {
		return ""
	}

	return strconv.Itoa(build.ID)
}

// GetData implements the provider interface
func (p *AppListProvider) GetData() (data [][]string) {
	return p.data
}

func vcsStr(b *storage.Build) string {
	if b == nil {
		return ""
	}

	vcsState := b.VCSState

	if len(vcsState.CommitID) == 0 {
		return ""
	}

	if vcsState.IsDirty {
		return fmt.Sprintf("%s-dirty", vcsState.CommitID)
	}

	return vcsState.CommitID
}
