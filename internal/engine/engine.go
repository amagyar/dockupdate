package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

// Client wraps the Docker Engine API client. It works against any
// Docker-compatible API socket, including Podman's.
type Client struct {
	cli    *client.Client
	socket string
	info   Info
}

// PingFunc probes a candidate host; it returns an error when unreachable.
// Replaceable in tests.
var PingFunc = func(ctx context.Context, host string) error {
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.Ping(ctx)
	return err
}

// Connect probes the candidate sockets in order and connects to the first
// one that answers. It returns the probed hosts in order on failure.
func Connect(ctx context.Context, e Environment) (*Client, []string, error) {
	candidates := Candidates(e)
	for _, host := range candidates {
		pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := PingFunc(pctx, host)
		cancel()
		if err != nil {
			continue
		}
		cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
		if err != nil {
			continue
		}
		c := &Client{cli: cli, socket: host}
		if err := c.loadInfo(ctx); err != nil {
			cli.Close()
			return nil, candidates, err
		}
		return c, candidates, nil
	}
	return nil, candidates, errors.New("no reachable engine socket")
}

// Socket returns the host the client connected to.
func (c *Client) Socket() string { return c.socket }

// Info returns the identified engine information.
func (c *Client) Info() Info { return c.info }

// Close releases the underlying client.
func (c *Client) Close() error { return c.cli.Close() }

func (c *Client) loadInfo(ctx context.Context) error {
	ver, err := c.cli.ServerVersion(ctx)
	if err != nil {
		return fmt.Errorf("query engine version: %w", err)
	}
	sys, sysErr := c.cli.Info(ctx)

	c.info = Info{Kind: "Docker", Version: ver.Version, Socket: c.socket}
	for _, comp := range ver.Components {
		if strings.Contains(strings.ToLower(comp.Name), "podman") {
			c.info.Kind = "Podman"
			c.info.Version = comp.Version
			break
		}
	}
	if sysErr == nil {
		c.info.OSType = sys.OSType
		c.info.Arch = sys.Architecture
	}
	return nil
}

// ListContainers returns all containers (running and stopped).
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(list))
	for _, s := range list {
		name := strings.TrimPrefix(strings.Join(s.Names, ", "), "/")
		out = append(out, Container{
			ID:      s.ID,
			Name:    name,
			Image:   s.Image,
			ImageID: s.ImageID,
			State:   s.State,
			Labels:  s.Labels,
		})
	}
	return out, nil
}

// ListNetworks returns all networks with their attached containers.
func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	list, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Network, 0, len(list))
	for _, s := range list {
		n := Network{ID: s.ID, Name: s.Name, Driver: s.Driver}
		ins, err := c.cli.NetworkInspect(ctx, s.ID, network.InspectOptions{})
		if err == nil {
			for _, cfg := range ins.IPAM.Config {
				if cfg.Subnet != "" {
					n.Subnet = cfg.Subnet
					break
				}
			}
			for id, ep := range ins.Containers {
				n.Containers = append(n.Containers, NetworkContainer{ID: id, Name: ep.Name, IPv4: ep.IPv4Address})
			}
		}
		out = append(out, n)
	}
	return out, nil
}

// ListContainersWithProjects attaches compose project info to network
// containers. Kept separate because network endpoints don't carry labels.
func (c *Client) FillNetworkProjects(ctx context.Context, nets []Network) error {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return err
	}
	projectOf := map[string]string{}
	for _, ct := range containers {
		projectOf[ct.Name] = ct.Project()
	}
	for i := range nets {
		for j := range nets[i].Containers {
			nets[i].Containers[j].Project = projectOf[nets[i].Containers[j].Name]
		}
	}
	return nil
}

// RepoDigests returns the repo digests recorded for an image.
func (c *Client) RepoDigests(ctx context.Context, imageRef string) ([]string, error) {
	ins, err := c.cli.ImageInspect(ctx, imageRef)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, rd := range ins.RepoDigests {
		if i := strings.LastIndex(rd, "@"); i >= 0 {
			out = append(out, rd[i+1:])
		}
	}
	return out, nil
}

// Progress is an aggregate pull progress snapshot.
type Progress struct {
	Current     int64 // downloaded bytes (0 when the engine reports none)
	Total       int64 // total bytes (0 when the engine reports none)
	LayersDone  int   // layers reported complete
	LayersTotal int   // layers seen so far
}

// PullImage pulls ref, streaming aggregate progress to the callback.
// auth is the base64-encoded X-Registry-Auth header ("" for anonymous).
// Engines that report no byte progress (Podman) still report per-layer
// status transitions, exposed via the Layers fields.
func (c *Client) PullImage(ctx context.Context, ref string, auth string, progress func(Progress)) error {
	rc, err := c.cli.ImagePull(ctx, ref, image.PullOptions{RegistryAuth: auth})
	if err != nil {
		return err
	}
	defer rc.Close()

	agg := NewLayerAggregator()
	dec := json.NewDecoder(rc)
	for {
		var msg jsonmessage.JSONMessage
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading pull stream: %w", err)
		}
		if msg.Error != nil {
			return msg.Error
		}
		var cur, tot int64
		if msg.Progress != nil {
			cur, tot = msg.Progress.Current, msg.Progress.Total
		}
		agg.Update(msg.ID, msg.Status, cur, tot)
		if progress != nil {
			cb, tb := agg.Totals()
			ld, lt := agg.Layers()
			progress(Progress{Current: cb, Total: tb, LayersDone: ld, LayersTotal: lt})
		}
	}
}

// LayerAggregator folds per-layer pull events into overall byte totals.
type LayerAggregator struct {
	layers map[string]*layerState
	order  []string
}

type layerState struct {
	current int64
	total   int64
	done    bool
}

// NewLayerAggregator creates an empty aggregator.
func NewLayerAggregator() *LayerAggregator {
	return &LayerAggregator{layers: map[string]*layerState{}}
}

// Update records one pull event for a layer. Statuses that signal
// completion mark the layer done; progressDetail bytes accumulate.
func (a *LayerAggregator) Update(id, status string, current, total int64) {
	if id == "" {
		return
	}
	st, ok := a.layers[id]
	if !ok {
		st = &layerState{}
		a.layers[id] = st
		a.order = append(a.order, id)
	}
	if total > 0 {
		st.total = total
	}
	if current > st.current {
		st.current = current
	}
	switch status {
	case "Already exists", "Download complete", "Pull complete", "Skipped", "Already have":
		st.done = true
		if st.total > 0 {
			st.current = st.total
		}
	}
}

// Totals returns aggregate (current, total) bytes across known layers.
func (a *LayerAggregator) Totals() (int64, int64) {
	var cur, tot int64
	for _, id := range a.order {
		st := a.layers[id]
		cur += st.current
		tot += st.total
	}
	return cur, tot
}

// Layers returns (done, total) layer counts.
func (a *LayerAggregator) Layers() (int, int) {
	done := 0
	for _, id := range a.order {
		if a.layers[id].done {
			done++
		}
	}
	return done, len(a.order)
}

// InspectContainer returns the full inspect record for a container.
func (c *Client) InspectContainer(ctx context.Context, id string) (container.InspectResponse, error) {
	return c.cli.ContainerInspect(ctx, id)
}

// RecreateContainer replaces a standalone container with a new one from the
// new image, preserving its config, host config, networks and name. The old
// container is kept (stopped, renamed) when creation fails.
func (c *Client) RecreateContainer(ctx context.Context, id string, newImage string) error {
	old, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}

	name := strings.TrimPrefix(old.Name, "/")
	backupName := fmt.Sprintf("%s-old-%d", name, time.Now().Unix())

	timeout := 10
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	if err := c.cli.ContainerRename(ctx, id, backupName); err != nil {
		return fmt.Errorf("rename old container: %w", err)
	}

	// Primary network goes into the create call; the rest are connected
	// after start. Endpoint settings are left empty so the engine
	// assigns fresh addresses.
	netConf := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
	var extraNetworks []string
	if old.NetworkSettings != nil {
		first := true
		for netName := range old.NetworkSettings.Networks {
			if first {
				netConf.EndpointsConfig[netName] = &network.EndpointSettings{}
				first = false
				continue
			}
			extraNetworks = append(extraNetworks, netName)
		}
	}

	conf := *old.Config
	conf.Image = newImage
	created, err := c.cli.ContainerCreate(ctx, &conf, old.HostConfig, netConf, nil, name)
	if err != nil {
		// Best-effort restore of the old container's name.
		_ = c.cli.ContainerRename(ctx, id, name)
		return fmt.Errorf("create: %w", err)
	}
	if err := c.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		_ = c.cli.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true})
		_ = c.cli.ContainerRename(ctx, id, name)
		return fmt.Errorf("start: %w", err)
	}
	for _, netName := range extraNetworks {
		if err := c.cli.NetworkConnect(ctx, netName, created.ID, &network.EndpointSettings{}); err != nil {
			return fmt.Errorf("connect network %s: %w", netName, err)
		}
	}
	// Old container stays present (stopped) for rollback; remove it only
	// once the replacement is healthy enough to keep running.
	if err := c.cli.ContainerRemove(ctx, id, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("remove old container: %w", err)
	}
	return nil
}

// ImageID returns the image ID a container is currently using.
func (c *Client) ImageID(ctx context.Context, containerID string) (string, error) {
	ins, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return ins.Image, nil
}

// ImageIDForRef returns the ID of the image a reference resolves to.
func (c *Client) ImageIDForRef(ctx context.Context, imageRef string) (string, error) {
	ins, err := c.cli.ImageInspect(ctx, imageRef)
	if err != nil {
		return "", err
	}
	return ins.ID, nil
}

// RemoveImage deletes an image by ID (best effort, used by --prune).
func (c *Client) RemoveImage(ctx context.Context, imageID string) error {
	_, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{})
	return err
}
