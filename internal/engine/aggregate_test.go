package engine

import "testing"

func TestLayerAggregatorEmpty(t *testing.T) {
	a := NewLayerAggregator()
	if cur, tot := a.Totals(); cur != 0 || tot != 0 {
		t.Fatalf("empty totals = %d/%d", cur, tot)
	}
}

func TestLayerAggregatorAccumulates(t *testing.T) {
	a := NewLayerAggregator()
	a.Update("l1", "Downloading", 50, 100)
	a.Update("l2", "Downloading", 25, 50)
	if cur, tot := a.Totals(); cur != 75 || tot != 150 {
		t.Fatalf("totals = %d/%d, want 75/150", cur, tot)
	}
	// Progress only moves forward per layer.
	a.Update("l1", "Downloading", 30, 100)
	if cur, _ := a.Totals(); cur != 75 {
		t.Fatalf("regressing progress must be ignored, cur = %d", cur)
	}
}

func TestLayerAggregatorCompletionStatuses(t *testing.T) {
	a := NewLayerAggregator()
	a.Update("l1", "Downloading", 80, 100)
	a.Update("l1", "Download complete", 0, 0)
	if cur, tot := a.Totals(); cur != 100 || tot != 100 {
		t.Fatalf("completed layer must report full bytes, got %d/%d", cur, tot)
	}
	a.Update("l2", "Already exists", 0, 0)
	if cur, tot := a.Totals(); cur != 100 || tot != 100 {
		t.Fatalf("already-exists layer with unknown size must not change totals, got %d/%d", cur, tot)
	}
}

func TestLayerAggregatorIgnoresEmptyID(t *testing.T) {
	a := NewLayerAggregator()
	a.Update("", "Pulling fs layer", 10, 10)
	if cur, tot := a.Totals(); cur != 0 || tot != 0 {
		t.Fatalf("empty id must be ignored, got %d/%d", cur, tot)
	}
}

func TestLayerAggregatorLayerCounts(t *testing.T) {
	a := NewLayerAggregator()
	a.Update("l1", "Pulling fs layer", 0, 0)
	a.Update("l2", "Pulling fs layer", 0, 0)
	if done, total := a.Layers(); done != 0 || total != 2 {
		t.Fatalf("layers = %d/%d, want 0/2", done, total)
	}
	a.Update("l1", "Download complete", 0, 0)
	if done, total := a.Layers(); done != 1 || total != 2 {
		t.Fatalf("layers = %d/%d, want 1/2", done, total)
	}
	a.Update("l2", "Pull complete", 0, 0)
	if done, total := a.Layers(); done != 2 || total != 2 {
		t.Fatalf("layers = %d/%d, want 2/2", done, total)
	}
}
