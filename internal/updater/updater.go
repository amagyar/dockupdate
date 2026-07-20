// Package updater executes container updates: pull with progress, digest
// verification, then restart (compose service or standalone recreation).
// Each task progresses independently through a small state machine.
package updater

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/adev/dockupdate/internal/engine"
)

// Phase is a task's position in the update pipeline.
type Phase int

const (
	PhasePending Phase = iota
	PhasePulling
	PhaseVerifying
	PhaseRestarting
	PhaseDone
	PhaseFailed
)

func (p Phase) String() string {
	switch p {
	case PhasePending:
		return "pending"
	case PhasePulling:
		return "pulling"
	case PhaseVerifying:
		return "verifying checksum"
	case PhaseRestarting:
		return "restarting service"
	case PhaseDone:
		return "success"
	case PhaseFailed:
		return "failed"
	}
	return "unknown"
}

// EventKind identifies what an Event reports.
type EventKind int

const (
	EventPhaseChange EventKind = iota // task moved to a new phase
	EventPullProgress                 // byte progress while pulling
	EventDone                         // finished successfully
	EventFailed                       // finished with error
)

// Event is emitted by workers; the TUI applies events to its row state.
type Event struct {
	TaskID      string
	Kind        EventKind
	Phase       Phase // for EventPhaseChange
	Current     int64 // for EventPullProgress (bytes, 0 if engine reports none)
	Total       int64 // for EventPullProgress (bytes, 0 if engine reports none)
	LayersDone  int   // for EventPullProgress
	LayersTotal int   // for EventPullProgress
	Err         error // for EventFailed
}

// ComposeTarget carries the label-derived data for a compose restart.
type ComposeTarget struct {
	Project     string
	Service     string
	ProjectDir  string
	ConfigFiles []string
}

// Task is one container/service update.
type Task struct {
	ID           string // stable identifier (container ID)
	Name         string // container name for display
	ImageRef     string // reference to pull
	Auth         string // base64 X-Registry-Auth, "" for anonymous
	RemoteDigest string // digest observed during the check phase
	Compose      *ComposeTarget
	Prune        bool

	Phase       Phase
	Current     int64
	Total       int64
	LayersDone  int
	LayersTotal int
	Err         error
	Started     time.Time
	Finished    time.Time
}

// Apply mutates task state from an event. It is pure w.r.t. the event
// stream and unit-tested directly.
func (t *Task) Apply(e Event) {
	switch e.Kind {
	case EventPhaseChange:
		t.Phase = e.Phase
		if e.Phase == PhasePulling && t.Started.IsZero() {
			t.Started = time.Now()
		}
	case EventPullProgress:
		t.Phase = PhasePulling
		t.Current, t.Total = e.Current, e.Total
		t.LayersDone, t.LayersTotal = e.LayersDone, e.LayersTotal
	case EventDone:
		t.Phase = PhaseDone
		t.Finished = time.Now()
	case EventFailed:
		t.Phase = PhaseFailed
		t.Err = e.Err
		t.Finished = time.Now()
	}
}

// Percent returns pull progress as 0..1, preferring byte progress and
// falling back to completed-layer counts (engines like Podman report no
// byte progress).
func (t *Task) Percent() float64 {
	if t.Total > 0 {
		p := float64(t.Current) / float64(t.Total)
		if p > 1 {
			return 1
		}
		return p
	}
	if t.LayersTotal > 0 {
		return float64(t.LayersDone) / float64(t.LayersTotal)
	}
	return 0
}

// Engine abstracts the container operations the updater needs.
type Engine interface {
	PullImage(ctx context.Context, ref, auth string, progress func(engine.Progress)) error
	RepoDigests(ctx context.Context, imageRef string) ([]string, error)
	RecreateContainer(ctx context.Context, id, newImage string) error
	ImageID(ctx context.Context, containerID string) (string, error)
	ImageIDForRef(ctx context.Context, imageRef string) (string, error)
	RemoveImage(ctx context.Context, imageID string) error
}

// Compose abstracts compose service recreation.
type Compose interface {
	UpRecreate(ctx context.Context, project, dir string, configFiles []string, service string) error
}

// Runner executes tasks with a bounded worker pool.
type Runner struct {
	engine      Engine
	compose     Compose
	concurrency int
}

// NewRunner builds a Runner. concurrency < 1 falls back to 3.
func NewRunner(eng Engine, comp Compose, concurrency int) *Runner {
	if concurrency < 1 {
		concurrency = 3
	}
	return &Runner{engine: eng, compose: comp, concurrency: concurrency}
}

// Concurrency returns the effective worker count.
func (r *Runner) Concurrency() int { return r.concurrency }

// Run executes all tasks concurrently, emitting events to the channel.
// The channel is closed when every task has finished. Each task proceeds
// through the phases independently: a fast pull does not wait for slow ones.
func (r *Runner) Run(ctx context.Context, tasks []*Task, events chan<- Event) {
	var wg sync.WaitGroup
	queue := make(chan *Task)

	workers := r.concurrency
	if workers > len(tasks) {
		workers = len(tasks)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				r.runTask(ctx, t, events)
			}
		}()
	}
	go func() {
		for _, t := range tasks {
			queue <- t
		}
		close(queue)
		wg.Wait()
		close(events)
	}()
}

func (r *Runner) emit(ctx context.Context, events chan<- Event, e Event) {
	select {
	case events <- e:
	case <-ctx.Done():
	}
}

func (r *Runner) fail(ctx context.Context, events chan<- Event, t *Task, err error) {
	r.emit(ctx, events, Event{TaskID: t.ID, Kind: EventFailed, Err: err})
}

func (r *Runner) runTask(ctx context.Context, t *Task, events chan<- Event) {
	fail := func(stage string, err error) {
		r.fail(ctx, events, t, fmt.Errorf("%s: %w", stage, err))
	}

	// Phase 1: pull.
	r.emit(ctx, events, Event{TaskID: t.ID, Kind: EventPhaseChange, Phase: PhasePulling})
	err := r.engine.PullImage(ctx, t.ImageRef, t.Auth, func(p engine.Progress) {
		r.emit(ctx, events, Event{
			TaskID: t.ID, Kind: EventPullProgress,
			Current: p.Current, Total: p.Total,
			LayersDone: p.LayersDone, LayersTotal: p.LayersTotal,
		})
	})
	if err != nil {
		fail("pull", err)
		return
	}

	// Phase 2: verify the pulled image matches the check-phase digest.
	r.emit(ctx, events, Event{TaskID: t.ID, Kind: EventPhaseChange, Phase: PhaseVerifying})
	if t.RemoteDigest != "" {
		digests, err := r.engine.RepoDigests(ctx, t.ImageRef)
		if err != nil {
			fail("verify", err)
			return
		}
		if !containsDigest(digests, t.RemoteDigest) {
			fail("verify", errors.New("checksum mismatch: pulled image digest differs from the checked remote digest"))
			return
		}
	}

	// Capture the old image for optional pruning.
	oldImageID, _ := r.engine.ImageID(ctx, t.ID)

	// Phase 3: restart.
	r.emit(ctx, events, Event{TaskID: t.ID, Kind: EventPhaseChange, Phase: PhaseRestarting})
	if t.Compose != nil {
		if r.compose == nil {
			fail("restart", errors.New("a compose provider is required but none was found"))
			return
		}
		if err := r.compose.UpRecreate(ctx, t.Compose.Project, t.Compose.ProjectDir, t.Compose.ConfigFiles, t.Compose.Service); err != nil {
			fail("restart", err)
			return
		}
	} else {
		if err := r.engine.RecreateContainer(ctx, t.ID, t.ImageRef); err != nil {
			fail("restart", err)
			return
		}
	}

	// Optional: prune the old image once the new container runs. The old
	// container ID may no longer exist at this point (compose/recreate
	// replace it), so compare against the image the ref now resolves to.
	if t.Prune && oldImageID != "" {
		if newID, err := r.engine.ImageIDForRef(ctx, t.ImageRef); err == nil && newID != oldImageID {
			_ = r.engine.RemoveImage(ctx, oldImageID)
		}
	}

	r.emit(ctx, events, Event{TaskID: t.ID, Kind: EventDone})
}

func containsDigest(digests []string, want string) bool {
	for _, d := range digests {
		if strings.EqualFold(d, want) {
			return true
		}
	}
	return false
}
