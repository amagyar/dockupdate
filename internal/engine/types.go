package engine

// Info describes the connected engine.
type Info struct {
	Kind    string // "Docker", "Podman" or "Docker-compatible"
	Version string
	OSType  string
	Arch    string
	Socket  string
}

// Container is the inventory record for a single container.
type Container struct {
	ID      string
	Name    string
	Image   string // image reference the container was created from
	ImageID string
	State   string // running, exited, ...
	Labels  map[string]string
}

// Compose label accessors. Docker Compose records com.docker.compose.*;
// podman-compose uses io.podman.compose.* instead, so both are checked.

func (c Container) label(keys ...string) string {
	for _, k := range keys {
		if v := c.Labels[k]; v != "" {
			return v
		}
	}
	return ""
}

func (c Container) Project() string {
	return c.label("com.docker.compose.project", "io.podman.compose.project")
}

func (c Container) Service() string {
	return c.label("com.docker.compose.service", "io.podman.compose.service")
}

func (c Container) ProjectDir() string {
	return c.label("com.docker.compose.project.working_dir", "io.podman.compose.project.working_dir")
}

func (c Container) ConfigFiles() string {
	return c.label("com.docker.compose.project.config_files", "io.podman.compose.config")
}

// Managed reports whether the container is managed by an external
// orchestrator (Kubernetes, Docker Swarm or a Podman pod) and therefore
// must not be updated by dockupdate.
func (c Container) Managed() bool {
	for _, k := range []string{
		"io.kubernetes.container.name",
		"com.docker.swarm.task.id",
		"com.docker.swarm.service.id",
		"io.podman.pod.name",
	} {
		if _, ok := c.Labels[k]; ok {
			return true
		}
	}
	return false
}

// Updatable reports whether the container is eligible for update checks:
// running and not managed by an external orchestrator.
func (c Container) Updatable() bool {
	return c.State == "running" && !c.Managed()
}

// NetworkContainer is a container attached to a network.
type NetworkContainer struct {
	ID      string
	Name    string
	IPv4    string
	Project string // compose project or "" for standalone
}

// Network is the inventory record for a network.
type Network struct {
	ID         string
	Name       string
	Driver     string
	Subnet     string
	Containers []NetworkContainer
}
