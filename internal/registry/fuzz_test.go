package registry

import (
	"strings"
	"testing"
)

func FuzzIsPinned(f *testing.F) {
	for _, seed := range []string{
		"",
		"nginx",
		"nginx:1.25",
		"nginx@sha256:" + strings.Repeat("a", 64),
		"@sha256:",
		"sha256:unanchored",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, ref string) {
		if got, want := IsPinned(ref), strings.Contains(ref, "@sha256:"); got != want {
			t.Fatalf("IsPinned(%q) = %v, want %v", ref, got, want)
		}
	})
}

func FuzzRegistryMatches(f *testing.F) {
	f.Add("https://index.docker.io/v1/", "index.docker.io")
	f.Add("docker.io", "index.docker.io")
	f.Add("ghcr.io", "ghcr.io")
	f.Add("evil.com", "example.com")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, entry, registry string) {
		// A match means the entry normalizes to the registry, so the
		// registry must also match itself.
		if registryMatches(entry, registry) && !registryMatches(registry, registry) {
			t.Fatalf("(%q, %q) matched but %q does not match itself", entry, registry, registry)
		}
	})
}
