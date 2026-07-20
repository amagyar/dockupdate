package engine

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// podmanMachineSocket asks the podman CLI for the machine's local API
// socket path (Docker-compatible). Returns "" when podman is unavailable.
func podmanMachineSocket() string {
	path, err := exec.LookPath("podman")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// #nosec G204 -- path comes from exec.LookPath("podman"), a fixed binary name.
	out, err := exec.CommandContext(ctx, path, "machine", "inspect",
		"--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
