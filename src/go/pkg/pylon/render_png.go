package pylon

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// JetBrains Mono Regular, embedded so the binary stays self-contained.
// The font is licensed under the SIL Open Font License; see
// assets/JetBrainsMono-OFL.txt for the full text (not embedded, but
// shipped in the source tree for humans).
//
//go:embed assets/JetBrainsMono-Regular.ttf
var jetBrainsMonoRegular []byte

// PNG render constants. Font size and DPI are chosen so the resulting
// cell width matches the SVG renderer's 7x16 cell — JetBrains Mono at
// 10pt / 72 DPI gives an advance width near 6-7 px, the tight grid
// that kept chart primitives from looking spacious in raster output.
const (
	pngFontSize = 10.0
	pngDPI      = 72.0
	pngPadding  = 4
)

// pngThemeColors returns the background and foreground RGBA for the
// given theme. Backgrounds mirror the paper / dark surfaces from the JS
// CSS (`--pylon-paper`, `--pylon-surface-dark`); foregrounds mirror the
// same `--pylon-ink` values used by RenderSVG.
func pngThemeColors(theme string) (bg, fg color.RGBA) {
	switch theme {
	case "dark":
		return color.RGBA{R: 0x17, G: 0x1d, B: 0x28, A: 0xff},
			color.RGBA{R: 0xe6, G: 0xdf, B: 0xc8, A: 0xff}
	case "light", "ascii", "simple", "":
		return color.RGBA{R: 0xfb, G: 0xf8, B: 0xef, A: 0xff},
			color.RGBA{R: 0x0f, G: 0x1c, B: 0x2d, A: 0xff}
	default:
		return color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			color.RGBA{R: 0x0f, G: 0x1c, B: 0x2d, A: 0xff}
	}
}

// RenderPNG turns an AST into a PNG bitmap. Layout mirrors RenderSVG:
// the ASCII grid is painted one row per text line, using the embedded
// JetBrains Mono Regular font at 14pt.
//
// Returns the encoded PNG bytes. The caller is responsible for writing
// them to the intended sink (stdout / file / buffer).
func RenderPNG(ast *Box) ([]byte, error) {
	rows := RenderRows(ast)
	if len(rows) == 0 {
		rows = []string{""}
	}

	face, err := loadPNGFace()
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	defer face.Close()

	cellW, lineH, ascent := pngCellMetrics(face)

	// Compute the widest row in *display* cells — matches ASCII
	// width semantics so CJK runes count as two cells even though
	// JetBrains Mono does not have wide-width glyphs for them. The
	// bitmap will simply leave a blank half-cell; fidelity for wide
	// scripts is explicitly out of scope (same caveat as RenderSVG).
	maxCols := 1
	for _, r := range rows {
		if w := displayWidth(r); w > maxCols {
			maxCols = w
		}
	}

	width := cellW*maxCols + 2*pngPadding
	height := lineH*len(rows) + 2*pngPadding

	bg, fg := pngThemeColors(ast.Meta.Theme)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{C: fg},
		Face: face,
	}
	for i, row := range rows {
		yTop := pngPadding + i*lineH
		paintBlockRuns(img, row, yTop, cellW, lineH, pngPadding, bg, fg)
		baseline := yTop + ascent
		drawer.Dot = fixed.P(pngPadding, baseline)
		// Text pass: blank out the cells already painted as rects so
		// the font glyph doesn't overlay the clean rectangle.
		drawer.DrawString(stripBlockGlyphs(row))
	}

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// loadPNGFace parses the embedded TTF and builds a font.Face sized for
// PNG rendering. The returned face must be closed by the caller.
func loadPNGFace() (font.Face, error) {
	ft, err := sfnt.Parse(jetBrainsMonoRegular)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    pngFontSize,
		DPI:     pngDPI,
		Hinting: font.HintingFull,
	})
}

// pngCellMetrics returns the advance width for the reference glyph M,
// the full line height (ascent + descent), and the ascent alone (used
// to offset the baseline from the top of each row).
func pngCellMetrics(face font.Face) (cellW, lineH, ascent int) {
	metrics := face.Metrics()
	lineH = (metrics.Ascent + metrics.Descent).Ceil()
	ascent = metrics.Ascent.Ceil()

	// Advance width for 'M' — JetBrains Mono is a monospace font so
	// every Latin glyph has the same advance, but 'M' is the
	// conventional reference choice.
	adv, ok := face.GlyphAdvance('M')
	if !ok || adv == 0 {
		// Fall back to ~1/2 em for robustness; shouldn't happen with
		// JetBrains Mono but keeps the function total.
		adv = fixed.I(int(pngFontSize / 2))
	}
	cellW = adv.Ceil()
	return cellW, lineH, ascent
}

// pngBlockOpacity mirrors svgBlockOpacity: maps a Unicode block-shade
// glyph to its ramp opacity (0 < op <= 1) for native rectangle
// rendering in the PNG output. Returns ok=false for any other rune.
func pngBlockOpacity(r rune) (float64, bool) {
	switch r {
	case '█':
		return 1.0, true
	case '▓':
		return 0.75, true
	case '▒':
		return 0.5, true
	case '░':
		return 0.25, true
	}
	return 0, false
}

// blendColor mixes bg and fg at the given foreground weight. Works in
// 16-bit premultiplied RGBA (image/color's native resolution) and
// returns an opaque RGBA so image/draw's Src operator lays it down
// as-is.
func blendColor(bg, fg color.Color, fgWeight float64) color.RGBA {
	br, bgn, bb, _ := bg.RGBA()
	fr, fgn, fb, _ := fg.RGBA()
	mix := func(b, f uint32) uint8 {
		v := float64(b)*(1-fgWeight) + float64(f)*fgWeight
		return uint8(uint32(v) >> 8)
	}
	return color.RGBA{R: mix(br, fr), G: mix(bgn, fgn), B: mix(bb, fb), A: 0xff}
}

// paintBlockRuns walks a row and fills contiguous runs of Unicode
// block-shade glyphs as solid rectangles in img. Each run takes its
// own blended color (bg mixed toward fg at the ramp's opacity), so
// bars render as clean dithered / solid blocks regardless of how the
// embedded font rasterises the `█▓▒░` code points.
func paintBlockRuns(img *image.RGBA, row string, yTop, cellW, lineH, padding int, bg, fg color.Color) {
	runes := []rune(row)
	i := 0
	for i < len(runes) {
		op, ok := pngBlockOpacity(runes[i])
		if !ok {
			i++
			continue
		}
		j := i
		for j < len(runes) {
			op2, ok2 := pngBlockOpacity(runes[j])
			if !ok2 || op2 != op {
				break
			}
			j++
		}
		rect := image.Rect(padding+i*cellW, yTop, padding+j*cellW, yTop+lineH)
		c := blendColor(bg, fg, op)
		draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
		i = j
	}
}

// stripBlockGlyphs replaces every `█▓▒░` rune with a space so the
// font-drawing pass doesn't overlay a glyph on top of the rectangle
// already painted by paintBlockRuns. Every other character passes
// through unchanged.
func stripBlockGlyphs(row string) string {
	var b strings.Builder
	b.Grow(len(row))
	for _, r := range row {
		if _, isBlock := pngBlockOpacity(r); isBlock {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
