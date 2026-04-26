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
