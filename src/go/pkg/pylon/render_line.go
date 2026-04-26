package pylon

import (
	"math"
	"strings"
)

// Line / area / scatter share the basic [{x,y}] shape from sparkline.
// Each renderer's wire prefix is its own user-facing name.

const (
	lineHeight    = 7  // visible rows in the line / area grid
	scatterHeight = 7  // visible rows in scatter (matches line)
	scatterWidth  = 30 // x-axis cells in scatter
)

// lineGlyphs bundles the markers and connectors needed to draw a
// connected line plot. ASCII swaps Unicode box-drawing for plain
// ASCII; output stays cell-for-cell identical to the JS reference.
type lineGlyphs struct {
	point, horiz, up, down, vert string
}

var (
	lineGlyphsDefault = lineGlyphs{
		point: "●", horiz: "─",
		up: "╱", down: "╲", vert: "│",
	}
	lineGlyphsASCII = lineGlyphs{
		point: "o", horiz: "-",
		up: "/", down: "\\", vert: "|",
	}
)

func lineGlyphRamp(bc boxChars) lineGlyphs {
	if bc == asciiBox {
		return lineGlyphsASCII
	}
	return lineGlyphsDefault
}

// renderLine plots discrete y values on a fixed-height grid with
// box-drawing connectors between adjacent points. Width = 2n-1 cells
// (one per point + one transition between neighbors). A flat series
// (all y equal) lays every point on the bottom row.
//
// Connector rule between point i (row r_i) and point i+1 (row r_{i+1}):
//   - r_i == r_{i+1}: horizontal between them
//   - |r_i - r_{i+1}| == 1: diagonal (up or down)
//   - |r_i - r_{i+1}| > 1:  vertical bar
//
// Validator already ruled out shape / empty / NaN, so type assertions
// are infallible.
func renderLine(series []map[string]interface{}, bc boxChars) []string {
	g := lineGlyphRamp(bc)
	n := len(series)
	ys := make([]float64, n)
	for i, e := range series {
		y, _ := e["y"].(float64)
		ys[i] = y
	}

	gMin, gMax := ys[0], ys[0]
	for _, v := range ys[1:] {
		if v < gMin {
			gMin = v
		}
		if v > gMax {
			gMax = v
		}
	}

	H := lineHeight
	rowFor := func(y float64) int {
		if gMax == gMin {
			return H - 1
		}
		return (H - 1) - int(math.Round((y-gMin)/(gMax-gMin)*float64(H-1)))
	}

	rows := make([]int, n)
	for i, y := range ys {
		rows[i] = rowFor(y)
	}

	width := 2*n - 1
	if width < 1 {
		width = 1
	}
	grid := make([][]string, H)
	for r := 0; r < H; r++ {
		grid[r] = make([]string, width)
		for c := 0; c < width; c++ {
			grid[r][c] = " "
		}
	}

	for i := 0; i < n; i++ {
		grid[rows[i]][2*i] = g.point
		if i == n-1 {
			continue
		}
		cur, nxt := rows[i], rows[i+1]
		col := 2*i + 1
		switch {
		case cur == nxt:
			grid[cur][col] = g.horiz
		case cur > nxt && cur-nxt == 1:
			grid[cur][col] = g.up
		case cur < nxt && nxt-cur == 1:
			grid[cur][col] = g.down
		default:
			midRow := (cur + nxt) / 2
			grid[midRow][col] = g.vert
		}
	}

	out := make([]string, H)
	for r := 0; r < H; r++ {
		out[r] = strings.Join(grid[r], "")
	}
	return out
}

// renderArea fills every column from the data point down to the
// baseline with a five-step ramp (heatmap glyphs). Higher y → taller
// solid stack. Reuses the heatmap glyph table so the visual language
// matches across renderers and theme: ascii produces a printable
// fallback automatically.
//
// Each entry → 1 column. Width = n. Height = lineHeight rows.
func renderArea(series []map[string]interface{}, bc boxChars) []string {
	glyphs := heatmapGlyphs(bc)
	rampLevels := len(glyphs) - 1 // 0 == blank; 1..rampLevels are filled
	n := len(series)
	ys := make([]float64, n)
	for i, e := range series {
		y, _ := e["y"].(float64)
		ys[i] = y
	}

	gMin, gMax := ys[0], ys[0]
	for _, v := range ys[1:] {
		if v < gMin {
			gMin = v
		}
		if v > gMax {
			gMax = v
		}
	}

	H := lineHeight
	// Each entry occupies one column, height = round((y-gMin)/(gMax-gMin) * H).
	heights := make([]int, n)
	for i, y := range ys {
		if gMax == gMin {
			heights[i] = 1
			continue
		}
		h := int(math.Round((y - gMin) / (gMax - gMin) * float64(H)))
		if h < 0 {
			h = 0
		}
		if h > H {
			h = H
		}
		heights[i] = h
	}

	out := make([]string, H)
	for r := 0; r < H; r++ {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			fromBottom := H - r // 1 = bottom row, H = top row
			if heights[i] >= fromBottom {
				// Inside the filled area: pick ramp shade based on how
				// close this row is to the top of the column. Top cell
				// of a tall column = full block; lower cells = lighter
				// ramp. This gives a soft fade-in from the baseline.
				delta := heights[i] - fromBottom
				idx := rampLevels - delta
				if idx < 1 {
					idx = 1
				}
				if idx > rampLevels {
					idx = rampLevels
				}
				sb.WriteString(glyphs[idx])
			} else {
				sb.WriteString(glyphs[0])
			}
		}
		out[r] = sb.String()
	}
	return out
}

// renderScatter places one dot per series entry on a 2D grid. The
// x-axis is normalized over scatterWidth cells, y-axis over
// scatterHeight rows. Multiple entries may collide on the same cell;
// the last write wins. A flat series (single x or single y) collapses
// onto a single row or column.
func renderScatter(series []map[string]interface{}, bc boxChars) []string {
	g := lineGlyphRamp(bc)
	n := len(series)
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i, e := range series {
		// Scatter x is treated numerically when possible; stringly-typed
		// labels are normalized to their position index 0..n-1 (matches
		// JS Number(label) || index fallback).
		switch v := e["x"].(type) {
		case float64:
			xs[i] = v
		default:
			xs[i] = float64(i)
		}
		y, _ := e["y"].(float64)
		ys[i] = y
	}

	xMin, xMax := xs[0], xs[0]
	yMin, yMax := ys[0], ys[0]
	for i := 1; i < n; i++ {
		if xs[i] < xMin {
			xMin = xs[i]
		}
		if xs[i] > xMax {
			xMax = xs[i]
		}
		if ys[i] < yMin {
			yMin = ys[i]
		}
		if ys[i] > yMax {
			yMax = ys[i]
		}
	}

	W := scatterWidth
	H := scatterHeight
	grid := make([][]string, H)
	for r := 0; r < H; r++ {
		grid[r] = make([]string, W)
		for c := 0; c < W; c++ {
			grid[r][c] = " "
		}
	}

	for i := 0; i < n; i++ {
		col := 0
		if xMax > xMin {
			col = int(math.Round((xs[i] - xMin) / (xMax - xMin) * float64(W-1)))
		}
		row := H - 1
		if yMax > yMin {
			row = (H - 1) - int(math.Round((ys[i]-yMin)/(yMax-yMin)*float64(H-1)))
		}
		if col < 0 {
			col = 0
		}
		if col >= W {
			col = W - 1
		}
		if row < 0 {
			row = 0
		}
		if row >= H {
			row = H - 1
		}
		grid[row][col] = g.point
	}

	out := make([]string, H)
	for r := 0; r < H; r++ {
		out[r] = strings.Join(grid[r], "")
	}
	return out
}

// validateXYSeries is shared by line / area / scatter — each accepts
// the same shape (`[{x,y}]` with numeric y, negative-tolerated,
// duplicate-x tolerated) but emits its own user-typed prefix. The
// `name` parameter threads through to the wire text.
func validateXYSeries(name string, series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ " + name + ": " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x,y}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
		_, hasX := entry["x"]
		yRaw, hasY := entry["y"]
		if !hasX || !hasY {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
		yNum, isNum := yRaw.(float64)
		if !isNum || math.IsNaN(yNum) {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
	}
	return "", "", false
}
