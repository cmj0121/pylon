package pylon

import (
	"fmt"
	"math"
	"strings"
)

// candlestickHeightDefault pins the body height in rows. Eight rows
// give enough vertical resolution to separate wick from body in
// typical OHLC series without dominating the surrounding diagram.
const candlestickHeightDefault = 8

// candlestickRamp bundles the five glyphs a candle column can paint.
// Small struct rather than a map keeps the picker allocation-free and
// the per-column dispatch a simple field read.
type candlestickRamp struct {
	blank, wick, bull, bear, doji string
}

// Candlestick glyph ramps. Default theme leans on box-drawing glyphs
// for a clean wick/body split; ASCII falls back to printable-only
// characters so terminals without Unicode still render a legible
// shape. Constants mirror src/js/pylon.js so output stays byte-
// identical across the two implementations.
var (
	candlestickGlyphsDefault = candlestickRamp{
		blank: " ", wick: "│", bull: "▒", bear: "█", doji: "─",
	}
	candlestickGlyphsASCII = candlestickRamp{
		blank: " ", wick: "|", bull: "+", bear: "#", doji: "-",
	}
)

// candlestickGlyphs picks the ramp for bc's theme.
func candlestickGlyphs(bc boxChars) candlestickRamp {
	if bc == asciiBox {
		return candlestickGlyphsASCII
	}
	return candlestickGlyphsDefault
}

// renderCandlestick lays out an OHLC series as a fixed-height grid.
// Two layout modes:
//
//   - Compact: every x is "" → column width 1, no footer row.
//   - Labelled: at least one non-empty x → column width fits the
//     widest label; candle glyphs stick to the left-most cell of each
//     column (cells 1..w-1 are spaces) so the shape stays visible
//     even under wide date labels; footer row carries centred labels.
//
// Normalisation is global across the whole series: gMin/gMax span
// every o/h/l/c. When gMin == gMax (flat series) the grid collapses
// to a single doji row at the bottom — the deterministic branch that
// sidesteps divide-by-zero. The validator has ruled out bad shape /
// non-finite prices by this point, so type assertions are infallible.
//
// colorOn wraps the BODY glyphs (bull, bear, doji) with the semantic
// ANSI palette via wrapANSI. Wick and blank glyphs stay uncoloured so
// the scaffolding of the chart reads the same in plain output. Under
// `theme: ascii` (bc == asciiBox) color is force-suppressed so ASCII
// output is always plain.
func renderCandlestick(series []map[string]interface{}, bc boxChars, colorOn bool) []string {
	if bc == asciiBox {
		colorOn = false
	}
	g := candlestickGlyphs(bc)
	H := candlestickHeightDefault
	n := len(series)

	labels := make([]string, n)
	os := make([]float64, n)
	hs := make([]float64, n)
	ls := make([]float64, n)
	cs := make([]float64, n)
	hasLabel := false
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		if labels[i] != "" {
			hasLabel = true
		}
		os[i], _ = e["o"].(float64)
		hs[i], _ = e["h"].(float64)
		ls[i], _ = e["l"].(float64)
		cs[i], _ = e["c"].(float64)
	}

	gMin, gMax := hs[0], hs[0]
	for i := 0; i < n; i++ {
		for _, v := range [4]float64{os[i], hs[i], ls[i], cs[i]} {
			if v < gMin {
				gMin = v
			}
			if v > gMax {
				gMax = v
			}
		}
	}
	flat := gMin == gMax
	span := gMax - gMin

	// rowOf maps a price to its grid row. Flat series force the
	// bottom row so every candle collapses to a single doji glyph.
	rowOf := func(p float64) int {
		if flat {
			return H - 1
		}
		idx := int(math.Round((p - gMin) / span * float64(H-1)))
		return H - 1 - idx
	}

	colW := make([]int, n)
	for i := range series {
		if hasLabel {
			w := displayWidth(labels[i])
			if w < 1 {
				w = 1
			}
			colW[i] = w
		} else {
			colW[i] = 1
		}
	}

	rows := make([]string, 0, H+1)
	for r := 0; r < H; r++ {
		var sb strings.Builder
		for i := range series {
			rH := rowOf(hs[i])
			rL := rowOf(ls[i])
			rO := rowOf(os[i])
			rC := rowOf(cs[i])
			bodyTop, bodyBot := rO, rC
			if rC < rO {
				bodyTop, bodyBot = rC, rO
			}

			var glyph string
			switch {
			case r < rH, r > rL:
				glyph = g.blank
			case r >= bodyTop && r <= bodyBot:
				// c == o → bodyTop == bodyBot, one doji row.
				switch {
				case cs[i] > os[i]:
					glyph = wrapANSI(g.bull, semBull, colorOn)
				case cs[i] < os[i]:
					glyph = wrapANSI(g.bear, semBear, colorOn)
				default:
					glyph = wrapANSI(g.doji, semDoji, colorOn)
				}
			default:
				glyph = g.wick
			}

			sb.WriteString(glyph)
			if colW[i] > 1 {
				sb.WriteString(strings.Repeat(" ", colW[i]-1))
			}
		}
		rows = append(rows, sb.String())
	}

	if hasLabel {
		var footer strings.Builder
		for i := range series {
			footer.WriteString(padRow(labels[i], colW[i], AlignCenter, 1))
		}
		rows = append(rows, footer.String())
	}
	return rows
}

// validateCandlestickSeries enforces the three candlestick-specific
// checks in order: shape (`[{x, o, h, l, c}]` with finite numbers),
// non-empty, and per-entry OHLC consistency (h >= max(o,c), l <=
// min(o,c)). Negative prices and duplicate x are tolerated — same
// tolerance model as sparkline, since OHLC data routinely carries
// signed values and repeated timestamps are rare but legal.
func validateCandlestickSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ candlestick: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x, o, h, l, c}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x, o, h, l, c}]")
		}
		if _, ok := entry["x"]; !ok {
			return emit(CodeBarShape, "expected [{x, o, h, l, c}]")
		}
		for _, k := range [4]string{"o", "h", "l", "c"} {
			raw, has := entry[k]
			if !has {
				return emit(CodeBarShape, "expected [{x, o, h, l, c}]")
			}
			num, isNum := raw.(float64)
			if !isNum || math.IsNaN(num) || math.IsInf(num, 0) {
				return emit(CodeBarShape, "expected [{x, o, h, l, c}]")
			}
		}
		o := entry["o"].(float64)
		h := entry["h"].(float64)
		l := entry["l"].(float64)
		c := entry["c"].(float64)
		hi := o
		if c > hi {
			hi = c
		}
		lo := o
		if c < lo {
			lo = c
		}
		if h < hi || l > lo {
			return emit(CodeCandlestickOHLC, "invalid ohlc")
		}
	}
	return "", "", false
}
