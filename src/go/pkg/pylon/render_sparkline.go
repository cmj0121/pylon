package pylon

import (
	"math"
	"strings"
)

// Sparkline glyph ramps. Eight levels for the default theme
// (U+2581..U+2588) give a smooth monotone shape; ASCII falls back to
// four levels since plain ASCII can't express a clean 8-step ramp.
// Constants mirror src/js/pylon.js so output stays byte-identical
// across the two implementations.
var (
	sparklineGlyphsDefault = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	sparklineGlyphsASCII   = []string{"_", "-", "*", "#"}
)

// sparklineGlyphs picks the ramp for bc's theme. Same dispatch shape
// as heatmapGlyphs / progressGlyphs.
func sparklineGlyphs(bc boxChars) []string {
	if bc == asciiBox {
		return sparklineGlyphsASCII
	}
	return sparklineGlyphsDefault
}

// renderSparkline lays out one glyph per entry on a single row.
// Normalisation is relative -- (y-min)/(max-min) -- so a series of
// large positive values still shows trend shape, and signed values
// work naturally (this is the only chart primitive that tolerates
// negative y). A flat series (min == max) collapses to the lowest
// glyph for every cell. The validator has ruled out bad shape / NaN
// by this point, so type assertions are infallible.
func renderSparkline(series []map[string]interface{}, bc boxChars) []string {
	glyphs := sparklineGlyphs(bc)
	n := len(glyphs)
	ys := make([]float64, len(series))
	for i, e := range series {
		y, _ := e["y"].(float64)
		ys[i] = y
	}

	minV, maxV := ys[0], ys[0]
	for _, v := range ys[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	span := maxV - minV
	var row strings.Builder
	for _, v := range ys {
		idx := 0
		if span > 0 {
			idx = int(math.Round((v - minV) / span * float64(n-1)))
		}
		row.WriteString(glyphs[idx])
	}
	return []string{row.String()}
}

// validateSparklineSeries checks the resolved series matches
// `[{x,y}]`. Unlike the bar family, sparkline tolerates negative y
// (trend viz -- signed deltas are normal) and duplicate x (it only
// uses the y sequence in order), so the walk does only shape + empty
// checks. Shares CodeBarShape / CodeBarEmpty with the bar family so
// users see the same diagnostic classes with the `sparkline:` prefix.
func validateSparklineSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ sparkline: " + reason, true
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
