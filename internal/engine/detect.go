package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment carries everything socket candidate resolution needs, so the
// resolution logic stays pure and unit-testable.
type Environment struct {
	Override string // --socket flag, wins over everything

	Env        func(string) string // environment lookup (os.Getenv)
	HomeDir    string
	TmpDir     string
	UID        int
	XDGRuntime string

	// PodmanSocket resolves the Podman machine API socket
	// (e.g. via `podman machine inspect`). May return "".
	PodmanSocket func() string
}

func envOr(get func(string) string, key string) string {
	if get == nil {
		return ""
	}
	return get(key)
}

// normalize turns a bare path into a unix:// host, leaving existing schemes
// (unix://, tcp://, npipe://, ssh://) untouched.
func normalize(host string) string {
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		return host
	}
	return "unix://" + host
}

// Candidates returns the probe-ordered list of engine hosts. Explicit
// configuration (flag or env vars) is authoritative: when set, it is the
// only candidate. Otherwise the well-known socket locations are probed in
// order.
func Candidates(e Environment) []string {
	if v := e.Override; v != "" {
		return []string{normalize(v)}
	}
	if v := envOr(e.Env, "DOCKUPDATE_HOST"); v != "" {
		return []string{normalize(v)}
	}
	if v := envOr(e.Env, "DOCKER_HOST"); v != "" {
		return []string{normalize(v)}
	}

	var out []string
	seen := map[string]bool{}
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, normalize(path))
	}

	add("/var/run/docker.sock")
	if e.HomeDir != "" {
		add(filepath.Join(e.HomeDir, ".docker", "run", "docker.sock"))
	}
	if e.PodmanSocket != nil {
		add(e.PodmanSocket())
	}
	if e.TmpDir != "" {
		matches, _ := filepath.Glob(filepath.Join(e.TmpDir, "podman", "*-api.sock"))
		for _, m := range matches {
			add(m)
		}
	}
	if e.XDGRuntime != "" {
		add(filepath.Join(e.XDGRuntime, "podman", "podman.sock"))
	}
	if e.UID >= 0 {
		add(fmt.Sprintf("/run/user/%d/podman/podman.sock", e.UID))
	}
	return out
}

// DefaultEnvironment builds an Environment from the current process.
func DefaultEnvironment(override string) Environment {
	home, _ := os.UserHomeDir()
	return Environment{
		Override:     override,
		Env:          os.Getenv,
		HomeDir:      home,
		TmpDir:       os.TempDir(),
		UID:          os.Getuid(),
		XDGRuntime:   os.Getenv("XDG_RUNTIME_DIR"),
		PodmanSocket: podmanMachineSocket,
	}
}
