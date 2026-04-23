package pylon

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Chart layout constants. Values mirror src/js/pylon.js so ASCII
// output stays byte-identical across the two implementations:
//   - barGlyph: U+2588 FULL BLOCK ("█"). One display cell.
//   - barWidthDefault: horizontal bar budget when the outer box has
//     no size constraint.
//   - barHeightDefault: vertical bar budget (rows). Not tightened by
//     outer size today — matches JS.
//   - vbarColumnMinWidth: floor for per-column width; ensures a lone
//     `█` has breathing room even when labels and values are short.
const (
	barGlyph           = "█"
	barWidthDefault    = 10
	barHeightDefault   = 10
	vbarColumnMinWidth = 3
)

// applyChartRenderer replaces a chart box's rendered itemRows when
// the box declares a renderer or carries a bare @ref. Returns the
// rows that substitute for what renderBoxRows would otherwise
// produce by walking b.Items.
//
// Flow:
//  1. Ask rendererInlineError for the single error that applies to
//     this box (covers bare @ref, unknown renderer, bad series, ...).
//     If ok=true, the box body is the warning string; the caller
//     wraps it in the normal border / pad machinery.
//  2. Otherwise dispatch on b.Renderer. `text` falls back to nil
//     when there's no @ref child — "| text" over raw text is a
//     no-op and the normal Items path should run.
//
// Bar width is fixed at barWidthDefault; size: constraints at the
// outer box shrink the clipRow budget rather than the chart's own
// layout, matching JS behaviour.
func applyChartRenderer(b *Box, data interface{}, bc boxChars) []string {
	// rendererInlineError reads only meta.Data; a minimal Meta is
	// enough to exercise the shared helper without dragging the
	// whole frontmatter struct through the render path.
	if _, msg, ok := rendererInlineError(b, Meta{Data: data}); ok {
		return []string{msg}
	}

	switch b.Renderer {
	case "text":
		return renderTextChart(b, data)
	case "hbar", "bar":
		return renderHBar(chartSeries(b, data), bc, barWidthDefault)
	case "vbar":
		return renderVBar(chartSeries(b, data))
	case "banner":
		return renderBanner(b, bc)
	case "progress":
		if firstDataRef(b) != nil {
			return renderProgressSeries(chartSeries(b, data), bc)
		}
		return renderProgressScalar(b, bc)
	case "heatmap":
		return renderHeatmap(chartSeries(b, data), bc)
	case "sparkline":
		return renderSparkline(chartSeries(b, data), bc)
	}
	return nil
}

// chartSeries pulls the resolved []{x,y} slice for a validated
// chart box. rendererInlineError has already ruled out missing
// refs, wrong shapes, and so on by the time we get here, so the
// type assertions are infallible — any drift would be a validator
// regression and should panic loudly rather than render garbage.
func chartSeries(b *Box, data interface{}) []map[string]interface{} {
	ref := firstDataRef(b)
	if ref == nil {
		return nil
	}
	raw, ok := lookupSeries(data, ref.Name)
	if !ok {
		return nil
	}
	arr, _ := raw.([]map[string]interface{})
	return arr
}

// renderTextChart handles `| text`. Two shapes:
//
//   - `[ @name | text ]` → one row carrying JSON.stringify(series).
//     Go's encoding/json produces the same byte sequence as JS for
//     the `{x, y}` series shape (keys sorted alphabetically, numeric
//     integers render without a decimal).
//   - `[ raw | text ]` → nil, signalling the caller to render
//     b.Items normally. A text renderer over raw text is a no-op.
func renderTextChart(b *Box, data interface{}) []string {
	ref := firstDataRef(b)
	if ref == nil {
		return nil
	}
	series, ok := lookupSeries(data, ref.Name)
	if !ok {
		return nil
	}
	raw, err := json.Marshal(series)
	if err != nil {
		// Shouldn't happen — the series is a plain []map[string]any
		// with scalar values, all of which encoding/json can handle.
		return []string{fmt.Sprintf("%v", series)}
	}
	return []string{string(raw)}
}

// renderHBar lays out a horizontal bar chart. Row shape (SPEC
// §hbar / bar): "{label} {│} {body}{value} {│}" where body is
// (bars + spaces) clipped to budget+1 display cells so the
// trailing space between the bar and the value label stays
// constant when y is small.
func renderHBar(series []map[string]interface{}, bc boxChars, budgetW int) []string {
	budget := budgetW
	if budget < 1 {
		budget = 1
	}

	labels := make([]string, len(series))
	values := make([]string, len(series))
	ys := make([]float64, len(series))
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		y, _ := e["y"].(float64)
		ys[i] = y
		values[i] = "(" + formatY(y) + ")"
	}

	labelW := 0
	valueW := 0
	maxY := 0.0
	for i := range series {
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
		if w := displayWidth(values[i]); w > valueW {
			valueW = w
		}
		if ys[i] > maxY {
			maxY = ys[i]
		}
	}

	out := make([]string, len(series))
	for i := range series {
		cells := 0
		if maxY > 0 {
			cells = int(math.Round(ys[i] / maxY * float64(budget)))
		}
		body := strings.Repeat(barGlyph, cells) + strings.Repeat(" ", budget)
		body = clipRow(body, budget+1)
		out[i] = padRow(labels[i], labelW, AlignRight, 0) + " " + bc.v + " " +
			body + padRow(values[i], valueW, AlignRight, 0) + " " + bc.v
	}
	return out
}

// renderVBar lays out a vertical bar chart. Grid: barHeightDefault
// rows of bars + 2 footer rows (labels, values). Each column is
// max(|label|, |value|, vbarColumnMinWidth) wide, left-biased
// centered. Height is not tightened by outer size today.
//
// Box chars aren't threaded in — vbar's grid is pure block + space
// glyphs, with the outer theme's border painted around it by
// renderBoxRows.
func renderVBar(series []map[string]interface{}) []string {
	n := len(series)
	labels := make([]string, n)
	values := make([]string, n)
	ys := make([]float64, n)
	colW := make([]int, n)
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		y, _ := e["y"].(float64)
		ys[i] = y
		values[i] = "(" + formatY(y) + ")"
		w := displayWidth(labels[i])
		if vw := displayWidth(values[i]); vw > w {
			w = vw
		}
		if w < vbarColumnMinWidth {
			w = vbarColumnMinWidth
		}
		colW[i] = w
	}

	maxY := 0.0
	for _, y := range ys {
		if y > maxY {
			maxY = y
		}
	}
	barH := make([]int, n)
	for i := range series {
		if maxY > 0 {
			barH[i] = int(math.Round(ys[i] / maxY * float64(barHeightDefault)))
		}
	}

	rows := make([]string, 0, barHeightDefault+2)
	for r := 0; r < barHeightDefault; r++ {
		var sb strings.Builder
		for i := range series {
			content := " "
			if barH[i] >= barHeightDefault-r {
				content = barGlyph
			}
			sb.WriteString(padRow(content, colW[i], AlignCenter, 1))
		}
		rows = append(rows, sb.String())
	}
	var labelRow, valueRow strings.Builder
	for i := range series {
		labelRow.WriteString(padRow(labels[i], colW[i], AlignCenter, 1))
		valueRow.WriteString(padRow(values[i], colW[i], AlignCenter, 1))
	}
	rows = append(rows, labelRow.String(), valueRow.String())
	return rows
}

// formatY renders a bar value. Integers come through without a
// decimal (JS does `String(y)` which also drops trailing zeros);
// fractions use %g so "1.5" stays "1.5", not "1.500000". NaN and
// Inf never reach here — rendererInlineError rejects them upstream.
func formatY(y float64) string {
	if y == math.Trunc(y) && !math.IsInf(y, 0) {
		return fmt.Sprintf("%d", int64(y))
	}
	return fmt.Sprintf("%g", y)
}
