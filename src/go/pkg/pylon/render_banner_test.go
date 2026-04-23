package pylon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestRenderBanner walks src/go/pkg/pylon/testdata_banner for matched
// .pylon / .ascii pairs and asserts byte-exact banner output. The
// fixture dir is kept OUTSIDE testdata/ so the parity-CI corpus
// stays in lockstep with the JS reference renderer (banner is Go-
// only in v1 — see PLAN.md "Isolation from parity CI").
func TestRenderBanner(t *testing.T) {
	entries, err := os.ReadDir("testdata_banner")
	if err != nil {
		t.Fatalf("read testdata_banner: %v", err)
	}
	found := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".pylon") {
			continue
		}
		base := strings.TrimSuffix(name, ".pylon")
		pylonPath := filepath.Join("testdata_banner", name)
		asciiPath := filepath.Join("testdata_banner", base+".ascii")

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
			wantStr := string(want)
			if got != wantStr {
				t.Errorf("mismatch\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s\n---",
					len(got), got, len(wantStr), wantStr)
			}
		})
	}
	if found == 0 {
		t.Fatalf("no fixtures found under testdata_banner/")
	}
}

// TestBannerFontTables is the front line of defence against font-
// table typos. Hand-written ASCII art is easy to mis-space; a single
// short row in one glyph corrupts every fixture that uses the
// letter. We assert:
//
//   - Both tables cover the same rune set (renderer falls back to
//     '?' on a miss so a missing key would silently diverge).
//   - Every glyph has exactly bannerRows rows.
//   - All 6 rows of a given glyph have identical display width —
//     counted in runes, not bytes, since ANSI-shadow glyphs use
//     multi-byte runes like █ ═ ╔ ╗ ╚ ╝. Width equality is what
//     makes the direct concatenation in renderBanner lay out as
//     a clean grid; trailing-space counts are NOT required to
//     match across the 6 rows (letters like Y / V / Z have
//     intentionally ragged trails whose leading+visible+trailing
//     still sum to the rune-width, so the rectangle holds).
//   - No tabs anywhere — they'd render unpredictably on terminals.
func TestBannerFontTables(t *testing.T) {
	if len(bannerFontDefault) != len(bannerFontASCII) {
		t.Fatalf("font tables disagree on size: default=%d ascii=%d",
			len(bannerFontDefault), len(bannerFontASCII))
	}
	for r := range bannerFontDefault {
		if _, ok := bannerFontASCII[r]; !ok {
			t.Errorf("rune %q present in default but missing in ascii", r)
		}
	}
	for r := range bannerFontASCII {
		if _, ok := bannerFontDefault[r]; !ok {
			t.Errorf("rune %q present in ascii but missing in default", r)
		}
	}

	checkTable := func(t *testing.T, name string, table map[rune][6]string) {
		t.Helper()
		for r, glyph := range table {
			// [6]string is compile-time guaranteed to have 6 entries,
			// but assert via the constant to document intent and
			// catch a future refactor that might flip the type.
			if len(glyph) != bannerRows {
				t.Errorf("%s[%q]: got %d rows, want %d", name, r, len(glyph), bannerRows)
				continue
			}
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
	}
	checkTable(t, "bannerFontDefault", bannerFontDefault)
	checkTable(t, "bannerFontASCII", bannerFontASCII)
}
