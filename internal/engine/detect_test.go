package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func envWith(pairs map[string]string) func(string) string {
	return func(k string) string { return pairs[k] }
}

func TestCandidatesOverrideWins(t *testing.T) {
	e := Environment{
		Override: "unix:///custom/docker.sock",
		Env:      envWith(map[string]string{"DOCKER_HOST": "tcp://1.2.3.4:2375"}),
	}
	got := Candidates(e)
	if len(got) != 1 || got[0] != "unix:///custom/docker.sock" {
		t.Fatalf("override should be the only candidate, got %v", got)
	}
}

func TestCandidatesEnvPrecedence(t *testing.T) {
	e := Environment{Env: envWith(map[string]string{
		"DOCKUPDATE_HOST": "tcp://10.0.0.1:2375",
		"DOCKER_HOST":     "tcp://10.0.0.2:2375",
	})}
	if got := Candidates(e); len(got) != 1 || got[0] != "tcp://10.0.0.1:2375" {
		t.Fatalf("DOCKUPDATE_HOST should win over DOCKER_HOST, got %v", got)
	}

	e = Environment{Env: envWith(map[string]string{"DOCKER_HOST": "tcp://10.0.0.2:2375"})}
	if got := Candidates(e); len(got) != 1 || got[0] != "tcp://10.0.0.2:2375" {
		t.Fatalf("DOCKER_HOST should be used, got %v", got)
	}
}

func TestCandidatesWellKnownOrder(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "podman"), 0o755); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(tmp, "podman", "podman-machine-default-api.sock")
	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	e := Environment{
		Env:          envWith(nil),
		HomeDir:      "/home/u",
		TmpDir:       tmp,
		UID:          501,
		XDGRuntime:   "/xdg",
		PodmanSocket: func() string { return "/podman/machine.sock" },
	}
	got := Candidates(e)
	want := []string{
		"unix:///var/run/docker.sock",
		"unix:///home/u/.docker/run/docker.sock",
		"unix:///podman/machine.sock",
		"unix://" + sock,
		"unix:///xdg/podman/podman.sock",
		"unix:///run/user/501/podman/podman.sock",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestCandidatesWithoutPodman(t *testing.T) {
	e := Environment{
		Env:          envWith(nil),
		HomeDir:      "/home/u",
		TmpDir:       t.TempDir(),
		UID:          1000,
		PodmanSocket: func() string { return "" },
	}
	got := Candidates(e)
	for _, c := range got {
		if c == "unix:///podman/machine.sock" {
			t.Fatalf("empty podman socket must not be a candidate: %v", got)
		}
	}
	if len(got) == 0 || got[0] != "unix:///var/run/docker.sock" {
		t.Fatalf("first fallback must be /var/run/docker.sock, got %v", got)
	}
}

func TestNormalize(t *testing.T) {
	if got := normalize("/a/b.sock"); got != "unix:///a/b.sock" {
		t.Fatalf("bare path: %q", got)
	}
	if got := normalize("tcp://h:2375"); got != "tcp://h:2375" {
		t.Fatalf("scheme kept: %q", got)
	}
	if got := normalize(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestContainerManagedAndUpdatable(t *testing.T) {
	c := Container{State: "running", Labels: map[string]string{"io.kubernetes.container.name": "app"}}
	if !c.Managed() || c.Updatable() {
		t.Fatalf("k8s-labeled container must be managed and not updatable")
	}
	c = Container{State: "running", Labels: map[string]string{}}
	if c.Managed() || !c.Updatable() {
		t.Fatalf("plain running container must be updatable")
	}
	c = Container{State: "exited", Labels: map[string]string{}}
	if c.Updatable() {
		t.Fatalf("stopped container must not be updatable")
	}
}

func TestComposeLabelAccessors(t *testing.T) {
	c := Container{Labels: map[string]string{
		"com.docker.compose.project":              "webapp",
		"com.docker.compose.service":              "web",
		"com.docker.compose.project.working_dir":  "/srv/webapp",
		"com.docker.compose.project.config_files": "/srv/webapp/compose.yaml",
	}}
	if c.Project() != "webapp" || c.Service() != "web" || c.ProjectDir() != "/srv/webapp" || c.ConfigFiles() != "/srv/webapp/compose.yaml" {
		t.Fatalf("compose label accessors broken: %+v", c)
	}
}
