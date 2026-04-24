package pylon

import (
	"fmt"
	"math"
	"strings"
)

// Gantt layout constants. Default budget (bar-area cells) is 20;
// size-aware layout shrinks or grows it to max(5, W - labelW - 4)
// when the outer box carries a `size: WxH` frontmatter. The `4`
// accounts for 3 left-pad cells + 1 label/bar gap, matching JS.
const (
	ganttBudgetDefault = 20
	ganttBudgetMin     = 5
	ganttSizePadding   = 4
)

// ganttGlyph picks the bar fill for bc's theme. Default is the full
// block; ASCII folds to `#`. Same dispatch shape as histGlyph.
func ganttGlyph(bc boxChars) string {
	if bc == asciiBox {
		return "#"
	}
	return barGlyph
}

// renderGantt lays out task bars on a shared budget. One row per
// entry: `{label} {bar-area}`, where the bar for task i spans cells
// round(start/maxEnd*budget) through round(end/maxEnd*budget)-1. The
// validator has already verified all entries carry {x, start, end}
// as the expected types, so assertions are infallible.
//
// Budget is size-aware when meta.Size is set: budget = max(5, W -
// labelW - 4). With no size, it falls back to 20.
func renderGantt(series []map[string]interface{}, bc boxChars, meta Meta) []string {
	glyph := ganttGlyph(bc)
	n := len(series)
	labels := make([]string, n)
	starts := make([]float64, n)
	ends := make([]float64, n)
	labelW := 0
	maxEnd := 0.0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		s, _ := e["start"].(float64)
		en, _ := e["end"].(float64)
		starts[i] = s
		ends[i] = en
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
		if en > maxEnd {
			maxEnd = en
		}
	}

	budget := ganttBudgetDefault
	if meta.Size != nil {
		b := meta.Size.W - labelW - ganttSizePadding
		if b < ganttBudgetMin {
			b = ganttBudgetMin
		}
		budget = b
	}

	out := make([]string, n)
	for i := 0; i < n; i++ {
		var barStart, barEnd int
		if maxEnd > 0 {
			barStart = int(math.Round(starts[i] / maxEnd * float64(budget)))
			barEnd = int(math.Round(ends[i]/maxEnd*float64(budget))) - 1
		} else {
			barStart = 0
			barEnd = -1
		}
		var sb strings.Builder
		for c := 0; c < budget; c++ {
			if c >= barStart && c <= barEnd {
				sb.WriteString(glyph)
			} else {
				sb.WriteString(" ")
			}
		}
		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + sb.String()
	}
	return out
}

// validateGanttSeries checks `[{x, start, end}]` shape. Entries must
// carry all three fields; start and end must be finite non-NaN
// float64 values. Range checks (start >= 0, end >= start) emit
// CodeGanttRange so the "shape vs range" distinction stays
// explicit. Empty shares CodeBarEmpty with the bar family for
// consistency.
func validateGanttSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ gantt: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x, start, end}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x, start, end}]")
		}
		_, hasX := entry["x"]
		sRaw, hasS := entry["start"]
		eRaw, hasE := entry["end"]
		if !hasX || !hasS || !hasE {
			return emit(CodeBarShape, "expected [{x, start, end}]")
		}
		sNum, sOk := sRaw.(float64)
		eNum, eOk := eRaw.(float64)
		if !sOk || !eOk || math.IsNaN(sNum) || math.IsNaN(eNum) {
			return emit(CodeBarShape, "expected [{x, start, end}]")
		}
	}
	// Range is a separate pass so shape errors surface ahead of
	// range errors across the whole series.
	for _, entry := range arr {
		sNum, _ := entry["start"].(float64)
		eNum, _ := entry["end"].(float64)
		if sNum < 0 || eNum < sNum {
			return emit(CodeGanttRange, "invalid range")
		}
	}
	return "", "", false
}
