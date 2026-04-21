package pylon

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestRenderPNG exercises the PNG pipeline end-to-end without asserting
// byte-exact output. PNG encoding is sensitive to compression-level and
// IDAT chunking, both of which can drift across Go toolchain versions;
// stable byte-level goldens would create churn without catching real
// bugs. Instead we verify the output is a valid PNG with dimensions
// that match the ASCII grid and a top-left pixel painted in the
// theme-appropriate background color.
func TestRenderPNG(t *testing.T) {
	cases := []struct {
		base string
		// wantBG is the expected top-left pixel (the theme background).
		wantBG [4]uint8
	}{
		{base: "hello", wantBG: [4]uint8{0xfb, 0xf8, 0xef, 0xff}},
		{base: "ascii-theme", wantBG: [4]uint8{0xfb, 0xf8, 0xef, 0xff}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.base, func(t *testing.T) {
			pylonPath := filepath.Join("testdata", tc.base+".pylon")
			src, err := os.ReadFile(pylonPath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			ast := Parse(string(src))
			got, err := RenderPNG(ast)
			if err != nil {
				t.Fatalf("RenderPNG: %v", err)
			}
			if len(got) == 0 {
				t.Fatal("RenderPNG: empty output")
			}

			img, err := png.Decode(bytes.NewReader(got))
			if err != nil {
				t.Fatalf("png.Decode: %v", err)
			}

			// Compute the expected minimum dimensions from the ASCII grid.
			rows := RenderRows(ast)
			maxCols := 1
			for _, r := range rows {
				if w := displayWidth(r); w > maxCols {
					maxCols = w
				}
			}
			wantMinW := maxCols * 4 // defensively loose: at least 4px per cell
			wantMinH := len(rows) * 8

			b := img.Bounds()
			if b.Dx() < wantMinW {
				t.Errorf("width %d < expected minimum %d (cols=%d)",
					b.Dx(), wantMinW, maxCols)
			}
			if b.Dy() < wantMinH {
				t.Errorf("height %d < expected minimum %d (rows=%d)",
					b.Dy(), wantMinH, len(rows))
			}

			// Top-left pixel should be the theme background — the
			// padding area is a solid fill, so there's no glyph in
			// the way.
			r, g, bl, a := img.At(b.Min.X, b.Min.Y).RGBA()
			got8 := [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
			if got8 != tc.wantBG {
				t.Errorf("top-left pixel = %v, want %v", got8, tc.wantBG)
			}

			// Sanity: decoded image should be addressable as RGBA.
			if _, ok := img.(*image.RGBA); !ok {
				// png.Decode may return NRGBA depending on encoder flags,
				// so just check that it's not nil here.
				if img == nil {
					t.Fatal("decoded image is nil")
				}
			}
		})
	}
}
