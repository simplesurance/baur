package flag

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateTimeFlagValueSetWithoutTZIsInLocalTimezone(t *testing.T) {
	if runtime.GOOS == "windows" {
		// test fails on windows because dt.Location().String() returns
		// "local" instead of "Pacific/Samoa".
		// A possible fix would be to check the time offset instead of
		// the tz name.
		t.Skip("test not adapted for windows")
	}
	const locName = "Pacific/Samoa"
	loc, err := time.LoadLocation(locName)
	require.NoError(t, err)

	t.Setenv("TZ", locName)

	dt := DateTimeFlagValue{}
	err = dt.Set(DateTimeFormat)
	require.NoError(t, err)
	assert.Equal(t, loc.String(), dt.Location().String())
}

func TestDateTimeFlagWithTimezone(t *testing.T) {
	dt := DateTimeFlagValue{}
	err := dt.Set("2019.01.02-09:08:21-MUT")
	require.NoError(t, err)

	assert.Equal(t, "MUT", dt.Location().String())
}
