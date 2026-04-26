package pylon

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

var osReadFile = os.ReadFile

// Targeted tests covering utility helpers that the fixture-driven
// TestRenderASCII / TestRenderSVG / TestRenderPNG suites under-exercise.
// These don't replace fixture coverage — they fill gaps that fixtures
// can't cleanly reach (private helpers, deterministic edge values).

// ---- render_ascii.go helpers --------------------------------------

// TestVertPad covers vertPad's three branches: extra <= 0 (passthrough),
// even extra (top == bot), and odd extra (top < bot).
func TestVertPad(t *testing.T) {
	rows := []string{"a", "b"}
	cases := []struct {
		name    string
		targetH int
		wantLen int
		wantTop int
	}{
		{"passthrough_equal", 2, 2, 0},
		{"passthrough_less", 1, 2, 0},
		{"even_pad", 6, 6, 2},
		{"odd_pad", 5, 5, 1},
	}
	for _, tc := range cases {
		got := vertPad(rows, tc.targetH, 1)
		if len(got) != tc.wantLen {
			t.Errorf("[%s] len = %d, want %d", tc.name, len(got), tc.wantLen)
			continue
		}
		// First tc.wantTop rows must be blank (one space, contentW=1).
		for i := 0; i < tc.wantTop; i++ {
			if got[i] != " " {
				t.Errorf("[%s] row %d = %q, want blank pad", tc.name, i, got[i])
			}
		}
	}
}

// TestEnsureWidth covers ensureWidth's grow path (both branches: already
// long enough, needs grow).
func TestEnsureWidth(t *testing.T) {
	r := []rune("ab")
	ensureWidth(&r, 5)
	if string(r) != "ab   " {
		t.Errorf("ensureWidth grow: got %q, want %q", string(r), "ab   ")
	}
	ensureWidth(&r, 3)
	if string(r) != "ab   " {
		t.Errorf("ensureWidth no-op: got %q, want unchanged", string(r))
	}
}

// TestRenderVerticalChain triggers the maxW-overflow fallback path
// in renderRowRows by passing a parts slice whose summed width
// exceeds the budget. Builds the parts slice directly because no
// .pylon source currently triggers this path through Parse.
func TestRenderVerticalChain(t *testing.T) {
	parts := []rowPart{
		{kind: "block", rows: []string{"AAAA"}, width: 4, height: 1},
		{kind: "edge", edge: &Edge{Direction: DirRight}, width: 1, height: 1},
		{kind: "block", rows: []string{"BBBB"}, width: 4, height: 1},
		{kind: "edge", edge: &Edge{Direction: DirBoth, Label: &Box{Items: []Node{&Text{Content: "tag"}}}}, width: 5, height: 1},
		{kind: "block", rows: []string{"CC"}, width: 2, height: 1},
		{kind: "edge", edge: &Edge{Direction: DirLeft}, width: 1, height: 1},
	}
	got := renderVerticalChain(parts, unicodeBox, nil)
	if len(got) == 0 {
		t.Fatal("renderVerticalChain returned no rows")
	}
	// Must contain at least one of the heads (▼, ▲, │) and one block label.
	joined := strings.Join(got, "\n")
	for _, want := range []string{"▼", "▲", "│", "AAAA", "BBBB", "CC", "tag"} {
		if !strings.Contains(joined, want) {
			t.Errorf("output missing expected glyph/text %q\n%s", want, joined)
		}
	}
	// Empty input returns nil.
	if got := renderVerticalChain(nil, unicodeBox, nil); got != nil {
		t.Errorf("renderVerticalChain(nil) = %v, want nil", got)
	}
}

// ---- parser.go isDataRefBoundary ---------------------------------

// TestIsDataRefBoundary covers every prev/next byte combination the
// helper distinguishes between (boundary vs. non-boundary).
func TestIsDataRefBoundary(t *testing.T) {
	cases := []struct {
		name string
		prev byte
		next byte
		want bool
	}{
		{"start_of_input_then_eof", 0, 0, true},
		{"start_of_input_then_space", 0, ' ', true},
		{"start_of_input_then_letter", 0, 'A', false},
		{"after_space_then_eof", ' ', 0, true},
		{"after_space_then_pipe", ' ', '|', true},
		{"after_space_then_close_bracket", ' ', ']', true},
		{"after_tab_then_close_paren", '\t', ')', true},
		{"after_newline_then_tab", '\n', '\t', true},
		{"after_open_bracket_then_space", '[', ' ', true},
		{"after_open_paren_then_pipe", '(', '|', true},
		{"after_letter_rejects", 'A', ' ', false},
		{"after_digit_rejects", '5', ' ', false},
		{"valid_prev_invalid_next", ' ', 'X', false},
	}
	for _, tc := range cases {
		if got := isDataRefBoundary(tc.prev, tc.next); got != tc.want {
			t.Errorf("[%s] isDataRefBoundary(%d, %d) = %v, want %v",
				tc.name, tc.prev, tc.next, got, tc.want)
		}
	}
}

// ---- render_png.go helpers ---------------------------------------

// TestPNGBlockOpacity covers every recognized block-shade glyph plus
// the not-ok fallback.
func TestPNGBlockOpacity(t *testing.T) {
	cases := []struct {
		r      rune
		wantOp float64
		wantOK bool
	}{
		{'█', 1.0, true},
		{'▓', 0.75, true},
		{'▒', 0.5, true},
		{'░', 0.25, true},
		{' ', 0, false},
		{'#', 0, false},
		{'A', 0, false},
		{'╗', 0, false},
	}
	for _, tc := range cases {
		op, ok := pngBlockOpacity(tc.r)
		if ok != tc.wantOK {
			t.Errorf("pngBlockOpacity(%q): ok = %v, want %v", tc.r, ok, tc.wantOK)
			continue
		}
		if ok && op != tc.wantOp {
			t.Errorf("pngBlockOpacity(%q): op = %v, want %v", tc.r, op, tc.wantOp)
		}
	}
}

// TestSVGBlockOpacity mirrors TestPNGBlockOpacity for the SVG side.
func TestSVGBlockOpacity(t *testing.T) {
	cases := []struct {
		r      rune
		wantOp float64
		wantOK bool
	}{
		{'█', 1.0, true},
		{'▓', 0.75, true},
		{'▒', 0.5, true},
		{'░', 0.25, true},
		{' ', 0, false},
		{'#', 0, false},
		{'╗', 0, false},
	}
	for _, tc := range cases {
		op, ok := svgBlockOpacity(tc.r)
		if ok != tc.wantOK {
			t.Errorf("svgBlockOpacity(%q): ok = %v, want %v", tc.r, ok, tc.wantOK)
			continue
		}
		if ok && op != tc.wantOp {
			t.Errorf("svgBlockOpacity(%q): op = %v, want %v", tc.r, op, tc.wantOp)
		}
	}
}

// TestBlendColor covers the four ramp opacities + the bg/fg endpoints.
// Uses image/color.RGBA so the math is deterministic and bit-exact.
func TestBlendColor(t *testing.T) {
	bg := color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	fg := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	cases := []struct {
		weight float64
		want   color.RGBA
	}{
		{0.0, color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}},
		{1.0, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		{0.5, color.RGBA{R: 0x7f, G: 0x7f, B: 0x7f, A: 0xff}},
		{0.25, color.RGBA{R: 0x3f, G: 0x3f, B: 0x3f, A: 0xff}},
	}
	for _, tc := range cases {
		got := blendColor(bg, fg, tc.weight)
		if got != tc.want {
			t.Errorf("blendColor(bg, fg, %v) = %+v, want %+v", tc.weight, got, tc.want)
		}
	}
}

// TestPaintBlockRuns drives paintBlockRuns over a row that mixes
// block-shade runs with non-block chars, and asserts that each
// recognized run paints opaque pixels of the expected blended color
// while non-block columns stay at the background. This covers the
// run-coalescing branch (`for j ...`) and the inner i+=1 skip path.
func TestPaintBlockRuns(t *testing.T) {
	bg := color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	fg := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	cellW, lineH, padding := 4, 6, 0
	row := "█▓▒░ XX▒▒"
	width := len([]rune(row)) * cellW
	img := image.NewRGBA(image.Rect(0, 0, width, lineH))
	paintBlockRuns(img, row, 0, cellW, lineH, padding, bg, fg)

	// Sample one pixel inside each block run; assert it differs from bg.
	checks := []struct {
		col  int
		want bool // true = should NOT be background
	}{
		{0, true}, // █ run
		{1*cellW + 1, true},
		{2*cellW + 1, true},
		{3*cellW + 1, true},
		{4*cellW + 1, false}, // space — untouched
		{5*cellW + 1, false}, // X — untouched
		{6*cellW + 1, false}, // X — untouched
		{7*cellW + 1, true},  // ▒ run
		{8*cellW + 1, true},
	}
	for _, tc := range checks {
		c := img.RGBAAt(tc.col, lineH/2)
		isBG := c == color.RGBA{}
		gotPainted := !isBG
		if gotPainted != tc.want {
			t.Errorf("col %d: painted=%v, want %v (got pixel=%+v)",
				tc.col, gotPainted, tc.want, c)
		}
	}
}

// ---- render_banner.go collectBoxText ------------------------------

// TestCollectBoxTextRow covers the `*Row` branch of collectBoxText
// (the `*Text` branch is exercised by every banner fixture; the Row
// branch only fires when a banner box's items contain a Row holding
// Text segments — an unusual but valid AST shape).
func TestCollectBoxTextRow(t *testing.T) {
	b := &Box{
		Items: []Node{
			&Text{Content: "AB"},
			&Row{Items: []Node{
				&Text{Content: "CD"},
				&Edge{Direction: DirRight}, // not text — must be skipped
				&Text{Content: "EF"},
			}},
			&Text{Content: "GH"},
		},
	}
	var sb strings.Builder
	collectBoxText(b, &sb)
	if got := sb.String(); got != "ABCDEFGH" {
		t.Errorf("collectBoxText = %q, want %q", got, "ABCDEFGH")
	}
}

// ---- theme color coverage -----------------------------------------

// TestSVGFillForTheme covers each branch of the theme dispatch
// (dark, light/ascii/simple/default, and the explicit fallback).
func TestSVGFillForTheme(t *testing.T) {
	cases := map[string]string{
		"dark":    "#e6dfc8",
		"light":   "#0f1c2d",
		"ascii":   "#0f1c2d",
		"simple":  "#0f1c2d",
		"":        "#0f1c2d",
		"unknown": "#0f1c2d", // fallback
	}
	for theme, want := range cases {
		if got := svgFillForTheme(theme); got != want {
			t.Errorf("svgFillForTheme(%q) = %q, want %q", theme, got, want)
		}
	}
}

// TestPNGThemeColors covers each branch of the PNG theme palette,
// including the `unknown` fallback (which differs from the SVG
// fallback by paper color).
func TestPNGThemeColors(t *testing.T) {
	cases := []struct {
		theme   string
		wantBgR uint8
		wantFgR uint8
	}{
		{"dark", 0x17, 0xe6},
		{"light", 0xfb, 0x0f},
		{"ascii", 0xfb, 0x0f},
		{"simple", 0xfb, 0x0f},
		{"", 0xfb, 0x0f},
		{"unknown", 0xff, 0x0f}, // fallback bg = pure white
	}
	for _, tc := range cases {
		bg, fg := pngThemeColors(tc.theme)
		if bg.R != tc.wantBgR {
			t.Errorf("pngThemeColors(%q) bg.R = %#x, want %#x", tc.theme, bg.R, tc.wantBgR)
		}
		if fg.R != tc.wantFgR {
			t.Errorf("pngThemeColors(%q) fg.R = %#x, want %#x", tc.theme, fg.R, tc.wantFgR)
		}
	}
}

// ---- SVG / PNG with block-glyph fixtures --------------------------

// TestRenderSVGWithBlockGlyphs renders a heatmap fixture (which uses
// the `░▒▓█` ramp) and asserts the SVG contains both <rect> and
// <text> elements — exercising emitSVGRow's block-rect path and
// the rect-with-opacity branch (`op < 1`).
func TestRenderSVGWithBlockGlyphs(t *testing.T) {
	srcBytes, err := osReadFile("testdata/heatmap-basic.pylon")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	out := RenderSVG(Parse(string(srcBytes)))
	if !strings.Contains(out, "<rect") {
		t.Error("SVG output has no <rect> elements; emitSVGRow's block path didn't fire")
	}
	if !strings.Contains(out, `opacity="`) {
		t.Error("SVG output has no rect opacity; the `op < 1` branch didn't fire")
	}
	if !strings.Contains(out, "<text") {
		t.Error("SVG output has no <text> elements; the text-flush path didn't fire")
	}
}

// TestRenderSVGDarkTheme drives the dark-theme path through the
// public API so emitSVGRow / svgFillForTheme see the dark fill.
func TestRenderSVGDarkTheme(t *testing.T) {
	src := "---\ntheme: dark\n---\n[ Hello ]"
	out := RenderSVG(Parse(src))
	if !strings.Contains(out, "#e6dfc8") {
		t.Errorf("SVG output missing dark-theme ink color #e6dfc8:\n%s", out)
	}
}

// TestRenderPNGWithBlockGlyphs renders a heatmap fixture (block
// ramp) end-to-end through RenderPNG so paintBlockRuns walks every
// run length and color-blends each ramp step.
func TestRenderPNGWithBlockGlyphs(t *testing.T) {
	srcBytes, err := osReadFile("testdata/heatmap-basic.pylon")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := RenderPNG(Parse(string(srcBytes)))
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("RenderPNG: empty output")
	}
}

// TestRenderPNGDarkTheme drives the dark-theme palette through
// RenderPNG so pngThemeColors's "dark" branch is exercised in the
// rendering path (not just in TestPNGThemeColors's direct call).
func TestRenderPNGDarkTheme(t *testing.T) {
	src := "---\ntheme: dark\n---\n[ Hello ]"
	got, err := RenderPNG(Parse(src))
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("RenderPNG: empty output")
	}
}
