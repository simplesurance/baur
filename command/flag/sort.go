package flag

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
)

// SortFlagValue implements pflag.Value
type SortFlagValue struct {
	*postgres.Sorter
}

// String implementing the Value interface
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

// Type returns the name of the sort flag. Implementing the Value interface
func (*SortFlagValue) Type() string {
	return "sort"
}

func parseSortFlag(str string) (*postgres.Sorter, error) {
	pieces := strings.Split(str, "-")
	if len(pieces) != 2 {
		return nil, errors.New("sorting string doesn't have 2 pieces")
	}

	first := strings.ToLower(pieces[0])
	second := strings.ToLower(pieces[1])

	if (first != "time" && first != "duration") || (second != "asc" && second != "desc") {
		return nil, errors.New("invalid sorting field or direction")
	}

	var sortField storage.Field
	if first == "time" {
		sortField = storage.FieldBuildStartDatetime
	} else {
		sortField = storage.FieldDuration
	}

	var sortDirection storage.OrderDirection
	if second == "asc" {
		sortDirection = storage.OrderAsc
	} else {
		sortDirection = storage.OrderDesc
	}

	return postgres.NewSorter(sortField, sortDirection), nil
}
