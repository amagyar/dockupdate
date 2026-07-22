package compose

import (
	"strings"
	"testing"
)

func FuzzSplitConfigFiles(f *testing.F) {
	for _, seed := range []string{
		"",
		"docker-compose.yml",
		"a.yml,b.yml",
		" a.yml , b.yml ,",
		",,,",
		"a.yml,,b.yml",
		"/path/with space/compose.yaml",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, label string) {
		out := SplitConfigFiles(label)
		for _, file := range out {
			if file == "" {
				t.Fatalf("empty entry for %q", label)
			}
			if file != strings.TrimSpace(file) {
				t.Fatalf("untrimmed entry %q for %q", file, label)
			}
			if strings.Contains(file, ",") {
				t.Fatalf("entry %q contains comma for %q", file, label)
			}
		}
		// Re-splitting the rejoined list must round-trip.
		again := SplitConfigFiles(strings.Join(out, ","))
		if len(again) != len(out) {
			t.Fatalf("re-split of %v mismatch: %v", out, again)
		}
	})
}
