package data_provider

import (
	"fmt"
	"strconv"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

type appListProvider struct {
	apps   []*baur.App
	storer storage.Storer
	data   [][]string
}

func NewAppListProvider(apps []*baur.App, storer storage.Storer) *appListProvider {
	return &appListProvider{
		apps:   apps,
		storer: storer,
	}
}

func (p *appListProvider) FetchData() {
	for _, app := range p.apps {
		status, build, err := baur.GetBuildStatus(p.storer, app)
		if err != nil {
			log.Fatalf("evaluating build status of app '%s' failed: %s\n", app, err)
		}

		// Show by default the following columns: Name, Build-Status, Build-id, Git commit
		p.data = append(p.data, []string{
			app.Name,
			status.String(),
			getBuildIdStr(build),
			vcsStr(build),
		})
	}

	return
}

func getBuildIdStr(build *storage.Build) string {
	if build == nil {
		return ""
	}

	return strconv.Itoa(build.ID)
}

func (p *appListProvider) GetData() (data [][]string) {
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
