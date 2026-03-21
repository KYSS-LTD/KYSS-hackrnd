package lobby

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewDockerClientUsesDaemonAPIVersion(t *testing.T) {
	t.Setenv("DOCKER_API_VERSION", "9.99")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_ping" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Api-Version", "1.44")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("DOCKER_HOST", srv.URL)
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")

	cli, err := newDockerClient(context.Background())
	if err != nil {
		t.Fatalf("newDockerClient() error = %v", err)
	}
	defer cli.Close()

	if got := cli.ClientVersion(); got != "1.44" {
		t.Fatalf("ClientVersion() = %q, want %q", got, "1.44")
	}
}

func TestNewDockerClientFallsBackWithoutAPIVersionHeader(t *testing.T) {
	t.Setenv("DOCKER_API_VERSION", "9.99")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_ping" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("DOCKER_HOST", srv.URL)
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")

	cli, err := newDockerClient(context.Background())
	if err != nil {
		t.Fatalf("newDockerClient() error = %v", err)
	}
	defer cli.Close()

	if got := cli.ClientVersion(); got == os.Getenv("DOCKER_API_VERSION") {
		t.Fatalf("ClientVersion() = %q, should not reuse DOCKER_API_VERSION", got)
	}
}
