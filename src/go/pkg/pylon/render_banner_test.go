package pylon

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Banner fixtures now live in testdata/ alongside every other fixture
// and are walked by TestRenderASCII / the parity gate. This file only
// holds the font-table invariants.

// TestBannerFontTables guards against font-table typos. Hand-written
// ASCII art is easy to mis-space and a single short row corrupts
// every fixture that uses the letter. Asserts:
//
//   - Both tables cover the same rune set (renderer falls back to '?'
//     on miss, so an asymmetric key would silently diverge).
//   - All 6 rows of a glyph have the same rune-count width (not byte
//     count — ANSI-shadow glyphs use multi-byte runes). Trailing-
//     space counts need NOT match across rows; letters like Y / V / Z
//     have ragged trails whose rune-width is what keeps the rectangle.
//   - No tabs anywhere — they'd render unpredictably on terminals.
func TestBannerFontTables(t *testing.T) {
	if len(bannerFontDefault) != len(bannerFontASCII) {
		t.Fatalf("font tables disagree on size: default=%d ascii=%d",
			len(bannerFontDefault), len(bannerFontASCII))
	}
	for r, defGlyph := range bannerFontDefault {
		asciiGlyph, ok := bannerFontASCII[r]
		if !ok {
			t.Errorf("rune %q present in default but missing in ascii", r)
			continue
		}
		checkBannerGlyph(t, "bannerFontDefault", r, defGlyph)
		checkBannerGlyph(t, "bannerFontASCII", r, asciiGlyph)
	}
}

func checkBannerGlyph(t *testing.T, name string, r rune, glyph [6]string) {
	t.Helper()
	width := utf8.RuneCountInString(glyph[0])
	for i, row := range glyph {
		if strings.ContainsRune(row, '\t') {
			t.Errorf("%s[%q] row %d: contains tab", name, r, i)
		}
		if w := utf8.RuneCountInString(row); w != width {
			t.Errorf("%s[%q] row %d: width %d, want %d (row=%q)",
				name, r, i, w, width, row)
		}
	}
}

// TestBannerFontMonospace enforces the contract that makes the
// monospace font monospace, which is stricter than the default↔ascii
// pair invariant. The renderer relies on a `█→#` row-level
// substitution to support `theme: ascii` without a sibling table, so
// any rune outside {`█`, ` `} would either appear unchanged in the
// substituted output or break the assumption that every cell is
// either fully filled or fully blank.
func TestBannerFontMonospace(t *testing.T) {
	checkPureBlockFont(t, "bannerFontMonospace", bannerFontMonospace, 8)
}

// TestBannerFontDigital enforces the same pure-block invariants as
// monospace, with the digital font's own width contract (6 cells).
func TestBannerFontDigital(t *testing.T) {
	checkPureBlockFont(t, "bannerFontDigital", bannerFontDigital, 6)
}

// TestBannerFontMini enforces the same pure-block invariants as
// monospace, with the mini font's narrower width contract (5 cells).
func TestBannerFontMini(t *testing.T) {
	checkPureBlockFont(t, "bannerFontMini", bannerFontMini, 5)
}

// checkPureBlockFont validates the invariants shared by every
// `█`+space banner font: each glyph has 6 rows of exactly wantWidth
// runes, the last rune of every row is the inter-letter pad space,
// every rune is in {`█`, ` `}, and the `?` fallback glyph is present.
// Used by all three pure-block fonts (monospace, digital, mini).
func checkPureBlockFont(t *testing.T, name string, font map[rune][6]string, wantWidth int) {
	t.Helper()
	if len(font) == 0 {
		t.Fatalf("%s is empty", name)
	}
	if _, ok := font['?']; !ok {
		t.Errorf("%s missing '?' fallback glyph", name)
	}
	for r, glyph := range font {
		for i, row := range glyph {
			if strings.ContainsRune(row, '\t') {
				t.Errorf("%s[%q] row %d: contains tab", name, r, i)
			}
			if w := utf8.RuneCountInString(row); w != wantWidth {
				t.Errorf("%s[%q] row %d: width %d, want %d (row=%q)",
					name, r, i, w, wantWidth, row)
				continue
			}
			runes := []rune(row)
			if runes[len(runes)-1] != ' ' {
				t.Errorf("%s[%q] row %d: last rune is %q, want ' ' (col-pad invariant)",
					name, r, i, runes[len(runes)-1])
			}
			for j, rr := range runes {
				if rr != '█' && rr != ' ' {
					t.Errorf("%s[%q] row %d col %d: rune %q outside {'█', ' '}",
						name, r, i, j, rr)
				}
			}
		}
	}
}

// TestBannerFontASCIIIsPureASCII pins that bannerFontASCII contains
// only printable ASCII (no chars >= 0x80). The default font mixes
// EAW-narrow ASCII spaces with EAW-ambiguous box-drawing chars, which
// produces visually misaligned rows when rendered in CJK / EAW-wide
// contexts (see SPEC banner section). The ASCII font is the
// recommended workaround precisely because every char is narrow —
// rows render at uniform visual width in any East Asian width mode.
// Adding a non-ASCII char to bannerFontASCII would silently break
// that property; this test guards against the regression.
func TestBannerFontASCIIIsPureASCII(t *testing.T) {
	for r, glyph := range bannerFontASCII {
		for i, row := range glyph {
			for _, c := range row {
				if c >= 0x80 {
					t.Errorf("bannerFontASCII[%q] row %d: rune %q (U+%04X) is non-ASCII; "+
						"the ASCII font is the EAW-uniform fallback and must stay narrow-only",
						r, i, c, c)
				}
			}
		}
	}
}

// TestRenderBannerASCIIThemeIsPureASCII covers the EAW-uniform
// guarantee end-to-end: render the worked-example "Pylon" through
// every banner font under `theme: ascii` and assert the output
// contains zero chars >= 0x80. This verifies both the table palette
// (already asserted by TestBannerFontMonospace/Digital/Mini) and
// the renderer's `█→#` substitution path actually deliver pure-ASCII
// output. A regression in either layer (a non-`█` glyph leaking
// through, or the substitution silently dropping a row) trips this
// test. SPEC promises this property for every font under ascii
// theme; this test is the runtime enforcement.
func TestRenderBannerASCIIThemeIsPureASCII(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"default", "---\ntheme: ascii\n---\n[ Pylon | banner ]"},
		{"monospace", "---\ntheme: ascii\nbanner: monospace\n---\n[ Pylon | banner ]"},
		{"digital", "---\ntheme: ascii\n---\n[ Pylon | banner:digital ]"},
		{"mini", "---\ntheme: ascii\n---\n[ Pylon | banner:mini ]"},
	}
	for _, tc := range cases {
		out := RenderASCII(Parse(tc.src))
		for i, row := range strings.Split(out, "\n") {
			for j, c := range row {
				if c >= 0x80 {
					t.Errorf("[%s] row %d col %d: rune %q (U+%04X) is non-ASCII under theme:ascii; "+
						"every banner font must yield pure-ASCII output under ascii theme to stay EAW-uniform",
						tc.name, i, j, c, c)
				}
			}
		}
	}
}

// TestRenderBannerASCIIThemeRowsAreUniformWidth is the direct
// EAW-alignment guarantee: render the worked-example "Pylon"
// through every banner font under `theme: ascii` and assert all
// six rendered banner rows have the same rune count. Equal
// rune-count + pure ASCII (the previous test) ⇒ equal display
// width in any East Asian width context. The default font without
// `theme: ascii` deliberately fails this property — see issue #1 —
// which is why this test only covers the ascii-theme paths.
func TestRenderBannerASCIIThemeRowsAreUniformWidth(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"default", "---\ntheme: ascii\n---\n[ Pylon | banner ]"},
		{"monospace", "---\ntheme: ascii\nbanner: monospace\n---\n[ Pylon | banner ]"},
		{"digital", "---\ntheme: ascii\n---\n[ Pylon | banner:digital ]"},
		{"mini", "---\ntheme: ascii\n---\n[ Pylon | banner:mini ]"},
	}
	for _, tc := range cases {
		out := RenderASCII(Parse(tc.src))
		rows := strings.Split(out, "\n")
		if len(rows) == 0 {
			t.Errorf("[%s]: empty render", tc.name)
			continue
		}
		want := utf8.RuneCountInString(rows[0])
		for i, row := range rows {
			if got := utf8.RuneCountInString(row); got != want {
				t.Errorf("[%s] row %d width %d, want %d (row=%q); "+
					"banner rows under theme:ascii must be uniform-width to stay EAW-aligned",
					tc.name, i, got, want, row)
			}
		}
	}
}
