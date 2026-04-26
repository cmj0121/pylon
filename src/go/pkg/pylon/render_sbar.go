package pylon

import (
	"fmt"
	"math"
	"strings"
)

// sbarBudget is the horizontal cell budget for stacked-bar rendering.
// Matches barWidthDefault so single-bar (`bar`) and stacked-bar
// (`sbar`) sit on the same horizontal grid when a fixture stacks
// them together.
const sbarBudget = 10

// renderSBar lays out a horizontal stacked bar: each row gets one
// stacked bar made of the heatmap shade ramp (one ramp slot per
// y-array element). Total bar width is normalized to sbarBudget cells
// based on the row's sum, scaled against the largest row sum.
//
// Row shape:  "{label} {│} {stack}{value} {│}" — same dressing as
// hbar so the two renderers compose visually.
//
// Validator has already ruled out non-numeric / negative y, so the
// type assertions here are infallible.
func renderSBar(series []map[string]interface{}, bc boxChars) []string {
	glyphs := heatmapGlyphs(bc)
	rampLevels := len(glyphs) - 1 // 0 == blank, 1..rampLevels are the fills

	n := len(series)
	labels := make([]string, n)
	rows := make([][]float64, n)
	sums := make([]float64, n)
	maxSum := 0.0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		ys, _ := e["y"].([]float64)
		rows[i] = ys
		s := 0.0
		for _, v := range ys {
			s += v
		}
		sums[i] = s
		if s > maxSum {
			maxSum = s
		}
	}

	labelW := 0
	valueW := 0
	values := make([]string, n)
	for i := range series {
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
		values[i] = "(" + formatY(sums[i]) + ")"
		if w := displayWidth(values[i]); w > valueW {
			valueW = w
		}
	}

	out := make([]string, n)
	for i, ys := range rows {
		// Total stacked width for this row, scaled to budget.
		totalCells := 0
		if maxSum > 0 {
			totalCells = int(math.Round(sums[i] / maxSum * float64(sbarBudget)))
		}
		// Distribute totalCells across the y entries proportional to
		// their value within the row, picking ramp slots round-robin
		// through the ramp so adjacent segments visually separate.
		rowSum := sums[i]
		var stack strings.Builder
		used := 0
		for j, v := range ys {
			share := 0
			if rowSum > 0 {
				share = int(math.Round(v / rowSum * float64(totalCells)))
			}
			if used+share > totalCells {
				share = totalCells - used
			}
			if share < 0 {
				share = 0
			}
			ramp := (j % rampLevels) + 1 // 1..rampLevels
			stack.WriteString(strings.Repeat(glyphs[ramp], share))
			used += share
		}
		// Pad the visual bar to the budget so trailing values align.
		body := stack.String() + strings.Repeat(" ", sbarBudget)
		body = clipRow(body, sbarBudget+1)
		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + bc.v + " " +
			body + padRow(values[i], valueW, AlignRight, 0)
	}
	return out
}

// validateSBarSeries reuses heatmap's `[{x, y:[n,...]}]` shape: every
// entry has a numeric-array y, all rows non-negative. Wire prefix is
// "sbar:". Empty `y` arrays are tolerated (the row stacks to zero).
func validateSBarSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ sbar: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
		}
		_, hasX := entry["x"]
		yRaw, hasY := entry["y"]
		if !hasX || !hasY {
			return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
		}
		ys, ok := yRaw.([]float64)
		if !ok {
			return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
		}
		for _, v := range ys {
			if math.IsNaN(v) {
				return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
			}
			if v < 0 {
				return emit(CodeBarNegativeY, "negative y")
			}
		}
	}
	return "", "", false
}
