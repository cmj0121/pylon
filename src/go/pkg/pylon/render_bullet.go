package pylon

import (
	"fmt"
	"math"
	"strings"
)

// bulletBudget is the cell budget for the bullet bar's horizontal
// extent. Same value as the standard bar budget so bullet rows align
// with bar rows when stacked.
const bulletBudget = 20

// bulletGlyphs collects the four-layer glyphs the bullet renderer
// uses: three band shades (poor / ok / good) + the actual-value
// foreground + the target marker. Theme: ascii folds the ramp shades
// into printable equivalents.
type bulletGlyphs struct {
	bandLo, bandMid, bandHi string
	bar, target             string
}

var (
	bulletGlyphsDefault = bulletGlyphs{
		bandLo: "░", bandMid: "▒", bandHi: "▓",
		bar: "█", target: "◆",
	}
	bulletGlyphsASCII = bulletGlyphs{
		bandLo: ".", bandMid: "+", bandHi: "*",
		bar: "#", target: "|",
	}
)

func bulletGlyphRamp(bc boxChars) bulletGlyphs {
	if bc == asciiBox {
		return bulletGlyphsASCII
	}
	return bulletGlyphsDefault
}

// renderBullet lays out a layered bar: shaded qualitative bands as
// the background, the actual value as a foreground bar, and (when
// present) a target marker. Entry shape: `{x, y[, target]}`. The
// implicit band thresholds are 33% / 66% / 100% of the row's
// reference scale (max of y and target across all rows).
//
// Each row: "{label} {│} {bands+bar}{target marker} {value}".
func renderBullet(series []map[string]interface{}, bc boxChars) []string {
	g := bulletGlyphRamp(bc)
	n := len(series)
	labels := make([]string, n)
	values := make([]string, n)
	ys := make([]float64, n)
	targets := make([]float64, n)
	hasTarget := make([]bool, n)
	maxRef := 0.0
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		y, _ := e["y"].(float64)
		ys[i] = y
		values[i] = "(" + formatY(y) + ")"
		if t, ok := e["target"].(float64); ok {
			targets[i] = t
			hasTarget[i] = true
			if t > maxRef {
				maxRef = t
			}
		}
		if y > maxRef {
			maxRef = y
		}
	}

	labelW := 0
	valueW := 0
	for i := range series {
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
		if w := displayWidth(values[i]); w > valueW {
			valueW = w
		}
	}

	budget := bulletBudget
	loEnd := budget / 3
	midEnd := 2 * budget / 3

	out := make([]string, n)
	for i := range series {
		bar := make([]string, budget)
		for c := 0; c < budget; c++ {
			switch {
			case c < loEnd:
				bar[c] = g.bandLo
			case c < midEnd:
				bar[c] = g.bandMid
			default:
				bar[c] = g.bandHi
			}
		}
		// Overlay the actual-value bar from col 0 up to the value cell.
		valCells := 0
		if maxRef > 0 {
			valCells = int(math.Round(ys[i] / maxRef * float64(budget)))
		}
		if valCells > budget {
			valCells = budget
		}
		for c := 0; c < valCells; c++ {
			bar[c] = g.bar
		}
		// Overlay the target marker, if any.
		if hasTarget[i] {
			tCell := 0
			if maxRef > 0 {
				tCell = int(math.Round(targets[i] / maxRef * float64(budget)))
			}
			if tCell >= budget {
				tCell = budget - 1
			}
			if tCell < 0 {
				tCell = 0
			}
			bar[tCell] = g.target
		}
		body := strings.Join(bar, "") + " "
		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + bc.v + " " +
			body + padRow(values[i], valueW, AlignRight, 0)
	}
	return out
}

// validateBulletSeries accepts `[{x, y[, target]}]`. y must be a
// non-negative number; target (when present) must be a non-negative
// number. Negative or NaN values fire as shape errors. Wire prefix
// is "bullet:".
func validateBulletSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ bullet: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x, y, target?}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x, y, target?}]")
		}
		_, hasX := entry["x"]
		yRaw, hasY := entry["y"]
		if !hasX || !hasY {
			return emit(CodeBarShape, "expected [{x, y, target?}]")
		}
		yNum, isNum := yRaw.(float64)
		if !isNum || math.IsNaN(yNum) {
			return emit(CodeBarShape, "expected [{x, y, target?}]")
		}
		if yNum < 0 {
			return emit(CodeBarNegativeY, "negative y")
		}
		if tRaw, has := entry["target"]; has {
			tNum, isNum := tRaw.(float64)
			if !isNum || math.IsNaN(tNum) {
				return emit(CodeBarShape, "expected [{x, y, target?}]")
			}
			if tNum < 0 {
				return emit(CodeBarNegativeY, "negative target")
			}
		}
	}
	return "", "", false
}
