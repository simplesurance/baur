package docker

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func base64EncUserPasswd(user, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", user, password),
	))
}

func TestGetAuth(t *testing.T) {
	const gcrHost = "eu.gcr.io"
	const gcrUser = "guest"
	const gcrPasswd = "123"
	const defRegistryUser = "guest"
	const defRegistryPasswd = "1234"

	jsonAuthcfg := fmt.Sprintf(`{
	"auths": {
		"%s": {
			"auth": "%s"
		},
		"%s": {
			"auth": "%s"
		}
	}
}`,
		gcrHost,
		base64EncUserPasswd(gcrUser, gcrPasswd),

		DefaultRegistry,
		base64EncUserPasswd(defRegistryUser, defRegistryPasswd),
	)

	reader := bytes.NewBufferString(jsonAuthcfg)
	authCfg, err := docker.NewAuthConfigurations(reader)
	require.NoError(t, err)

	client := &Client{
		debugLogFn: func(string, ...interface{}) { return },
		auths:      authCfg,
	}

	auth := client.getAuth(gcrHost)
	require.NotNil(t, auth)
	assert.Equal(t, auth.ServerAddress, gcrHost)
	assert.Equal(t, auth.Password, gcrPasswd)
	assert.Equal(t, auth.Username, gcrUser)

	auth = client.getAuth(DefaultRegistry)
	require.NotNil(t, auth)
	assert.Equal(t, auth.ServerAddress, DefaultRegistry)
	assert.Equal(t, auth.Password, defRegistryPasswd)
	assert.Equal(t, auth.Username, defRegistryUser)

	// should return DefaultRegistry auth data
	auth = client.getAuth("")
	require.NotNil(t, auth)
	assert.Equal(t, auth.ServerAddress, DefaultRegistry)
	assert.Equal(t, auth.Password, defRegistryPasswd)
	assert.Equal(t, auth.Username, defRegistryUser)
}
