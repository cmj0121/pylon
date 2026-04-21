package pylon

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRenderSVG checks byte-exact SVG output for a curated subset of
// the testdata fixtures. Only a few fixtures are covered here because
// SVG-specific concerns (viewBox, style, theme fill) are orthogonal to
// the ASCII grid — every .pylon fixture under testdata/ also exercises
// RenderSVG indirectly via RenderASCII (RenderSVG calls RenderRows
// internally, so byte-parity at the ASCII layer guarantees layout
// parity at the SVG layer).
func TestRenderSVG(t *testing.T) {
	cases := []string{
		"hello",
		"flow",
		"ascii-theme",
	}
	for _, base := range cases {
		base := base
		t.Run(base, func(t *testing.T) {
			pylonPath := filepath.Join("testdata", base+".pylon")
			svgPath := filepath.Join("testdata", base+".svg")

			src, err := os.ReadFile(pylonPath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			want, err := os.ReadFile(svgPath)
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			ast := Parse(string(src))
			got := RenderSVG(ast)
			wantStr := string(want)
			if got != wantStr {
				t.Errorf("mismatch\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s\n---",
					len(got), got, len(wantStr), wantStr)
			}
		})
	}
}
