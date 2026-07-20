package updater

import (
	"github.com/amagyar/dockupdate/internal/engine"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeEngine implements Engine for tests.
type fakeEngine struct {
	mu         sync.Mutex
	pulled     []string
	recreated  []string
	removedImg []string

	digests     map[string][]string // ref -> repo digests
	pullErr     error
	recreateErr error
	digestsErr  error

	oldImageID string
	newImageID string

	inFlight    int
	maxInFlight int
	pullBlock   chan struct{} // when non-nil, pulls wait for a value
}

func (f *fakeEngine) PullImage(ctx context.Context, ref, auth string, progress func(engine.Progress)) error {
	f.mu.Lock()
	f.inFlight++
	if f.inFlight > f.maxInFlight {
		f.maxInFlight = f.inFlight
	}
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.inFlight--; f.mu.Unlock() }()

	if f.pullBlock != nil {
		select {
		case <-f.pullBlock:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if f.pullErr != nil {
		return f.pullErr
	}
	f.mu.Lock()
	f.pulled = append(f.pulled, ref)
	f.mu.Unlock()
	if progress != nil {
		progress(engine.Progress{Current: 50, Total: 100, LayersDone: 1, LayersTotal: 2})
		progress(engine.Progress{Current: 100, Total: 100, LayersDone: 2, LayersTotal: 2})
	}
	return nil
}

func (f *fakeEngine) RepoDigests(ctx context.Context, ref string) ([]string, error) {
	if f.digestsErr != nil {
		return nil, f.digestsErr
	}
	return f.digests[ref], nil
}

func (f *fakeEngine) RecreateContainer(ctx context.Context, id, newImage string) error {
	if f.recreateErr != nil {
		return f.recreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recreated = append(f.recreated, id)
	return nil
}

func (f *fakeEngine) ImageID(ctx context.Context, containerID string) (string, error) {
	return f.oldImageID, nil
}

func (f *fakeEngine) ImageIDForRef(ctx context.Context, ref string) (string, error) {
	return f.newImageID, nil
}

func (f *fakeEngine) RemoveImage(ctx context.Context, imageID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removedImg = append(f.removedImg, imageID)
	return nil
}

// fakeCompose implements Compose for tests.
type fakeCompose struct {
	mu    sync.Mutex
	err   error
	calls []string
}

func (f *fakeCompose) UpRecreate(ctx context.Context, project, dir string, configFiles []string, service string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("%s/%s", project, service))
	return f.err
}

func runCollect(r *Runner, tasks []*Task) []Event {
	events := make(chan Event, 256)
	r.Run(context.Background(), tasks, events)
	var out []Event
	for e := range events {
		out = append(out, e)
	}
	return out
}

func phasesOf(events []Event, taskID string) []Phase {
	var out []Phase
	for _, e := range events {
		if e.TaskID == taskID && e.Kind == EventPhaseChange {
			out = append(out, e.Phase)
		}
	}
	return out
}

func hasEvent(events []Event, taskID string, kind EventKind) bool {
	for _, e := range events {
		if e.TaskID == taskID && e.Kind == kind {
			return true
		}
	}
	return false
}

func TestTaskApplyTransitions(t *testing.T) {
	tk := &Task{ID: "x"}
	tk.Apply(Event{TaskID: "x", Kind: EventPhaseChange, Phase: PhasePulling})
	if tk.Phase != PhasePulling || tk.Started.IsZero() {
		t.Fatalf("pulling: %+v", tk)
	}
	tk.Apply(Event{TaskID: "x", Kind: EventPullProgress, Current: 25, Total: 100})
	if tk.Percent() != 0.25 {
		t.Fatalf("percent = %v", tk.Percent())
	}
	tk.Apply(Event{TaskID: "x", Kind: EventPullProgress, Current: 200, Total: 100})
	if tk.Percent() != 1 {
		t.Fatalf("percent must clamp to 1, got %v", tk.Percent())
	}
	tk.Apply(Event{TaskID: "x", Kind: EventPhaseChange, Phase: PhaseVerifying})
	tk.Apply(Event{TaskID: "x", Kind: EventPhaseChange, Phase: PhaseRestarting})
	tk.Apply(Event{TaskID: "x", Kind: EventDone})
	if tk.Phase != PhaseDone || tk.Finished.IsZero() {
		t.Fatalf("done: %+v", tk)
	}

	tk2 := &Task{ID: "y"}
	tk2.Apply(Event{TaskID: "y", Kind: EventFailed, Err: errors.New("boom")})
	if tk2.Phase != PhaseFailed || tk2.Err == nil {
		t.Fatalf("failed: %+v", tk2)
	}
}

func TestPhaseStrings(t *testing.T) {
	want := map[Phase]string{
		PhasePulling:    "pulling",
		PhaseVerifying:  "verifying checksum",
		PhaseRestarting: "restarting service",
		PhaseDone:       "success",
		PhaseFailed:     "failed",
	}
	for p, w := range want {
		if p.String() != w {
			t.Fatalf("phase %d = %q, want %q", p, p.String(), w)
		}
	}
}

func TestRunComposeTaskHappyPath(t *testing.T) {
	eng := &fakeEngine{digests: map[string][]string{"nginx:1.26": {"sha256:remote"}}}
	comp := &fakeCompose{}
	r := NewRunner(eng, comp, 3)

	task := &Task{
		ID: "c1", Name: "web", ImageRef: "nginx:1.26", RemoteDigest: "sha256:remote",
		Compose: &ComposeTarget{Project: "webapp", Service: "web"},
	}
	events := runCollect(r, []*Task{task})

	want := []Phase{PhasePulling, PhaseVerifying, PhaseRestarting}
	got := phasesOf(events, "c1")
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("phases = %v, want %v", got, want)
	}
	if !hasEvent(events, "c1", EventDone) {
		t.Fatalf("missing done event: %v", events)
	}
	if len(comp.calls) != 1 || comp.calls[0] != "webapp/web" {
		t.Fatalf("compose calls = %v", comp.calls)
	}
	if len(eng.recreated) != 0 {
		t.Fatalf("compose task must not use standalone recreate: %v", eng.recreated)
	}
}

func TestRunStandaloneTaskHappyPath(t *testing.T) {
	eng := &fakeEngine{digests: map[string][]string{"redis:7": {"sha256:remote"}}}
	r := NewRunner(eng, nil, 3)

	task := &Task{ID: "c2", Name: "cache", ImageRef: "redis:7", RemoteDigest: "sha256:remote"}
	events := runCollect(r, []*Task{task})

	if !hasEvent(events, "c2", EventDone) {
		t.Fatalf("missing done: %v", events)
	}
	if len(eng.recreated) != 1 || eng.recreated[0] != "c2" {
		t.Fatalf("standalone recreate not called: %v", eng.recreated)
	}
}

func TestRunChecksumMismatchFailsBeforeRestart(t *testing.T) {
	eng := &fakeEngine{digests: map[string][]string{"nginx:1.26": {"sha256:other"}}}
	comp := &fakeCompose{}
	r := NewRunner(eng, comp, 3)

	task := &Task{
		ID: "c1", Name: "web", ImageRef: "nginx:1.26", RemoteDigest: "sha256:remote",
		Compose: &ComposeTarget{Project: "webapp", Service: "web"},
	}
	events := runCollect(r, []*Task{task})

	var failEv *Event
	for i, e := range events {
		if e.Kind == EventFailed {
			failEv = &events[i]
		}
	}
	if failEv == nil || !strings.Contains(failEv.Err.Error(), "checksum mismatch") {
		t.Fatalf("want checksum mismatch failure, got %v", events)
	}
	if len(comp.calls) != 0 {
		t.Fatalf("restart must not happen after mismatch: %v", comp.calls)
	}
}

func TestRunPullErrorFails(t *testing.T) {
	eng := &fakeEngine{pullErr: errors.New("network down")}
	r := NewRunner(eng, nil, 3)
	events := runCollect(r, []*Task{{ID: "c1", ImageRef: "x:1"}})
	if !hasEvent(events, "c1", EventFailed) || hasEvent(events, "c1", EventDone) {
		t.Fatalf("pull error must fail the task: %v", events)
	}
}

func TestRunComposeMissingProvider(t *testing.T) {
	eng := &fakeEngine{digests: map[string][]string{"nginx:1.26": {"sha256:remote"}}}
	r := NewRunner(eng, nil, 3) // no compose provider
	task := &Task{
		ID: "c1", ImageRef: "nginx:1.26", RemoteDigest: "sha256:remote",
		Compose: &ComposeTarget{Project: "p", Service: "s"},
	}
	events := runCollect(r, []*Task{task})
	var failEv *Event
	for i, e := range events {
		if e.Kind == EventFailed {
			failEv = &events[i]
		}
	}
	if failEv == nil || !strings.Contains(failEv.Err.Error(), "compose provider") {
		t.Fatalf("want compose-provider error, got %v", events)
	}
}

func TestRunFailureIsolation(t *testing.T) {
	eng := &fakeEngine{
		digests: map[string][]string{
			"good:1": {"sha256:ok"},
			"bad:1":  {"sha256:ok"},
		},
	}
	// Make the standalone recreate fail only for the bad task by keying on ID.
	eng2 := &selectiveFailEngine{fakeEngine: eng, failID: "bad-id"}
	r := NewRunner(eng2, nil, 5)

	tasks := []*Task{
		{ID: "good-id", ImageRef: "good:1", RemoteDigest: "sha256:ok"},
		{ID: "bad-id", ImageRef: "bad:1", RemoteDigest: "sha256:ok"},
		{ID: "good-id-2", ImageRef: "good:1", RemoteDigest: "sha256:ok"},
	}
	events := runCollect(r, tasks)

	if !hasEvent(events, "bad-id", EventFailed) {
		t.Fatalf("bad task must fail: %v", events)
	}
	for _, id := range []string{"good-id", "good-id-2"} {
		if !hasEvent(events, id, EventDone) {
			t.Fatalf("%s must complete despite sibling failure: %v", id, events)
		}
	}
}

type selectiveFailEngine struct {
	*fakeEngine
	failID string
}

func (s *selectiveFailEngine) RecreateContainer(ctx context.Context, id, newImage string) error {
	if id == s.failID {
		return errors.New("create: boom")
	}
	return s.fakeEngine.RecreateContainer(ctx, id, newImage)
}

func TestRunConcurrencyBound(t *testing.T) {
	block := make(chan struct{})
	eng := &fakeEngine{
		pullBlock: block,
		digests:   map[string][]string{"img:1": {"sha256:x"}},
	}
	r := NewRunner(eng, nil, 2)

	tasks := []*Task{
		{ID: "t1", ImageRef: "img:1", RemoteDigest: "sha256:x"},
		{ID: "t2", ImageRef: "img:1", RemoteDigest: "sha256:x"},
		{ID: "t3", ImageRef: "img:1", RemoteDigest: "sha256:x"},
		{ID: "t4", ImageRef: "img:1", RemoteDigest: "sha256:x"},
		{ID: "t5", ImageRef: "img:1", RemoteDigest: "sha256:x"},
	}
	events := make(chan Event, 256)
	r.Run(context.Background(), tasks, events)

	// Let workers start, then release pulls one by one.
	deadline := time.After(5 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case block <- struct{}{}:
		case <-deadline:
			t.Fatal("timed out releasing pulls")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range events {
	}
	if eng.maxInFlight > 2 {
		t.Fatalf("concurrency bound violated: %d pulls in flight", eng.maxInFlight)
	}
}

func TestRunIndependentCompletion(t *testing.T) {
	// Slow task blocks until released; fast task must finish first.
	block := make(chan struct{})
	eng := &blockingFirstEngine{
		fakeEngine: &fakeEngine{digests: map[string][]string{
			"slow:1": {"sha256:x"},
			"fast:1": {"sha256:x"},
		}},
		blockRef: "slow:1",
		block:    block,
	}
	r := NewRunner(eng, nil, 2)
	tasks := []*Task{
		{ID: "slow", ImageRef: "slow:1", RemoteDigest: "sha256:x"},
		{ID: "fast", ImageRef: "fast:1", RemoteDigest: "sha256:x"},
	}
	events := make(chan Event, 256)
	r.Run(context.Background(), tasks, events)

	// Wait for the fast task's done event before releasing the slow pull.
	gotFastDone := false
	timeout := time.After(5 * time.Second)
	for !gotFastDone {
		select {
		case ev := <-events:
			if ev.TaskID == "fast" && ev.Kind == EventDone {
				gotFastDone = true
			}
			if ev.TaskID == "slow" && ev.Kind == EventDone {
				t.Fatal("slow task finished before fast task")
			}
		case <-timeout:
			t.Fatal("fast task did not finish while slow was blocked")
		}
	}
	close(block)
	for range events {
	}
}

type blockingFirstEngine struct {
	*fakeEngine
	blockRef string
	block    chan struct{}
	once     sync.Once
}

func (b *blockingFirstEngine) PullImage(ctx context.Context, ref, auth string, progress func(engine.Progress)) error {
	if ref == b.blockRef {
		<-b.block
		b.once.Do(func() {})
	}
	return b.fakeEngine.PullImage(ctx, ref, auth, progress)
}

func TestPruneBehavior(t *testing.T) {
	eng := &fakeEngine{
		digests:    map[string][]string{"img:1": {"sha256:x"}},
		oldImageID: "sha256:old",
		newImageID: "sha256:new",
	}
	r := NewRunner(eng, nil, 1)

	// Prune enabled: old image removed.
	events := runCollect(r, []*Task{{ID: "t1", ImageRef: "img:1", RemoteDigest: "sha256:x", Prune: true}})
	if !hasEvent(events, "t1", EventDone) {
		t.Fatalf("prune task failed: %v", events)
	}
	if len(eng.removedImg) != 1 || eng.removedImg[0] != "sha256:old" {
		t.Fatalf("old image must be pruned: %v", eng.removedImg)
	}

	// Prune disabled: nothing removed.
	eng2 := &fakeEngine{
		digests:    map[string][]string{"img:1": {"sha256:x"}},
		oldImageID: "sha256:old",
		newImageID: "sha256:new",
	}
	r2 := NewRunner(eng2, nil, 1)
	runCollect(r2, []*Task{{ID: "t1", ImageRef: "img:1", RemoteDigest: "sha256:x"}})
	if len(eng2.removedImg) != 0 {
		t.Fatalf("without --prune no image must be removed: %v", eng2.removedImg)
	}
}
