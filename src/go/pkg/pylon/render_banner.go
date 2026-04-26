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
// Font selection has two axes. The frontmatter `banner:` key picks the
// family (default vs. monospace); when monospace is selected the
// theme-driven `bc` only decides whether to substitute `█→#` on the
// finalized rows. When banner is unset the existing default↔ascii pair
// is consulted as before.
func renderBanner(b *Box, bc boxChars) []string {
	monospace := b.Meta.Banner == "monospace"

	var font map[rune][6]string
	switch {
	case monospace:
		font = bannerFontMonospace
	case bc == asciiBox:
		font = bannerFontASCII
	default:
		font = bannerFontDefault
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
	if monospace && bc == asciiBox {
		// Pure-`█` palette → total-function substitution to `#`. Done
		// once per finalized row instead of per glyph.
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], "█", "#")
		}
	}
	return out
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
