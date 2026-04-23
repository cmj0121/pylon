package pylon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderASCII walks src/go/pkg/pylon/testdata for matched .pylon /
// .ascii pairs and asserts byte-exact ASCII output parity with the
// JS reference renderer.
func TestRenderASCII(t *testing.T) {
	runFixtureDir(t, "testdata")
}

// runFixtureDir byte-compares every *.pylon / *.ascii pair under dir.
// `.ascii` fixtures have no trailing newline (JS emits the pure
// rendered block), so the compare is direct.
func runFixtureDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	found := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".pylon") {
			continue
		}
		base := strings.TrimSuffix(name, ".pylon")
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s: read source: %v", name, err)
			continue
		}
		want, err := os.ReadFile(filepath.Join(dir, base+".ascii"))
		if err != nil {
			t.Errorf("%s: read expected: %v", name, err)
			continue
		}
		found++

		t.Run(base, func(t *testing.T) {
			got := RenderASCII(Parse(string(src)))
			wantStr := string(want)
			if got != wantStr {
				t.Errorf("mismatch\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s\n---",
					len(got), got, len(wantStr), wantStr)
			}
		})
	}
	if found == 0 {
		t.Fatalf("no fixtures found under %s/", dir)
	}
}
