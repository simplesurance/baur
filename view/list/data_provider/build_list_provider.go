package data_provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
	"strconv"
	"time"
)

type buildListProvider struct {
	storer storage.Storer
	data   [][]string
}

func NewBuildListProvider(storer storage.Storer) *buildListProvider {
	return &buildListProvider{
		storer: storer,
	}
}

func (p *buildListProvider) FetchData(filters []storage.CanFilter, sorters []storage.CanSort) error {
	data, err := p.storer.GetBuilds(filters, sorters)
	if err != nil {
		return errors.Wrap(err, "error while trying to retrieve builds")
	}

	p.data = buildsToStrings(data)

	return nil
}

func (p *buildListProvider) GetData() (data [][]string) {
	return p.data
}

func buildsToStrings(buildsWithDuration []*storage.BuildWithDuration) (strings [][]string) {
	for _, buildWithDuration := range buildsWithDuration {
		build := buildWithDuration.Build

		strings = append(strings, []string{
			strconv.Itoa(build.ID),
			build.Application.Name,
			build.StartTimeStamp.Format(time.RFC850),
			fmt.Sprintf("%.0f", buildWithDuration.Duration),
			build.TotalInputDigest,
		})
	}

	return
}
