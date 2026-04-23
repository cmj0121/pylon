package pylon

import "strings"

// bannerRows is the fixed height of a banner renderer's output.
// Every glyph in both font tables is exactly this many rows so the
// concatenation below is a straight row-by-row append.
const bannerRows = 6

// renderBanner turns a box's literal text content into block letters.
// Theme pick is driven by bc.theme (set upstream in boxCharsFor) so
// the caller keeps its one-arg signature. DataRef children are
// silently dropped in v1 — see PLAN.md for the parity-drift note.
//
// Input is every *Text's Content, including those nested inside the
// box's Rows. The result is uppercased; unknown runes fall back to
// the '?' glyph rather than aborting so fixtures stay deterministic.
func renderBanner(b *Box, bc boxChars) []string {
	font := bannerFontDefault
	if bc.theme == "ascii" {
		font = bannerFontASCII
	}

	var sb strings.Builder
	collectBannerText(b, &sb)
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
	return out
}

// collectBannerText concatenates the Content of every *Text reachable
// from b via its Items or nested Rows. Refs, DataRefs, nested Boxes,
// and Edges are ignored in v1 — the banner primitive is literal-
// string only.
func collectBannerText(b *Box, sb *strings.Builder) {
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
