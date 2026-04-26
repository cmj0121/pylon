package pylon

import "strings"

// bannerRows is the fixed height of a banner renderer's output.
// Every glyph in all three font tables (default, ascii, monospace)
// is exactly this many rows so the concatenation below is a straight
// row-by-row append.
const bannerRows = 6

// renderBanner turns a box's literal text content into block letters.
// Input is uppercased before lookup; unknown runes fall back to the
// '?' glyph so fixtures stay deterministic.
//
// Font selection follows a three-tier resolution order:
//  1. Per-box renderer arg `[ ... | banner:FONT ]` (highest priority).
//  2. Frontmatter `banner: FONT` (image-wide default).
//  3. Theme dispatch — ANSI Shadow under default, `#`+space under ascii.
//
// Recognized FONT values: `monospace`, `digital`, `mini`. Unknown values
// silently fall through to the next tier (matching `theme:`/`color:`
// silent-ignore policy). The three pure-`█` fonts share a row-level
// `█→#` substitution that derives the ascii-theme variant on the fly.
func renderBanner(b *Box, bc boxChars) []string {
	font, pureBlock := pickBannerFont(b.RendererArg, b.Meta.Banner)
	if !pureBlock {
		if bc == asciiBox {
			font = bannerFontASCII
		} else {
			font = bannerFontDefault
		}
	}

	var sb strings.Builder
	collectBoxText(b, &sb)
	text := strings.ToUpper(sb.String())

	if text == "" {
		return []string{"", "", "", "", "", ""}
	}

	rows := make([]strings.Builder, bannerRows)
	for _, r := range text {
		glyph, ok := font[r]
		if !ok {
			glyph = font['?']
		}
		for i := 0; i < bannerRows; i++ {
			rows[i].WriteString(glyph[i])
		}
	}

	out := make([]string, bannerRows)
	for i := range rows {
		out[i] = rows[i].String()
	}
	if pureBlock && bc == asciiBox {
		// Pure-`█` palette → total-function substitution to `#`. Done
		// once per finalized row instead of per glyph.
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], "█", "#")
		}
	}
	return out
}

// pickBannerFont resolves the per-box arg and frontmatter `banner:` key
// into a font table. Returns (table, true) when a pure-`█` font was
// selected (caller does the `█→#` substitution under ascii theme), and
// (nil, false) when neither tier specified a known font (caller falls
// back to the default↔ascii pair).
func pickBannerFont(boxArg, metaBanner string) (map[rune][6]string, bool) {
	for _, name := range []string{boxArg, metaBanner} {
		switch name {
		case "monospace":
			return bannerFontMonospace, true
		case "digital":
			return bannerFontDigital, true
		case "mini":
			return bannerFontMini, true
		}
	}
	return nil, false
}

// collectBoxText concatenates the Content of every *Text reachable
// through b's Items and direct Row children. Shared by banner
// (uppercases and looks up glyphs) and progress (parses as number);
// Refs, DataRefs, nested Boxes, and Edges are ignored.
func collectBoxText(b *Box, sb *strings.Builder) {
	for _, it := range b.Items {
		switch x := it.(type) {
		case *Text:
			sb.WriteString(x.Content)
		case *Row:
			for _, rit := range x.Items {
				if t, ok := rit.(*Text); ok {
					sb.WriteString(t.Content)
				}
			}
		}
	}
}
