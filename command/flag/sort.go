package flag

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
)

// SortFlagFormatDescr contains the format description of SortFlag values
const SortFlagFormatDescr = "<FIELD>-<ORDER>"

// SortFlagValue implements pflag.Value
type SortFlagValue struct {
	*postgres.Sorter
}

// String returns an empty string
func (v *SortFlagValue) String() string {
	return ""
}

// Set implementing the Value interface
func (v *SortFlagValue) Set(sortStr string) error {
	sort, err := parseSortFlag(sortStr)
	if err != nil {
		return errors.Wrap(err, "error while parsing sort")
	}

	v.Sorter = sort

	return nil
}

// Type returns the value description string
func (*SortFlagValue) Type() string {
	return SortFlagFormatDescr
}

func parseSortFlag(str string) (*postgres.Sorter, error) {
	pieces := strings.Split(str, "-")
	if len(pieces) != 2 {
		return nil, fmt.Errorf("invalid format, format must be %s", SortFlagFormatDescr)
	}

	field := strings.ToLower(pieces[0])
	order := strings.ToLower(pieces[1])

	if order != "asc" && order != "desc" {
		return nil, errors.New("invalid sort order, must be \"asc\" or \"desc\"")
	}

	if field != "time" && field != "duration" {
		return nil, errors.New("invalid sorting field, must be \"time\" or \"duration\"")
	}

	var sortField storage.Field
	if field == "time" {
		sortField = storage.FieldBuildStartDatetime
	} else {
		sortField = storage.FieldDuration
	}

	var sortDirection storage.OrderDirection
	if order == "asc" {
		sortDirection = storage.OrderAsc
	} else {
		sortDirection = storage.OrderDesc
	}

	return postgres.NewSorter(sortField, sortDirection), nil
}
