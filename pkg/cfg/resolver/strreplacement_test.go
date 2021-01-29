package resolver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrReplacement(t *testing.T) {
	testcases := []struct {
		old    string
		new    string
		in     string
		result string
	}{
		{
			old:    "hello",
			new:    "bye",
			in:     "hello",
			result: "bye",
		},

		{
			old:    "hello",
			new:    "bye",
			in:     "hellobye",
			result: "byebye",
		},

		{
			old:    "hello",
			new:    "bye",
			in:     "byehellobye",
			result: "byebyebye",
		},

		{
			old: "hello",
			new: "bye",

			in:     "yo",
			result: "yo",
		},

		{
			old: "hello",
			new: "bye",

			in:     "",
			result: "",
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%q->%q", tc.in, tc.result), func(t *testing.T) {

			r := StrReplacement{Old: tc.old, New: tc.new}

			result, err := r.Resolve(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.result, result)
		})
	}
}
