package pylon

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Progress layout constants. Budget is fixed at 20 cells per row;
// the glyph pair is swapped under `theme: ascii`.
const (
	progressBudget        = 20
	progressFilledDefault = "█"
	progressEmptyDefault  = "░"
	progressFilledASCII   = "#"
	progressEmptyASCII    = "."
)

// progressGlyphs picks the filled / empty cell glyphs for bc's theme.
// Mirrors renderBanner's `bc == asciiBox` check.
func progressGlyphs(bc boxChars) (filled, empty string) {
	if bc == asciiBox {
		return progressFilledASCII, progressEmptyASCII
	}
	return progressFilledDefault, progressEmptyDefault
}

// progressKind picks the semantic ANSI palette slot for a pct value.
// Thresholds split the 0..100 range into bear (<40), doji (40..79),
// and bull (>=80) bands so the visual reads as "bad / warning / good"
// without further configuration. Same cutoffs as the JS reference.
func progressKind(pct float64) semKind {
	switch {
	case pct >= 80:
		return semBull
	case pct >= 40:
		return semDoji
	default:
		return semBear
	}
}

// renderProgressScalar renders a `[ N | progress ]` box as a single
// row. The validator has already verified the body parses as a
// number, so any re-parse failure renders as 0 %. Under `theme: ascii`
// color is force-suppressed so ASCII output stays plain.
func renderProgressScalar(b *Box, bc boxChars, colorOn bool) []string {
	if bc == asciiBox {
		colorOn = false
	}
	filled, empty := progressGlyphs(bc)
	y, _ := parseProgressScalar(b)
	return []string{renderProgressRow(y, filled, empty, colorOn)}
}

// renderProgressSeries lays out N rows for a `[ @ref | progress ]`
// box: `{label} {bar-20} {pct-4}` per row, single-space separators,
// label column right-padded to the widest entry. Under `theme: ascii`
// color is force-suppressed so ASCII output stays plain.
func renderProgressSeries(series []map[string]interface{}, bc boxChars, colorOn bool) []string {
	if bc == asciiBox {
		colorOn = false
	}
	filled, empty := progressGlyphs(bc)
	labels := make([]string, len(series))
	ys := make([]float64, len(series))
	labelW := 0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		y, _ := e["y"].(float64)
		ys[i] = y
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
	}
	out := make([]string, len(series))
	for i := range series {
		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + renderProgressRow(ys[i], filled, empty, colorOn)
	}
	return out
}

// renderProgressRow formats `{bar-20}{space}{pct-4}`. `%3d%%` right-
// aligns the percent in 4 cells so "100%" sits flush while " 75%"
// and "  0%" lead with spaces. Filled cells take the semantic colour
// picked by progressKind; empty cells stay plain so the background
// of the bar reads as scaffolding rather than content.
func renderProgressRow(y float64, filled, empty string, colorOn bool) string {
	pct := clampPercent(y)
	cells := int(math.Round(pct / 100 * float64(progressBudget)))
	filledRun := strings.Repeat(filled, cells)
	if colorOn && cells > 0 {
		filledRun = wrapANSI(filledRun, progressKind(pct), true)
	}
	bar := filledRun + strings.Repeat(empty, progressBudget-cells)
	return bar + " " + fmt.Sprintf("%3d%%", int(math.Round(pct)))
}

// clampPercent silently folds y into [0, 100]; NaN collapses to 0
// so a malformed fixture prints "0%" rather than "NaN%".
func clampPercent(y float64) float64 {
	if math.IsNaN(y) || y < 0 {
		return 0
	}
	if y > 100 {
		return 100
	}
	return y
}

// parseProgressScalar reads the concatenated Text content of b as a
// float. Returns (0, false) on empty, unparseable, NaN, or Inf. Used
// by both the validator (to surface CodeProgressNotNumber) and the
// renderer (to read the already-validated value).
func parseProgressScalar(b *Box) (float64, bool) {
	var sb strings.Builder
	collectBoxText(b, &sb)
	raw := strings.TrimSpace(sb.String())
	if raw == "" {
		return 0, false
	}
	y, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(y) || math.IsInf(y, 0) {
		return 0, false
	}
	return y, true
}
