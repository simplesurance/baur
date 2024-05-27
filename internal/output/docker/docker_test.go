package docker

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
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
	const defRegistryUser = "guest"
	const defRegistryPasswd = "1234"

	const exampleHostname = "example.com"
	exampleURL := fmt.Sprintf("https://%s/v3", exampleHostname)
	const exampleUser = "thepresident"
	const examplePasswd = "abc"

	const myRegistryHostname = "myregistry.com"
	const myRegistryPort = dockerRegistryDefaultPort
	myRegistryURL := fmt.Sprintf("https://%s", net.JoinHostPort(myRegistryHostname, myRegistryPort))
	const myRegistryUser = "hugo"
	const myRegistryPasswd = "hello"

	jsonAuthcfg := fmt.Sprintf(`{
	"auths": {
		"%s": {
			"auth": "%s"
		},
		"%s": {
			"auth": "%s"
		},
		"%s": {
			"auth": "%s"
		}
	}
}`,
		DefaultRegistry,
		base64EncUserPasswd(defRegistryUser, defRegistryPasswd),

		exampleURL,
		base64EncUserPasswd(exampleUser, examplePasswd),

		myRegistryURL,
		base64EncUserPasswd(myRegistryUser, myRegistryPasswd),
	)

	reader := bytes.NewBufferString(jsonAuthcfg)
	authCfg, err := docker.NewAuthConfigurations(reader)
	require.NoError(t, err)

	client := &Client{
		auths: authCfg,
	}

	t.Run("default-url", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(DefaultRegistry)
		require.NotNil(t, auth)

		assert.Equal(t, DefaultRegistry, auth.ServerAddress)
		assert.Equal(t, defRegistryPasswd, auth.Password)
		assert.Equal(t, defRegistryUser, auth.Username)
	})

	t.Run("example-url", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(exampleURL)
		require.NotNil(t, auth)

		assert.Equal(t, exampleURL, auth.ServerAddress)
		assert.Equal(t, examplePasswd, auth.Password)
		assert.Equal(t, exampleUser, auth.Username)
	})

	// when a URL is used as server and the image is tagged without url
	// (e.g. gcr.eu.io/image:tag, getAuth() only gets the hostname as arg
	t.Run("example-hostname", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(exampleHostname)
		require.NotNil(t, auth)

		assert.Equal(t, exampleURL, auth.ServerAddress)
		assert.Equal(t, examplePasswd, auth.Password)
		assert.Equal(t, exampleUser, auth.Username)
	})

	t.Run("no-server-panic", func(t *testing.T) {
		client.debugLogFn = t.Logf
		require.Panics(t, func() { client.getAuth("") })
	})

	t.Run("myregistry-host", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(myRegistryHostname)
		require.NotNil(t, auth)

		assert.Equal(t, myRegistryURL, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})

	t.Run("myregistry-hostport", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(fmt.Sprintf("%s:%s", myRegistryHostname, myRegistryPort))
		require.NotNil(t, auth)

		assert.Equal(t, myRegistryURL, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})

	t.Run("myregistry-url", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(myRegistryURL)
		require.NotNil(t, auth)

		assert.Equal(t, myRegistryURL, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})
}
