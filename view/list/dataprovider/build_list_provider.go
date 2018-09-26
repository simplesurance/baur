package dataprovider

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

// BuildListProvider is the provider for the list of builds
type BuildListProvider struct {
	storer storage.Storer
	data   [][]string
}

// NewBuildListProvider constructor
func NewBuildListProvider(storer storage.Storer) *BuildListProvider {
	return &BuildListProvider{
		storer: storer,
	}
}

// FetchData fetches data
func (p *BuildListProvider) FetchData(filters []storage.CanFilter, sorters []storage.CanSort) error {
	data, err := p.storer.GetBuilds(filters, sorters)
	if err != nil {
		return errors.Wrap(err, "error while trying to retrieve builds")
	}

	p.data = buildsToStrings(data)

	return nil
}

// GetData implements the provider interface
func (p *BuildListProvider) GetData() (data [][]string) {
	return p.data
}

func buildsToStrings(buildsWithDuration []*storage.BuildWithDuration) (strings [][]string) {
	for _, buildWithDuration := range buildsWithDuration {
		build := buildWithDuration.Build

		strings = append(strings, []string{
			strconv.Itoa(build.ID),
			build.Application.Name,
			build.StartTimeStamp.Format(DateTimeFormat),
			fmt.Sprintf("%.0f", buildWithDuration.Duration),
			build.TotalInputDigest,
		})
	}

	return
}
