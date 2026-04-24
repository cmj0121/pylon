package pylon

import (
	"fmt"
	"math"
	"strings"
)

// histHeightDefault is the fixed row budget for `| hist`. Matches JS.
// Column width is 1 (no gap between bars); an optional footer row
// carries x labels when any label is non-empty.
const histHeightDefault = 8

// histGlyph picks the bar fill for bc's theme. Default is the full
// block (U+2588); ASCII folds to `#`. Mirrors the JS ramp picker so
// output stays byte-identical across sides.
func histGlyph(bc boxChars) string {
	if bc == asciiBox {
		return "#"
	}
	return barGlyph
}

// renderHist lays out a histogram: fixed-height grid of single-cell
// bars with no inter-column gap. Footer appears only when at least one
// x label is non-empty; when it appears, column width expands to the
// widest label (min 1). Validation (bar-family shape/empty/negative/
// duplicate) has already run upstream via barSeriesInlineError, so
// type assertions here are infallible.
func renderHist(series []map[string]interface{}, bc boxChars) []string {
	glyph := histGlyph(bc)
	n := len(series)
	labels := make([]string, n)
	ys := make([]float64, n)
	hasLabel := false
	labelW := 1
	maxY := 0.0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		if labels[i] != "" {
			hasLabel = true
		}
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
		y, _ := e["y"].(float64)
		ys[i] = y
		if y > maxY {
			maxY = y
		}
	}

	// When no x labels appear the grid stays tight (1 cell per bar);
	// with labels, every column widens to labelW and the bar glyph is
	// centred within it (left-biased by padRow when slack is odd).
	colW := 1
	if hasLabel {
		colW = labelW
	}

	cells := make([]int, n)
	for i, y := range ys {
		if maxY > 0 {
			cells[i] = int(math.Round(y / maxY * float64(histHeightDefault)))
		}
	}

	rows := make([]string, 0, histHeightDefault+1)
	for r := 0; r < histHeightDefault; r++ {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			content := " "
			if cells[i] >= histHeightDefault-r {
				content = glyph
			}
			if colW == 1 {
				sb.WriteString(content)
			} else {
				sb.WriteString(padRow(content, colW, AlignCenter, 0))
			}
		}
		rows = append(rows, sb.String())
	}
	if hasLabel {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			sb.WriteString(padRow(labels[i], colW, AlignCenter, 0))
		}
		rows = append(rows, sb.String())
	}
	return rows
}
