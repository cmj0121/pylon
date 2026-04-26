package pylon

import (
	"fmt"
	"math"
	"strings"
)

// boxBudget is the horizontal cell budget for a box-plot row. 20 cells
// gives whiskers + box + median enough resolution while staying
// terminal-friendly.
const boxBudget = 20

// boxGlyphs collects the four-layer box-plot glyphs. ASCII folds the
// box-drawing borders to printable equivalents and the IQR fill to
// `#` so terminals without box-drawing chars still produce a
// recognizable plot.
type boxGlyphs struct {
	whisker, edgeL, edgeR, fill, median string
}

var (
	boxGlyphsDefault = boxGlyphs{
		whisker: "─", edgeL: "┤", edgeR: "├",
		fill: "▓", median: "█",
	}
	boxGlyphsASCII = boxGlyphs{
		whisker: "-", edgeL: "[", edgeR: "]",
		fill: "#", median: "X",
	}
)

func boxGlyphRamp(bc boxChars) boxGlyphs {
	if bc == asciiBox {
		return boxGlyphsASCII
	}
	return boxGlyphsDefault
}

// renderBox lays out a horizontal box-plot row per series entry.
// Entry shape: `{x, min, q1, med, q3, max}`. All five numbers map
// onto a global scale (gMin..gMax across all rows) so multiple rows
// render comparably. Layout per row, from left to right:
//
//	whiskers ── edgeL ▓...▓ │ ▓...▓ edgeR ──── whiskers
//	         min        q1   med   q3         max
//
// Each cell is one glyph wide. The renderer collapses degenerate
// shapes (q1 == q3, etc.) to the minimum visually-readable widths
// (the box stays at least 3 cells wide so the median strip fits).
func renderBox(series []map[string]interface{}, bc boxChars) []string {
	g := boxGlyphRamp(bc)
	n := len(series)
	labels := make([]string, n)
	mins := make([]float64, n)
	q1s := make([]float64, n)
	meds := make([]float64, n)
	q3s := make([]float64, n)
	maxs := make([]float64, n)
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		mins[i], _ = e["min"].(float64)
		q1s[i], _ = e["q1"].(float64)
		meds[i], _ = e["med"].(float64)
		q3s[i], _ = e["q3"].(float64)
		maxs[i], _ = e["max"].(float64)
	}

	gMin, gMax := mins[0], maxs[0]
	for i := 0; i < n; i++ {
		if mins[i] < gMin {
			gMin = mins[i]
		}
		if maxs[i] > gMax {
			gMax = maxs[i]
		}
	}

	budget := boxBudget
	cellOf := func(v float64) int {
		if gMax == gMin {
			return budget / 2
		}
		c := int(math.Round((v - gMin) / (gMax - gMin) * float64(budget-1)))
		if c < 0 {
			c = 0
		}
		if c >= budget {
			c = budget - 1
		}
		return c
	}

	labelW := 0
	for _, lab := range labels {
		if w := displayWidth(lab); w > labelW {
			labelW = w
		}
	}

	out := make([]string, n)
	for i := 0; i < n; i++ {
		cMin := cellOf(mins[i])
		cQ1 := cellOf(q1s[i])
		cMed := cellOf(meds[i])
		cQ3 := cellOf(q3s[i])
		cMax := cellOf(maxs[i])
		if cQ3 < cQ1 {
			cQ1, cQ3 = cQ3, cQ1
		}

		row := make([]string, budget)
		for c := 0; c < budget; c++ {
			row[c] = " "
		}
		// Whiskers.
		for c := cMin; c <= cQ1 && c < budget; c++ {
			row[c] = g.whisker
		}
		for c := cQ3; c <= cMax && c < budget; c++ {
			row[c] = g.whisker
		}
		// IQR fill.
		for c := cQ1; c <= cQ3 && c < budget; c++ {
			row[c] = g.fill
		}
		// Box edges + median strip override the fill.
		row[cQ1] = g.edgeL
		row[cQ3] = g.edgeR
		row[cMed] = g.median

		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + bc.v + " " +
			strings.Join(row, "")
	}
	return out
}

// validateBoxSeries enforces the 5-number summary shape:
//
//	[{x, min, q1, med, q3, max}] with min ≤ q1 ≤ med ≤ q3 ≤ max
//
// All five numeric fields are required. Wire prefix is "box:".
func validateBoxSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ box: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x, min, q1, med, q3, max}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	keys := []string{"min", "q1", "med", "q3", "max"}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x, min, q1, med, q3, max}]")
		}
		if _, ok := entry["x"]; !ok {
			return emit(CodeBarShape, "expected [{x, min, q1, med, q3, max}]")
		}
		vals := make([]float64, len(keys))
		for i, k := range keys {
			raw, has := entry[k]
			if !has {
				return emit(CodeBarShape, "expected [{x, min, q1, med, q3, max}]")
			}
			num, isNum := raw.(float64)
			if !isNum || math.IsNaN(num) {
				return emit(CodeBarShape, "expected [{x, min, q1, med, q3, max}]")
			}
			vals[i] = num
		}
		for i := 1; i < len(vals); i++ {
			if vals[i] < vals[i-1] {
				return emit(CodeBarShape, "invalid order: min ≤ q1 ≤ med ≤ q3 ≤ max")
			}
		}
	}
	return "", "", false
}
