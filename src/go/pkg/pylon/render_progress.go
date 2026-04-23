package pylon

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Progress layout constants. The budget is fixed at 20 cells (see
// design doc): a single-row bar that mirrors the JS side byte-for-
// byte. progressFilledDefault / progressEmptyDefault pick the
// Unicode glyphs; the ASCII theme swaps to ASCII fallbacks.
//   - progressFilledDefault: U+2588 FULL BLOCK ("█")
//   - progressEmptyDefault:  U+2591 LIGHT SHADE ("░")
//   - progressFilledASCII:   "#"
//   - progressEmptyASCII:    "."
const (
	progressBudget        = 20
	progressFilledDefault = "█"
	progressEmptyDefault  = "░"
	progressFilledASCII   = "#"
	progressEmptyASCII    = "."
)

// renderProgress dispatches a `| progress` box to either the scalar
// path (literal number in Text content) or the series path
// (resolved @ref → []{x,y}). rendererInlineError has already
// verified the relevant shape by the time we reach here.
func renderProgress(b *Box, data interface{}, bc boxChars) []string {
	filled, empty := progressGlyphs(bc)
	if ref := firstDataRef(b); ref != nil {
		series, _ := lookupSeries(data, ref.Name)
		arr, _ := series.([]map[string]interface{})
		return renderProgressSeries(arr, filled, empty)
	}
	// Scalar path: the validator already ensured the concatenated
	// Text parses as a number, so the error return is unreachable —
	// a zero value renders as "0%" rather than panicking.
	y, _ := parseProgressScalar(b)
	return []string{renderProgressRow(y, filled, empty)}
}

// progressGlyphs picks the filled / empty cell glyphs for the active
// theme. Mirrors renderBanner's `bc == asciiBox` check so the two
// implementations stay in lockstep.
func progressGlyphs(bc boxChars) (string, string) {
	if bc == asciiBox {
		return progressFilledASCII, progressEmptyASCII
	}
	return progressFilledDefault, progressEmptyDefault
}

// renderProgressSeries lays out N rows for a `[ @ref | progress ]`
// box. Label column is right-padded to the widest entry; each row
// is `{label} {bar-20} {pct-4}` with single-space separators.
func renderProgressSeries(series []map[string]interface{}, filled, empty string) []string {
	labelW := 0
	labels := make([]string, len(series))
	ys := make([]float64, len(series))
	for i, e := range series {
		labels[i] = fmt.Sprintf("%v", e["x"])
		y, _ := e["y"].(float64)
		ys[i] = y
		if w := displayWidth(labels[i]); w > labelW {
			labelW = w
		}
	}
	out := make([]string, len(series))
	for i := range series {
		label := padRow(labels[i], labelW, AlignRight, 0)
		out[i] = label + " " + renderProgressRow(ys[i], filled, empty)
	}
	return out
}

// renderProgressRow formats the "{bar-20-cells} {pct-4}" tail shared
// by both the scalar and series paths. The percentage column is
// right-aligned within 4 cells (3-digit number + "%") so "100%" sits
// flush and " 75%"/"  0%" lead with spaces.
func renderProgressRow(y float64, filled, empty string) string {
	pct := clampPercent(y)
	cells := int(math.Round(pct / 100 * float64(progressBudget)))
	if cells < 0 {
		cells = 0
	}
	if cells > progressBudget {
		cells = progressBudget
	}
	bar := strings.Repeat(filled, cells) + strings.Repeat(empty, progressBudget-cells)
	// "%3d%%" renders the percent right-aligned in 4 cells: "100%",
	// " 75%", "  0%". The single-space separator plus the padded
	// percent produces the two-space visual breathing room before
	// sub-100 values noted in the design doc.
	return bar + " " + fmt.Sprintf("%3d%%", int(math.Round(pct)))
}

// clampPercent silently clamps y into [0, 100]. NaN collapses to 0
// so a malformed fixture still prints a deterministic row rather
// than "NaN%". The validator is still responsible for surfacing
// malformed scalar input — this is a renderer-level safety net.
func clampPercent(y float64) float64 {
	if math.IsNaN(y) {
		return 0
	}
	if y < 0 {
		return 0
	}
	if y > 100 {
		return 100
	}
	return y
}

// parseProgressScalar reads the concatenated Text content of a box
// as a float. Whitespace is stripped. Returns (value, true) when the
// body parses cleanly; (0, false) on NaN / empty / unparseable. The
// validator calls this to decide between a clean render and the
// CodeProgressNotNumber diagnostic.
func parseProgressScalar(b *Box) (float64, bool) {
	var sb strings.Builder
	collectBannerText(b, &sb)
	raw := strings.TrimSpace(sb.String())
	if raw == "" {
		return 0, false
	}
	y, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(y) || math.IsInf(y, 0) {
		return 0, false
	}
	return y, true
}
