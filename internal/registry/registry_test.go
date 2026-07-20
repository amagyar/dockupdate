package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
)

func digestOf(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("sha256:%x", sum)
}

// fakeRegistry serves a minimal OCI registry for one repo "app".
type fakeRegistry struct {
	t          *testing.T
	manifests  map[string][]byte // ref (tag or digest) -> manifest body
	mediaTypes map[string]string
}

func newFakeRegistry(t *testing.T) *fakeRegistry {
	return &fakeRegistry{t: t, manifests: map[string][]byte{}, mediaTypes: map[string]string{}}
}

func (f *fakeRegistry) addManifest(ref, mediaType string, body []byte) string {
	f.manifests[ref] = body
	f.mediaTypes[ref] = mediaType
	return digestOf(body)
}

func (f *fakeRegistry) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v2/app/manifests/", func(w http.ResponseWriter, r *http.Request) {
		ref := strings.TrimPrefix(r.URL.Path, "/v2/app/manifests/")
		body, ok := f.manifests[ref]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", f.mediaTypes[ref])
		w.Header().Set("Docker-Content-Digest", digestOf(body))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
	})
	return mux
}

const manifestV2 = "application/vnd.docker.distribution.manifest.v2+json"
const indexV1 = "application/vnd.oci.image.index.v1+json"
const ociManifestV1 = "application/vnd.oci.image.manifest.v1+json"

func testPlatform() ggcrv1.Platform {
	return ggcrv1.Platform{OS: "linux", Architecture: "arm64"}
}

func serverRef(t *testing.T, srv *httptest.Server, repo, tag string) string {
	host := strings.TrimPrefix(srv.URL, "http://")
	return fmt.Sprintf("%s/%s:%s", host, repo, tag)
}

func TestCheckUpToDate(t *testing.T) {
	fr := newFakeRegistry(t)
	body := []byte(`{"schemaVersion":2,"mediaType":"` + manifestV2 + `"}`)
	d := fr.addManifest("1.0", manifestV2, body)
	srv := httptest.NewServer(fr.handler())
	defer srv.Close()

	c := NewChecker(testPlatform(), true)
	res := c.Check(context.Background(), serverRef(t, srv, "app", "1.0"), []string{d})
	if res.Kind != KindUpToDate {
		t.Fatalf("got %v (%v), want up to date", res.Kind, res.Err)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	fr := newFakeRegistry(t)
	body := []byte(`{"schemaVersion":2,"mediaType":"` + manifestV2 + `"}`)
	remote := fr.addManifest("1.0", manifestV2, body)
	srv := httptest.NewServer(fr.handler())
	defer srv.Close()

	local := "sha256:" + strings.Repeat("0", 64)
	c := NewChecker(testPlatform(), true)
	res := c.Check(context.Background(), serverRef(t, srv, "app", "1.0"), []string{local})
	if res.Kind != KindUpdateAvailable {
		t.Fatalf("got %v (%v), want update available", res.Kind, res.Err)
	}
	if res.RemoteDigest != remote {
		t.Fatalf("RemoteDigest = %q, want %q", res.RemoteDigest, remote)
	}
}

func TestCheckMultiArchIndexMatchesPlatformManifest(t *testing.T) {
	fr := newFakeRegistry(t)

	// Child manifest for linux/arm64.
	configDigest := "sha256:" + strings.Repeat("1", 64)
	child := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":%q,"size":2},"layers":[]}`, ociManifestV1, configDigest))
	childDigest := fr.addManifest("", ociManifestV1, child) // placeholder, re-add below
	delete(fr.manifests, "")
	fr.manifests[childDigest] = child
	fr.mediaTypes[childDigest] = ociManifestV1

	// Index pointing at the child.
	index := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"manifests":[{"mediaType":%q,"digest":%q,"size":%d,"platform":{"os":"linux","architecture":"arm64"}}]}`, indexV1, ociManifestV1, childDigest, len(child)))
	indexDigest := fr.addManifest("latest", indexV1, index)

	srv := httptest.NewServer(fr.handler())
	defer srv.Close()
	ref := serverRef(t, srv, "app", "latest")
	c := NewChecker(testPlatform(), true)

	// Engines record either the index digest or the platform manifest
	// digest locally; both must count as up to date.
	for _, local := range []string{indexDigest, childDigest} {
		res := c.Check(context.Background(), ref, []string{local})
		if res.Kind != KindUpToDate {
			t.Fatalf("local %s: got %v (%v), want up to date", local, res.Kind, res.Err)
		}
	}
	if res := c.Check(context.Background(), ref, []string{"sha256:"+strings.Repeat("9", 64)}); res.Kind != KindUpdateAvailable {
		t.Fatalf("stale local digest: got %v, want update available", res.Kind)
	}
}

func TestCheckLocalBuild(t *testing.T) {
	c := NewChecker(testPlatform(), true)
	res := c.Check(context.Background(), "example.com/app:dev", nil)
	if res.Kind != KindLocalBuild {
		t.Fatalf("got %v, want local build", res.Kind)
	}
}

func TestCheckPinned(t *testing.T) {
	c := NewChecker(testPlatform(), true)
	res := c.Check(context.Background(), "nginx@sha256:"+strings.Repeat("a", 64), []string{"sha256:x"})
	if res.Kind != KindPinned {
		t.Fatalf("got %v, want pinned", res.Kind)
	}
}

func TestCheckFailedWhenRegistryDown(t *testing.T) {
	fr := newFakeRegistry(t) // no manifests registered -> 404
	srv := httptest.NewServer(fr.handler())
	defer srv.Close()

	c := NewChecker(testPlatform(), true)
	res := c.Check(context.Background(), serverRef(t, srv, "app", "missing"), []string{"sha256:"+strings.Repeat("2", 64)})
	if res.Kind != KindFailed || res.Err == nil {
		t.Fatalf("got %v (%v), want failed with error", res.Kind, res.Err)
	}
}

func TestIsPinned(t *testing.T) {
	if !IsPinned("nginx@sha256:abc") {
		t.Fatal("digest ref must be pinned")
	}
	if IsPinned("nginx:1.25") {
		t.Fatal("tag ref must not be pinned")
	}
}

func TestResultString(t *testing.T) {
	if (Result{Kind: KindUpdateAvailable}).String() != "update available" {
		t.Fatal("update available string")
	}
	if (Result{Kind: KindFailed, Err: fmt.Errorf("boom")}).String() != "check failed: boom" {
		t.Fatal("failed string with error")
	}
}

func TestRegistryMatches(t *testing.T) {
	for entry, reg := range map[string]string{
		"https://index.docker.io/v1/": "index.docker.io",
		"docker.io":                   "index.docker.io",
		"ghcr.io":                     "ghcr.io",
		"example.com:5000":            "example.com:5000",
	} {
		if !registryMatches(entry, reg) {
			t.Fatalf("%q should match %q", entry, reg)
		}
	}
	if registryMatches("evil.com", "example.com") {
		t.Fatal("mismatch must not match")
	}
}
