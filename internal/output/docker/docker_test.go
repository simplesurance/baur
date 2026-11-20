package docker

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
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

		auth, err := client.getAuth(DefaultRegistry)
		require.NoError(t, err)

		assert.Equal(t, DefaultRegistry, auth.ServerAddress)
		assert.Equal(t, defRegistryPasswd, auth.Password)
		assert.Equal(t, defRegistryUser, auth.Username)
	})

	t.Run("example-url", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth(exampleURL)
		require.NoError(t, err)

		assert.Equal(t, exampleURL, auth.ServerAddress)
		assert.Equal(t, examplePasswd, auth.Password)
		assert.Equal(t, exampleUser, auth.Username)
	})

	// when a URL is used as server and the image is tagged without url
	// (e.g. gcr.eu.io/image:tag, getAuth() only gets the hostname as arg
	t.Run("example-hostname", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth(exampleHostname)
		require.NoError(t, err)

		assert.Equal(t, exampleURL, auth.ServerAddress)
		assert.Equal(t, examplePasswd, auth.Password)
		assert.Equal(t, exampleUser, auth.Username)
	})

	t.Run("no-server-panic", func(t *testing.T) {
		client.debugLogFn = t.Logf
		require.Panics(t, func() { _, _ = client.getAuth("") })
	})

	// docker's own credential lookup does not match a bare hostname to a
	// stored "host:5000" entry, the fallback to the literal auths entries
	// does, with the default registry port.
	t.Run("myregistry-host-without-port", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth(myRegistryHostname)
		require.NoError(t, err)

		assert.Equal(t, myRegistryHostname, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})

	t.Run("example-hostname-with-default-port", func(t *testing.T) {
		client.debugLogFn = t.Logf

		server := net.JoinHostPort(exampleHostname, myRegistryPort)
		auth, err := client.getAuth(server)
		require.NoError(t, err)

		assert.Equal(t, server, auth.ServerAddress)
		assert.Equal(t, examplePasswd, auth.Password)
		assert.Equal(t, exampleUser, auth.Username)
	})

	t.Run("unknown-registry", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth("unknown.com")
		require.NoError(t, err)

		assert.Equal(t, "unknown.com", auth.ServerAddress)
		assert.Empty(t, auth.Username)
		assert.Empty(t, auth.Password)
	})

	t.Run("myregistry-hostport", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth(fmt.Sprintf("%s:%s", myRegistryHostname, myRegistryPort))
		require.NoError(t, err)

		assert.Equal(t, myRegistryURL, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})

	t.Run("myregistry-url", func(t *testing.T) {
		client.debugLogFn = t.Logf

		auth, err := client.getAuth(myRegistryURL)
		require.NoError(t, err)

		assert.Equal(t, myRegistryURL, auth.ServerAddress)
		assert.Equal(t, myRegistryPasswd, auth.Password)
		assert.Equal(t, myRegistryUser, auth.Username)
	})
}

const fakeHelperErr = "helper exploded"

// configWithFailingHelper returns a config with a credsStore whose helper
// always fails.
func configWithFailingHelper(t *testing.T) *configfile.ConfigFile {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake credential helper is a shell script")
	}

	helperDir := t.TempDir()
	helper := "#!/bin/sh\necho '" + fakeHelperErr + "'\nexit 1\n"
	err := os.WriteFile(filepath.Join(helperDir, "docker-credential-baurtest"), []byte(helper), 0o755)
	require.NoError(t, err)
	t.Setenv("PATH", helperDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := configfile.New("")
	cfg.CredentialsStore = "baurtest"
	return cfg
}

func TestGetAuthCredentialHelperFails(t *testing.T) {
	t.Run("no-auths-entry", func(t *testing.T) {
		client := &Client{cfg: configWithFailingHelper(t), debugLogFn: t.Logf}

		auth, err := client.getAuth("myregistry.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), fakeHelperErr)

		assert.Equal(t, "myregistry.com", auth.ServerAddress)
		assert.Empty(t, auth.Username)
		assert.Empty(t, auth.Password)
	})

	t.Run("auths-entry", func(t *testing.T) {
		cfg := configWithFailingHelper(t)
		cfg.AuthConfigs = map[string]types.AuthConfig{
			"registry.local:5000": {Username: "u", Password: "p", ServerAddress: "registry.local:5000"},
		}
		client := &Client{cfg: cfg, debugLogFn: t.Logf}

		auth, err := client.getAuth("registry.local:5000")
		require.Error(t, err)
		assert.Contains(t, err.Error(), fakeHelperErr)

		assert.Equal(t, "registry.local:5000", auth.ServerAddress)
		assert.Equal(t, "u", auth.Username)
		assert.Equal(t, "p", auth.Password)
	})
}

func TestGetAuthDockerAuthConfigEnv(t *testing.T) {
	t.Setenv(
		"DOCKER_AUTH_CONFIG",
		`{"auths":{"envregistry.com":{"auth":"`+base64.StdEncoding.EncodeToString([]byte("envuser:envpasswd"))+`"}}}`,
	)

	client := &Client{cfg: configfile.New(""), debugLogFn: t.Logf}

	auth, err := client.getAuth("envregistry.com")
	require.NoError(t, err)

	assert.Equal(t, "envuser", auth.Username)
	assert.Equal(t, "envpasswd", auth.Password)
}

func TestPushStreamError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := pushStreamError(strings.NewReader(
			`{"status":"Pushing","id":"63ca1fbb43ae"}`+"\r\n"+
				`{"status":"latest: digest: sha256:abc size: 528"}`+"\r\n",
		), t.Logf)
		assert.NoError(t, err)
	})

	t.Run("error-last-line-with-newline", func(t *testing.T) {
		err := pushStreamError(strings.NewReader(
			`{"status":"Preparing","id":"63ca1fbb43ae"}`+"\r\n"+
				`{"errorDetail":{"message":"denied"}}`+"\r\n",
		), t.Logf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("error-last-line-without-newline", func(t *testing.T) {
		err := pushStreamError(strings.NewReader(
			`{"status":"Preparing","id":"63ca1fbb43ae"}`+"\r\n"+
				`{"errorDetail":{"message":"denied"}}`,
		), t.Logf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("only-line-without-newline", func(t *testing.T) {
		err := pushStreamError(strings.NewReader(`{"error":"denied"}`), t.Logf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("read-error", func(t *testing.T) {
		err := pushStreamError(iotest.ErrReader(errors.New("connection reset")), t.Logf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection reset")
	})

	t.Run("lines-are-not-a-format-string", func(t *testing.T) {
		var logged []string
		logFn := func(format string, args ...any) {
			logged = append(logged, fmt.Sprintf(format, args...))
		}

		err := pushStreamError(strings.NewReader(`{"errorDetail":{"message":"quota 100% exceeded"}}`), logFn)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "quota 100% exceeded")

		require.Len(t, logged, 1)
		assert.Contains(t, logged[0], "quota 100% exceeded")
		assert.NotContains(t, logged[0], "MISSING")
	})
}

// fakeDaemon is a minimal docker daemon API server. It accepts every tag
// request and answers push requests with pushResponse.
type fakeDaemon struct {
	pushResponse string

	taggedRepo string
	taggedTag  string
	pushedRepo string
	pushAuth   registry.AuthConfig
}

func startFakeDaemon(t *testing.T, pushResponse string) (*fakeDaemon, *dockerclient.Client) {
	t.Helper()

	d := &fakeDaemon{pushResponse: pushResponse}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/tag"):
			d.taggedRepo = r.URL.Query().Get("repo")
			d.taggedTag = r.URL.Query().Get("tag")
			w.WriteHeader(http.StatusCreated)

		case strings.HasSuffix(r.URL.Path, "/push"):
			_, name, _ := strings.Cut(r.URL.Path, "/images/")
			d.pushedRepo = strings.TrimSuffix(name, "/push")

			authJSON, err := base64.URLEncoding.DecodeString(r.Header.Get(registry.AuthHeader))
			if err == nil {
				_ = json.Unmarshal(authJSON, &d.pushAuth)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, d.pushResponse)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	clt, err := dockerclient.NewClientWithOpts(
		dockerclient.WithHost("tcp://"+srv.Listener.Addr().String()),
		dockerclient.WithVersion("1.47"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clt.Close() })

	return d, clt
}

const pushSuccessResponse = `{"status":"Pushing","id":"63ca1fbb43ae"}` + "\r\n" +
	`{"status":"1.0: digest: sha256:abc size: 528"}` + "\r\n"

func TestUpload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d, clt := startFakeDaemon(t, pushSuccessResponse)
		client := &Client{clt: clt, cfg: configfile.New(""), debugLogFn: t.Logf}

		uri, err := client.Upload("sha256:1234", "registry.local:5000", "team/app", "1.0")
		require.NoError(t, err)

		assert.Equal(t, "registry.local:5000/team/app:1.0", uri)
		assert.Equal(t, "registry.local:5000/team/app", d.taggedRepo)
		assert.Equal(t, "1.0", d.taggedTag)
		assert.Equal(t, "registry.local:5000/team/app", d.pushedRepo)
	})

	t.Run("push-error-without-newline", func(t *testing.T) {
		_, clt := startFakeDaemon(t, `{"errorDetail":{"message":"no basic auth credentials"}}`)
		client := &Client{clt: clt, cfg: configfile.New(""), debugLogFn: t.Logf}

		_, err := client.Upload("sha256:1234", "registry.local:5000", "team/app", "1.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no basic auth credentials")
	})

	t.Run("no-registry-pushes-to-docker-hub", func(t *testing.T) {
		d, clt := startFakeDaemon(t, pushSuccessResponse)
		cfg := configfile.New("")
		cfg.AuthConfigs = map[string]types.AuthConfig{
			DefaultRegistry: {Username: "hubuser", Password: "hubpasswd", ServerAddress: DefaultRegistry},
		}
		client := &Client{clt: clt, cfg: cfg, debugLogFn: t.Logf}

		uri, err := client.Upload("sha256:1234", "", "team/app", "1.0")
		require.NoError(t, err)

		assert.Equal(t, "team/app:1.0", uri)
		assert.Equal(t, "docker.io/team/app", d.taggedRepo)
		assert.Equal(t, "docker.io/team/app", d.pushedRepo)
		assert.Equal(t, "hubuser", d.pushAuth.Username)
		assert.Equal(t, "hubpasswd", d.pushAuth.Password)
	})

	// the daemon ignores credentials whose ServerAddress host is not the
	// registry host
	t.Run("auths-key-without-port-sends-registry-as-server-address", func(t *testing.T) {
		d, clt := startFakeDaemon(t, pushSuccessResponse)
		cfg := configfile.New("")
		cfg.AuthConfigs = map[string]types.AuthConfig{
			"https://registry.local": {Username: "u", Password: "p", ServerAddress: "https://registry.local"},
		}
		client := &Client{clt: clt, cfg: cfg, debugLogFn: t.Logf}

		_, err := client.Upload("sha256:1234", "registry.local:5000", "team/app", "1.0")
		require.NoError(t, err)

		assert.Equal(t, "registry.local:5000", d.pushAuth.ServerAddress)
		assert.Equal(t, "u", d.pushAuth.Username)
		assert.Equal(t, "p", d.pushAuth.Password)
	})

	t.Run("failing-helper-push-without-auth-succeeds", func(t *testing.T) {
		d, clt := startFakeDaemon(t, pushSuccessResponse)
		client := &Client{clt: clt, cfg: configWithFailingHelper(t), debugLogFn: t.Logf}

		_, err := client.Upload("sha256:1234", "registry.local:5000", "team/app", "1.0")
		require.NoError(t, err)
		assert.Empty(t, d.pushAuth.Username)
	})

	t.Run("failing-helper-push-error-contains-helper-error", func(t *testing.T) {
		_, clt := startFakeDaemon(t, `{"errorDetail":{"message":"no basic auth credentials"}}`)
		client := &Client{clt: clt, cfg: configWithFailingHelper(t), debugLogFn: t.Logf}

		_, err := client.Upload("sha256:1234", "registry.local:5000", "team/app", "1.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no basic auth credentials")
		assert.Contains(t, err.Error(), fakeHelperErr)
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

	// newer Docker API types only define "errorDetail", "error" is deprecated
	t.Run("registry-error-detail-only", func(t *testing.T) {
		err := pushStreamLineError(`{"errorDetail":{"message":"denied: requested access to the resource is denied"}}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied: requested access to the resource is denied")
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
