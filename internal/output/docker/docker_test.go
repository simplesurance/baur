package docker

import (
	"fmt"
	"net"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuth(t *testing.T) {
	const defRegistryUser = "guest"
	const defRegistryPasswd = "1234"

	const exampleHostname = "example.com"
	exampleURL := fmt.Sprintf("https://%s/v3", exampleHostname)
	const exampleUser = "thepresident"
	const examplePasswd = "abc"

	const myRegistryHostname = "myregistry.com"
	const myRegistryPort = "5000"
	myRegistryURL := fmt.Sprintf("https://%s", net.JoinHostPort(myRegistryHostname, myRegistryPort))
	const myRegistryUser = "hugo"
	const myRegistryPasswd = "hello"

	cfg := configfile.New("")
	cfg.AuthConfigs = map[string]types.AuthConfig{
		DefaultRegistry: {
			Username:      defRegistryUser,
			Password:      defRegistryPasswd,
			ServerAddress: DefaultRegistry,
		},
		exampleURL: {
			Username:      exampleUser,
			Password:      examplePasswd,
			ServerAddress: exampleURL,
		},
		myRegistryURL: {
			Username:      myRegistryUser,
			Password:      myRegistryPasswd,
			ServerAddress: myRegistryURL,
		},
	}

	client := &Client{
		cfg: cfg,
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

	// docker's own credential lookup (credentials.fileStore.Get(),
	// vendored from docker/cli) only matches a stored key's hostname
	// including its port - a bare hostname doesn't match a stored
	// "host:port" entry. Unlike the bespoke matching this package used to
	// do itself, this is exactly the same behavior the docker CLI and
	// credsStore-based lookups have, which is the point of delegating to
	// cfg.GetAuthConfig() instead of reimplementing it.
	t.Run("myregistry-host-without-port-not-found", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth := client.getAuth(myRegistryHostname)
		require.NotNil(t, auth)

		assert.Equal(t, myRegistryHostname, auth.ServerAddress)
		assert.Empty(t, auth.Username)
		assert.Empty(t, auth.Password)
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

func TestPushStreamLineError(t *testing.T) {
	t.Run("progress-message", func(t *testing.T) {
		err := pushStreamLineError(`{"status":"Preparing","progressDetail":{},"id":"63ca1fbb43ae"}`)
		assert.NoError(t, err)
	})

	// captured live from a real `docker push` to ECR without valid
	// credentials - the exact shape baur silently ignored before.
	t.Run("registry-error", func(t *testing.T) {
		err := pushStreamLineError(`{"error":"no basic auth credentials","errorDetail":{"message":"no basic auth credentials"}}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no basic auth credentials")
	})

	t.Run("blank-line", func(t *testing.T) {
		err := pushStreamLineError("\n")
		assert.NoError(t, err)
	})

	t.Run("not-json", func(t *testing.T) {
		err := pushStreamLineError("not json at all")
		assert.NoError(t, err)
	})
}
