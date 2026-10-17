package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/f1bonacc1/process-compose/src/config"
)

func TestGetLatestReleaseNameUserAgent(t *testing.T) {
	saved := config.Version
	config.Version = "v1.2.3"
	t.Cleanup(func() { config.Version = saved })

	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"name":"v1.2.4"}`))
	}))
	defer srv.Close()

	name, err := getLatestReleaseName(srv.URL)
	if err != nil {
		t.Fatalf("getLatestReleaseName() error = %v", err)
	}
	if name != "v1.2.4" {
		t.Errorf("getLatestReleaseName() = %q, want %q", name, "v1.2.4")
	}
	if want := "process-compose/v1.2.3"; gotUA != want {
		t.Errorf("User-Agent = %q, want %q", gotUA, want)
	}
}
