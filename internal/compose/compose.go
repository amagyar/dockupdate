// Package compose detects and invokes the available compose provider
// (docker compose, podman compose, docker-compose, podman-compose).
package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Provider is a detected, working compose provider.
type Provider struct {
	Name   string   // human-readable, e.g. "podman compose"
	bin    string   // executable path
	prefix []string // e.g. ["compose"] for docker/podman, [] for *-compose
}

// candidate pairs an executable with the args that invoke compose.
var candidates = []struct {
	name   string
	bin    string
	prefix []string
}{
	{"docker compose", "docker", []string{"compose"}},
	{"podman compose", "podman", []string{"compose"}},
	{"docker-compose", "docker-compose", nil},
	{"podman-compose", "podman-compose", nil},
}

// execCommand is replaceable in tests.
var execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...) // #nosec G204 -- name is a provider binary from a fixed candidate list, resolved via LookPath
}

var lookPath = exec.LookPath

// Detect returns the first provider whose `version` probe succeeds.
func Detect() (*Provider, error) {
	for _, cand := range candidates {
		path, err := lookPath(cand.bin)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		args := append(append([]string{}, cand.prefix...), "version")
		err = execCommand(ctx, path, args...).Run()
		cancel()
		if err == nil {
			return &Provider{Name: cand.name, bin: path, prefix: cand.prefix}, nil
		}
	}
	return nil, errors.New("no compose provider found (looked for: docker compose, podman compose, docker-compose, podman-compose)")
}

// UpRecreate recreates a compose service: up -d --force-recreate <service>.
// Project metadata comes from the container's compose labels. The project
// directory is applied as the command's working directory (the common
// denominator across all four providers; --project-directory is not
// supported by podman-compose). Stderr is captured for the failure state.
func (p *Provider) UpRecreate(ctx context.Context, project, dir string, configFiles []string, service string) error {
	args := append([]string{}, p.prefix...)
	args = append(args, "-p", project)
	for _, f := range configFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "up", "-d", "--force-recreate", service)

	cmd := execCommand(ctx, p.bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s up failed: %s", p.Name, msg)
	}
	return nil
}

// SplitConfigFiles splits the com.docker.compose.project.config_files label
// value into individual file paths.
func SplitConfigFiles(label string) []string {
	var out []string
	for _, f := range strings.Split(label, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
