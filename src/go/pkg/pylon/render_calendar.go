package pylon

import (
	"math"
	"strings"
	"time"
)

// dowLabels are the row labels in the calendar grid. Each entry
// renders one calendar day-of-week, Sunday-first to match JS
// Date.getDay() and most western calendars.
var (
	dowLabelsDefault = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	dowLabelsASCII   = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

// renderCalendar lays out a GitHub-style year-of-days heatmap. Each
// input entry carries a `date` string in YYYY-MM-DD form and a
// numeric y. Output: 7 rows (Sun..Sat) × N cols where N is the
// number of weeks spanned by the input dates. Each cell shades by
// the heatmap ramp.
//
// Dates outside the input range render as blank (heatmap glyph 0).
// Multiple entries on the same date sum (additive).
func renderCalendar(series []map[string]interface{}, bc boxChars) []string {
	glyphs := heatmapGlyphs(bc)
	rampLevels := len(glyphs) - 1
	dows := dowLabelsDefault
	if bc == asciiBox {
		dows = dowLabelsASCII
	}

	type cell struct {
		date time.Time
		y    float64
	}
	cells := make([]cell, 0, len(series))
	for _, e := range series {
		ds, _ := e["date"].(string)
		y, _ := e["y"].(float64)
		t, err := time.Parse("2006-01-02", ds)
		if err != nil {
			continue
		}
		cells = append(cells, cell{date: t, y: y})
	}
	if len(cells) == 0 {
		return []string{""}
	}

	// Find min/max date to bound the grid. Anchor the first column
	// to the start of the week containing the earliest date so each
	// column maps to a clean Sun..Sat week.
	minDate := cells[0].date
	maxDate := cells[0].date
	maxY := cells[0].y
	for _, c := range cells {
		if c.date.Before(minDate) {
			minDate = c.date
		}
		if c.date.After(maxDate) {
			maxDate = c.date
		}
		if c.y > maxY {
			maxY = c.y
		}
	}
	// Anchor to the Sunday on/before minDate.
	anchor := minDate.AddDate(0, 0, -int(minDate.Weekday()))
	totalDays := int(maxDate.Sub(anchor).Hours()/24) + 1
	weeks := (totalDays + 6) / 7
	if weeks < 1 {
		weeks = 1
	}

	grid := make([][]string, 7)
	for r := 0; r < 7; r++ {
		grid[r] = make([]string, weeks)
		for c := 0; c < weeks; c++ {
			grid[r][c] = glyphs[0]
		}
	}
	for _, c := range cells {
		days := int(c.date.Sub(anchor).Hours() / 24)
		col := days / 7
		row := days % 7
		if col < 0 || col >= weeks || row < 0 || row >= 7 {
			continue
		}
		idx := 0
		if maxY > 0 {
			idx = int(math.Round(c.y / maxY * float64(rampLevels)))
		}
		if idx < 1 {
			idx = 1
		}
		if idx > rampLevels {
			idx = rampLevels
		}
		grid[row][col] = glyphs[idx]
	}

	labelW := 3 // "Sun", "Mon", ... all 3 cells wide
	out := make([]string, 7)
	for r := 0; r < 7; r++ {
		out[r] = padRow(dows[r], labelW, AlignLeft, 0) + " " + strings.Join(grid[r], "")
	}
	return out
}

// validateCalendarSeries enforces `[{date, y}]` with `date` a
// YYYY-MM-DD literal string and `y` a non-negative number. Wire
// prefix is "calendar:".
func validateCalendarSeries(series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ calendar: " + reason, true
	}
	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{date, y}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{date, y}]")
		}
		dRaw, hasDate := entry["date"]
		yRaw, hasY := entry["y"]
		if !hasDate || !hasY {
			return emit(CodeBarShape, "expected [{date, y}]")
		}
		dStr, ok := dRaw.(string)
		if !ok {
			return emit(CodeBarShape, "expected [{date, y}]")
		}
		if _, err := time.Parse("2006-01-02", dStr); err != nil {
			return emit(CodeBarShape, "invalid date: "+dStr)
		}
		yNum, isNum := yRaw.(float64)
		if !isNum || math.IsNaN(yNum) {
			return emit(CodeBarShape, "expected [{date, y}]")
		}
		if yNum < 0 {
			return emit(CodeBarNegativeY, "negative y")
		}
	}
	return "", "", false
}
