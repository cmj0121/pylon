package pylon

import (
	"fmt"
	"math"
	"strings"
)

// Heatmap glyph ramps. Five levels: blank + four fills. ASCII theme
// swaps in a printable-only set so terminals without box-drawing
// glyphs still produce a monotone ramp. Constants mirror
// src/js/pylon.js so ASCII output stays byte-identical across sides.
var (
	heatmapGlyphsDefault = []string{" ", "░", "▒", "▓", "█"}
	heatmapGlyphsASCII   = []string{" ", ".", "+", "*", "#"}
)

// heatmapGlyphs picks the 5-level ramp for bc's theme. Same shape as
// progressGlyphs — a single bc check drives the swap.
func heatmapGlyphs(bc boxChars) []string {
	if bc == asciiBox {
		return heatmapGlyphsASCII
	}
	return heatmapGlyphsDefault
}

// renderHeatmap lays out a labeled 2D matrix. Each row is `{label}
// {grid}`; the grid is one glyph per cell, no separators. Label
// column is suppressed when every label is the empty string. The
// validator has ruled out ragged rows / negative values / non-number
// entries, so type assertions here are infallible.
func renderHeatmap(series []map[string]interface{}, bc boxChars) []string {
	glyphs := heatmapGlyphs(bc)
	labels := make([]string, len(series))
	rows := make([][]float64, len(series))
	hasLabel := false
	maxV := 0.0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		if labels[i] != "" {
			hasLabel = true
		}
		ys, _ := e["y"].([]float64)
		rows[i] = ys
		for _, v := range ys {
			if v > maxV {
				maxV = v
			}
		}
	}

	labelW := 0
	if hasLabel {
		for _, lab := range labels {
			if w := displayWidth(lab); w > labelW {
				labelW = w
			}
		}
	}

	out := make([]string, len(series))
	for i, ys := range rows {
		var grid strings.Builder
		for _, v := range ys {
			idx := 0
			if maxV > 0 {
				idx = int(math.Round(v / maxV * 4))
			}
			grid.WriteString(glyphs[idx])
		}
		if hasLabel {
			out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + grid.String()
		} else {
			out[i] = grid.String()
		}
	}
	return out
}

// validateHeatmapSeries checks the resolved series matches the
// `[{x, y:[n,...]}]` shape: every row carries a numeric-array `y`,
// all rows share a width, and every value is non-negative. Returns
// ok=false on clean input; otherwise the diagnostic tuple that
// rendererInlineError wraps into wire text. Ragged widths and non-
// numeric / NaN y entries collapse to CodeHeatmapShape — the model
// treats them as one shape class.
func validateHeatmapSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ heatmap: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeHeatmapShape, "expected [{x, y:[n,...]}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	width := -1
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
		if width == -1 {
			width = len(ys)
		} else if len(ys) != width {
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
