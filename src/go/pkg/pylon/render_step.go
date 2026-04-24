package pylon

import (
	"math"
)

// Step layout constants. Height is fixed at 5 rows; each entry
// occupies two cells (run + transition connector). Totals therefore
// come out as H=5 rows by 2n-1 cells wide; matches JS.
const stepHeightDefault = 5

// stepGlyphs bundles the six glyphs the cumulative-step line needs,
// themed. ASCII folds every corner to `+` since plain ASCII has no
// dedicated box-drawing glyphs.
type stepGlyphs struct {
	horiz, vert      string
	cornerTL, cornerTR,
	cornerBL, cornerBR string
}

var (
	stepGlyphsDefault = stepGlyphs{
		horiz: "─", vert: "│",
		cornerTL: "┌", cornerTR: "┐",
		cornerBL: "└", cornerBR: "┘",
	}
	stepGlyphsASCII = stepGlyphs{
		horiz: "-", vert: "|",
		cornerTL: "+", cornerTR: "+",
		cornerBL: "+", cornerBR: "+",
	}
)

// stepGlyphRamp picks the glyph set for bc's theme. Same dispatch
// shape as sparklineGlyphs.
func stepGlyphRamp(bc boxChars) stepGlyphs {
	if bc == asciiBox {
		return stepGlyphsASCII
	}
	return stepGlyphsDefault
}

// renderStep lays out a cumulative step line. Each entry is two
// cells wide: the horizontal run at the entry's row, and a
// transition column to the next entry's row. The last entry has
// no transition — just the horizontal run.
//
// Normalisation is global: row = (H-1) - round((y - gMin) / (gMax -
// gMin) * (H-1)), so higher y sits visually higher (lower row
// index). A flat series (gMin == gMax) collapses every entry to the
// bottom row.
//
// The validator has already ruled out shape / empty, so type
// assertions are infallible. Negative y and duplicate x are tolerated
// (trend viz, like sparkline).
func renderStep(series []map[string]interface{}, bc boxChars) []string {
	g := stepGlyphRamp(bc)
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

	H := stepHeightDefault
	rowFor := func(y float64) int {
		if gMax == gMin {
			return H - 1
		}
		return (H - 1) - int(math.Round((y-gMin)/(gMax-gMin)*float64(H-1)))
	}

	rowsOfY := make([]int, n)
	for i, y := range ys {
		rowsOfY[i] = rowFor(y)
	}

	width := 2*n - 1
	if width < 1 {
		width = 1
	}
	// Initialise grid as row-major [][]string cells so every (r, c)
	// either carries exactly one glyph or a single space.
	grid := make([][]string, H)
	for r := 0; r < H; r++ {
		grid[r] = make([]string, width)
		for c := 0; c < width; c++ {
			grid[r][c] = " "
		}
	}

	for i := 0; i < n; i++ {
		runCol := 2 * i
		grid[rowsOfY[i]][runCol] = g.horiz
		if i == n-1 {
			continue
		}
		transCol := 2*i + 1
		cur := rowsOfY[i]
		nxt := rowsOfY[i+1]
		switch {
		case cur == nxt:
			grid[cur][transCol] = g.horiz
		case cur < nxt:
			// Row index increases == y value stepped DOWN. Top of the
			// connector sits at `cur`, bottom at `nxt`.
			grid[cur][transCol] = g.cornerTR
			for r := cur + 1; r < nxt; r++ {
				grid[r][transCol] = g.vert
			}
			grid[nxt][transCol] = g.cornerBL
		default:
			// cur > nxt == y value stepped UP. Bottom of the connector
			// sits at `cur`, top at `nxt`.
			grid[cur][transCol] = g.cornerBR
			for r := nxt + 1; r < cur; r++ {
				grid[r][transCol] = g.vert
			}
			grid[nxt][transCol] = g.cornerTL
		}
	}

	out := make([]string, H)
	for r := 0; r < H; r++ {
		row := ""
		for c := 0; c < width; c++ {
			row += grid[r][c]
		}
		out[r] = row
	}
	return out
}

// validateStepSeries checks `[{x,y}]` shape with sparkline's looser
// rules — negative y and duplicate x are both fine, only shape and
// emptiness matter. Shares CodeBarShape / CodeBarEmpty with the bar
// family so users see the same diagnostic class with the `step:`
// prefix.
func validateStepSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ step: " + reason, true
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
