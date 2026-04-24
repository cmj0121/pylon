package pylon

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

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
		baseline := pngPadding + i*lineH + ascent
		drawer.Dot = fixed.P(pngPadding, baseline)
		drawer.DrawString(row)
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
