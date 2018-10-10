package flag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simplesurance/baur/storage"
)

// SortFlagFormatDescr contains the format description of SortFlag values
const SortFlagFormatDescr = "<FIELD>-<ORDER>"

// SortFlagValue implements pflag.Value
type SortFlagValue struct {
	storage.Sorter
}

// String returns the Field and Order as string
func (s *SortFlagValue) String() string {
	return fmt.Sprintf("%s-%s", s.Field, s.Order)
}

// Set parses the passed string and sets the SortFlagValue
func (s *SortFlagValue) Set(sortStr string) error {
	var err error

	pieces := strings.Split(sortStr, "-")
	if len(pieces) != 2 {
		return fmt.Errorf("format must be %s", SortFlagFormatDescr)
	}

	switch strings.ToLower(pieces[0]) {
	case "time":
		s.Field = storage.FieldBuildStartTime
	case "duration":
		s.Field = storage.FieldBuildDuration
	default:
		return errors.New("field must be \"time\" or \"duration\"")
	}

	s.Order, err = storage.OrderFromStr(pieces[1])
	if err != nil {
		return err
	}

	return nil
}

// Type returns the value description string
func (*SortFlagValue) Type() string {
	return SortFlagFormatDescr
}
