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
