//go:build live

// Live smoke tests against the real engine socket. Run with:
//
//	go test -tags live ./internal/engine/...
package engine_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/adev/dockupdate/internal/engine"
)

// TestLivePullStreamDebug dumps the raw pull progress messages the engine
// sends, to adapt the aggregator to the engine's dialect.
func TestLivePullStreamDebug(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	eng, _, err := engine.Connect(ctx, engine.DefaultEnvironment(""))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer eng.Close()

	cli, err := client.NewClientWithOpts(client.WithHost(eng.Socket()), client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer cli.Close()

	rc, err := cli.ImagePull(ctx, "docker.io/library/alpine:3.18", image.PullOptions{})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for i := 0; i < 200; i++ {
		var raw map[string]json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode: %v", err)
		}
		var msg jsonmessage.JSONMessage
		_ = json.Unmarshal([]byte(mustMarshal(raw)), &msg)
		t.Logf("id=%q status=%q progress=%+v raw=%s", msg.ID, msg.Status, msg.Progress, mustMarshal(raw))
	}
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}


func TestLiveConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, probed, err := engine.Connect(ctx, engine.DefaultEnvironment(""))
	if err != nil {
		t.Fatalf("connect failed (probed %v): %v", probed, err)
	}
	defer c.Close()

	info := c.Info()
	t.Logf("connected: %s %s via %s (%s/%s)", info.Kind, info.Version, info.Socket, info.OSType, info.Arch)
	if info.Kind == "" || info.Version == "" {
		t.Fatalf("engine not identified: %+v", info)
	}

	containers, err := c.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers: %v", err)
	}
	t.Logf("containers: %d", len(containers))
	for _, ct := range containers {
		t.Logf("  %s %s %s project=%q service=%q", ct.Name, ct.Image, ct.State, ct.Project(), ct.Service())
	}

	networks, err := c.ListNetworks(ctx)
	if err != nil {
		t.Fatalf("list networks: %v", err)
	}
	for _, n := range networks {
		t.Logf("  net %s driver=%s subnet=%s containers=%d", n.Name, n.Driver, n.Subnet, len(n.Containers))
	}
}
