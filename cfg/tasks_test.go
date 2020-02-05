package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppsWithoutAnyTasksAreValid(t *testing.T) {
	app := App{Name: "testapp"}

	err := app.Validate()
	assert.NoError(t, err)
}
