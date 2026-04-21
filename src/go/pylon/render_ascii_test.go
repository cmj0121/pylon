package pylon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderASCII walks src/go/pylon/testdata for matched .pylon /
// .ascii pairs and asserts byte-exact ASCII output parity with the
// JS reference renderer.
func TestRenderASCII(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	found := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".pylon") {
			continue
		}
		base := strings.TrimSuffix(name, ".pylon")
		pylonPath := filepath.Join("testdata", name)
		asciiPath := filepath.Join("testdata", base+".ascii")

		src, err := os.ReadFile(pylonPath)
		if err != nil {
			t.Errorf("%s: read source: %v", name, err)
			continue
		}
		want, err := os.ReadFile(asciiPath)
		if err != nil {
			t.Errorf("%s: read expected: %v", name, err)
			continue
		}
		found++

		t.Run(base, func(t *testing.T) {
			ast := Parse(string(src))
			got := RenderASCII(ast)
			// `.ascii` fixtures have no trailing newline (JS emits
			// the pure rendered block). Use direct string compare.
			wantStr := string(want)
			if got != wantStr {
				t.Errorf("mismatch\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s\n---",
					len(got), got, len(wantStr), wantStr)
			}
		})
	}
	if found == 0 {
		t.Fatalf("no fixtures found under testdata/")
	}
}
