package pylon

import (
	"strings"
	"testing"
)

// TestRenderText_AtRef asserts that `[ @data | text ]` emits the
// JSON-stringified series as the box body. Go's encoding/json
// matches JS's JSON.stringify for the {x,y} shape (keys sorted
// alphabetically, integer floats without a decimal).
func TestRenderText_AtRef(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n---\n[- @data | text -]"
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌─────────────────────────────────────┐",
		`│   [{"x":1,"y":10},{"x":2,"y":20}]   │`,
		"└─────────────────────────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("text @ref mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderText_RawText asserts that `| text` over raw text is a
// no-op — the body renders exactly as it would without the pipe.
func TestRenderText_RawText(t *testing.T) {
	withPipe := RenderASCII(Parse("[ hello | text ]"))
	withoutPipe := RenderASCII(Parse("[ hello ]"))
	if withPipe != withoutPipe {
		t.Errorf("| text over raw should be a no-op\n--- with pipe ---\n%s\n--- without ---\n%s",
			withPipe, withoutPipe)
	}
}

// TestRenderHBar_Basic locks the SPEC §hbar example byte-for-byte.
// Budget 10, three entries, block glyph U+2588.
func TestRenderHBar_Basic(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n  - x: 3\n    y: 15\n---\n[ @data | bar ]"
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌───────────────────────────┐",
		"│   1 │ █████      (10) │   │",
		"│   2 │ ██████████ (20) │   │",
		"│   3 │ ████████   (15) │   │",
		"└───────────────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("hbar SPEC example mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderHBar_AllZero asserts the all-zero degenerate case
// renders zero bar cells without crashing. maxY=0 must not divide
// by zero; the renderer falls back to cells=0 per series entry.
func TestRenderHBar_AllZero(t *testing.T) {
	src := "---\ndata:\n  - x: a\n    y: 0\n  - x: b\n    y: 0\n---\n[ @data | bar ]"
	got := RenderASCII(Parse(src))
	if strings.Contains(got, barGlyph) {
		t.Errorf("all-zero hbar should render no block glyphs; got:\n%s", got)
	}
	// Body must still carry labels and (0) values.
	if !strings.Contains(got, "a") || !strings.Contains(got, "b") || !strings.Contains(got, "(0)") {
		t.Errorf("all-zero hbar missing labels or values:\n%s", got)
	}
}

// TestRenderHBar_RoundingBoundary pins math.Round half-away-from-
// zero for non-negative fractions (which match JS's Math.round half-
// up). Series chosen so cells = round(y/maxY * 10) hits the .5
// boundary. maxY=10, y=5 → 5.0 (whole), y=1 → 1.0. Use y=3 against
// maxY=4 → 3/4*10 = 7.5, rounds to 8.
func TestRenderHBar_RoundingBoundary(t *testing.T) {
	src := "---\ndata:\n  - x: small\n    y: 3\n  - x: big\n    y: 4\n---\n[ @data | hbar ]"
	got := RenderASCII(Parse(src))
	// big = 10 cells (full budget); small = round(3/4*10) = round(7.5) = 8.
	// The two rows should carry 10 and 8 blocks respectively.
	lines := strings.Split(got, "\n")
	smallRow, bigRow := "", ""
	for _, l := range lines {
		if strings.Contains(l, "small") {
			smallRow = l
		}
		if strings.Contains(l, "big") {
			bigRow = l
		}
	}
	if smallRow == "" || bigRow == "" {
		t.Fatalf("missing expected rows in:\n%s", got)
	}
	if strings.Count(smallRow, barGlyph) != 8 {
		t.Errorf("small row should have 8 block glyphs (round(3/4*10)=8); got %d in %q",
			strings.Count(smallRow, barGlyph), smallRow)
	}
	if strings.Count(bigRow, barGlyph) != 10 {
		t.Errorf("big row should have 10 block glyphs; got %d in %q",
			strings.Count(bigRow, barGlyph), bigRow)
	}
}

// TestRenderVBar_Basic locks the SPEC §vbar example byte-for-byte.
// Budget 10 rows tall, three entries, column width 4 each (driven
// by the "(NN)" value label width).
func TestRenderVBar_Basic(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n  - x: 3\n    y: 15\n---\n[ @data | vbar ]"
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌──────────────────┐",
		"│        █         │",
		"│        █         │",
		"│        █   █     │",
		"│        █   █     │",
		"│        █   █     │",
		"│    █   █   █     │",
		"│    █   █   █     │",
		"│    █   █   █     │",
		"│    █   █   █     │",
		"│    █   █   █     │",
		"│    1   2   3     │",
		"│   (10)(20)(15)   │",
		"└──────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("vbar SPEC example mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderVBar_AllZero asserts the all-zero case renders empty
// bar rows + footers without a crash. Footers must still show
// labels and (0) values centered per column.
func TestRenderVBar_AllZero(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: 0\n  - x: 2\n    y: 0\n---\n[ @data | vbar ]"
	got := RenderASCII(Parse(src))
	if strings.Contains(got, barGlyph) {
		t.Errorf("all-zero vbar should render no block glyphs:\n%s", got)
	}
	if !strings.Contains(got, "(0)") {
		t.Errorf("all-zero vbar missing (0) value label:\n%s", got)
	}
}

// TestBarAlias_ErrorPreservesName asserts the `bar` alias threads
// the user-typed name into the inline error. This is the critical
// contract from U2's rendererInlineError helper — the renderer
// path uses the same helper so drift is impossible.
func TestBarAlias_ErrorPreservesName(t *testing.T) {
	// Empty series via manual injection (YAML subset can't express
	// an empty list).
	ast := Parse("[ @s | bar ]")
	ast.Meta.Data = map[string]interface{}{
		"s": []map[string]interface{}{},
	}
	got := RenderASCII(ast)
	if !strings.Contains(got, "⚠ bar: empty series") {
		t.Errorf("expected '⚠ bar: empty series' in output; got:\n%s", got)
	}
	if strings.Contains(got, "hbar:") {
		t.Errorf("bar alias should not surface 'hbar:' in error text; got:\n%s", got)
	}
}

// TestRenderChart_UnknownRenderer asserts that an unknown renderer
// name produces the inline warning from rendererInlineError.
func TestRenderChart_UnknownRenderer(t *testing.T) {
	got := RenderASCII(Parse("[ foo | weird ]"))
	if !strings.Contains(got, "⚠ unknown renderer: weird") {
		t.Errorf("expected unknown-renderer warning; got:\n%s", got)
	}
}

// TestRenderProgress_Scalar locks the scalar form byte-for-byte.
// Budget 20, y=75 → round(75/100*20)=15 filled cells; pct column
// right-padded to "75%" inside the 4-cell "%3d%%" field.
func TestRenderProgress_Scalar(t *testing.T) {
	got := RenderASCII(Parse("[ 75 | progress ]"))
	want := strings.Join([]string{
		"┌───────────────────────────────┐",
		"│   ███████████████░░░░░  75%   │",
		"└───────────────────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("progress scalar mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderProgress_Clamp asserts silent clamp: y=150 renders as
// 100% (20 filled cells) without a diagnostic.
func TestRenderProgress_Clamp(t *testing.T) {
	got := RenderASCII(Parse("[ 150 | progress ]"))
	if !strings.Contains(got, "████████████████████ 100%") {
		t.Errorf("progress clamp should produce 20 filled + '100%%'; got:\n%s", got)
	}
	if strings.Contains(got, "⚠") {
		t.Errorf("progress out-of-range should not warn inline; got:\n%s", got)
	}
}

// TestRenderProgress_ASCIITheme asserts the ASCII theme swaps the
// filled/empty glyphs to `#` / `.` and preserves the 20-cell budget.
func TestRenderProgress_ASCIITheme(t *testing.T) {
	src := "---\ntheme: ascii\n---\n[ 50 | progress ]"
	got := RenderASCII(Parse(src))
	if !strings.Contains(got, "##########..........  50%") {
		t.Errorf("ASCII theme progress should use # / .; got:\n%s", got)
	}
}

// TestRenderProgress_Series locks the series form: right-padded
// label column, one row per entry, same "{bar} {pct}" tail.
func TestRenderProgress_Series(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  work:",
		"    - x: build",
		"      y: 100",
		"    - x: deploy",
		"      y: 10",
		"---",
		"[ @work | progress ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	if !strings.Contains(got, " build ████████████████████ 100%") {
		t.Errorf("progress series missing 100%% row:\n%s", got)
	}
	if !strings.Contains(got, "deploy ██░░░░░░░░░░░░░░░░░░  10%") {
		t.Errorf("progress series missing 10%% row:\n%s", got)
	}
}

// TestRenderProgress_NotNumber asserts the scalar path emits the
// CodeProgressNotNumber inline warning for non-numeric bodies.
func TestRenderProgress_NotNumber(t *testing.T) {
	got := RenderASCII(Parse("[ hello | progress ]"))
	if !strings.Contains(got, "⚠ progress: expected number") {
		t.Errorf("expected progress-not-number warning; got:\n%s", got)
	}
}

// TestRenderSparkline_Basic locks the eight-level default ramp
// byte-for-byte. y = 0..7 mapped across the full ramp puts one
// glyph per level in monotone order.
func TestRenderSparkline_Basic(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  s:",
		"    - x: 1",
		"      y: 0",
		"    - x: 2",
		"      y: 1",
		"    - x: 3",
		"      y: 2",
		"    - x: 4",
		"      y: 3",
		"    - x: 5",
		"      y: 4",
		"    - x: 6",
		"      y: 5",
		"    - x: 7",
		"      y: 6",
		"    - x: 8",
		"      y: 7",
		"---",
		"[ @s | sparkline ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌──────────────┐",
		"│   ▁▂▃▄▅▆▇█   │",
		"└──────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("sparkline mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderSparkline_Flat asserts a constant series collapses to
// the lowest glyph for every cell (exercises the min == max branch).
func TestRenderSparkline_Flat(t *testing.T) {
	src := "---\ndata:\n  s:\n    - x: 1\n      y: 5\n    - x: 2\n      y: 5\n    - x: 3\n      y: 5\n---\n[ @s | sparkline ]"
	got := RenderASCII(Parse(src))
	if !strings.Contains(got, "▁▁▁") {
		t.Errorf("flat sparkline should collapse to lowest glyph; got:\n%s", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("flat sparkline should NOT paint any full blocks; got:\n%s", got)
	}
}

// TestRenderSparkline_NegativeTolerated asserts sparkline (unlike the
// bar family) accepts negative y and renders without a ⚠ warning.
func TestRenderSparkline_NegativeTolerated(t *testing.T) {
	src := "---\ndata:\n  s:\n    - x: 1\n      y: -5\n    - x: 2\n      y: 0\n    - x: 3\n      y: 5\n---\n[ @s | sparkline ]"
	got := RenderASCII(Parse(src))
	if strings.Contains(got, "⚠") {
		t.Errorf("sparkline should NOT warn on negative y; got:\n%s", got)
	}
	// min = -5, max = 5, span = 10. y=-5 → idx 0 (▁); y=0 → idx 4 (▅);
	// y=5 → idx 7 (█).
	if !strings.Contains(got, "▁▅█") {
		t.Errorf("expected ▁▅█ glyph sequence; got:\n%s", got)
	}
}

// TestRenderSparkline_DuplicateXTolerated asserts sparkline does NOT
// emit CodeBarDuplicateX — only the y-sequence matters, so repeated
// x values are fine.
func TestRenderSparkline_DuplicateXTolerated(t *testing.T) {
	src := "---\ndata:\n  s:\n    - x: a\n      y: 1\n    - x: a\n      y: 2\n---\n[ @s | sparkline ]"
	got := RenderASCII(Parse(src))
	if strings.Contains(got, "duplicate") {
		t.Errorf("sparkline should NOT warn on duplicate x; got:\n%s", got)
	}
}

// TestRenderCandlestick_Basic locks a labelled, mixed bull/bear/doji
// series byte-for-byte. Values chosen so Wed = bull (c > o), Tue =
// bear (c < o), Thu = doji (c == o). Column width fits the widest
// 3-char label; candle glyphs stick to the leftmost cell.
func TestRenderCandlestick_Basic(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  q:",
		"    - x: Mon",
		"      o: 10",
		"      h: 14",
		"      l: 8",
		"      c: 12",
		"    - x: Tue",
		"      o: 12",
		"      h: 13",
		"      l: 9",
		"      c: 10",
		"    - x: Wed",
		"      o: 10",
		"      h: 16",
		"      l: 10",
		"      c: 15",
		"    - x: Thu",
		"      o: 15",
		"      h: 15",
		"      l: 11",
		"      c: 11",
		"    - x: Fri",
		"      o: 11",
		"      h: 13",
		"      l: 11",
		"      c: 13",
		"---",
		"[ @q | candlestick ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌─────────────────────┐",
		"│         │           │",
		"│         ▒  █        │",
		"│   │     ▒  █        │",
		"│   ▒  █  ▒  █  ▒     │",
		"│   ▒  █  ▒  █  ▒     │",
		"│   ▒  █  ▒           │",
		"│   │  │              │",
		"│   │                 │",
		"│   MonTueWedThuFri   │",
		"└─────────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("candlestick basic mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderCandlestick_Compact asserts the compact layout: every x
// is the empty string, column width collapses to 1, and the footer
// row is suppressed. Same body-height contract as the labelled form.
func TestRenderCandlestick_Compact(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  q:",
		`    - x: ""`,
		"      o: 10",
		"      h: 14",
		"      l: 8",
		"      c: 12",
		`    - x: ""`,
		"      o: 12",
		"      h: 13",
		"      l: 9",
		"      c: 10",
		`    - x: ""`,
		"      o: 10",
		"      h: 16",
		"      l: 10",
		"      c: 15",
		"---",
		"[ @q | candlestick ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	lines := strings.Split(got, "\n")
	// 8 body rows + 2 border rows = 10; no footer row.
	if len(lines) != 10 {
		t.Errorf("compact should have 10 rows (8 body + 2 borders), got %d:\n%s",
			len(lines), got)
	}
	if strings.Contains(got, "x") || strings.Contains(got, "Mon") {
		t.Errorf("compact should not carry any label text; got:\n%s", got)
	}
}

// TestRenderCandlestick_Flat exercises the gMin == gMax branch: every
// candle collapses to a single doji glyph on the bottom body row, the
// remaining seven rows are blank.
func TestRenderCandlestick_Flat(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  q:",
		"    - x: A",
		"      o: 5",
		"      h: 5",
		"      l: 5",
		"      c: 5",
		"    - x: B",
		"      o: 5",
		"      h: 5",
		"      l: 5",
		"      c: 5",
		"    - x: C",
		"      o: 5",
		"      h: 5",
		"      l: 5",
		"      c: 5",
		"---",
		"[ @q | candlestick ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	if !strings.Contains(got, "───") {
		t.Errorf("flat series should render three doji glyphs on one row; got:\n%s", got)
	}
	// None of the other candlestick glyphs should appear.
	if strings.Contains(got, "│") && !strings.Contains(got, "│   ───") {
		// `│` appearing outside the border (i.e., inside the grid) is a bug.
		for _, line := range strings.Split(got, "\n") {
			trimmed := strings.TrimPrefix(strings.TrimSuffix(line, "│"), "│")
			if strings.Contains(trimmed, "│") {
				t.Errorf("flat series should not paint any wicks; got:\n%s", got)
				break
			}
		}
	}
	if strings.ContainsAny(got, "▒█") {
		t.Errorf("flat series should not paint any bodies (all dojis); got:\n%s", got)
	}
}

// TestRenderCandlestick_ASCII asserts the ASCII theme swaps in the
// printable-only ramp (|, +, #, -) and keeps every other layout
// decision identical to the default theme.
func TestRenderCandlestick_ASCII(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"theme: ascii",
		"data:",
		"  q:",
		"    - x: Mon",
		"      o: 10",
		"      h: 14",
		"      l: 8",
		"      c: 12",
		"    - x: Tue",
		"      o: 12",
		"      h: 13",
		"      l: 9",
		"      c: 10",
		"    - x: Wed",
		"      o: 10",
		"      h: 16",
		"      l: 10",
		"      c: 15",
		"    - x: Thu",
		"      o: 15",
		"      h: 15",
		"      l: 11",
		"      c: 11",
		"    - x: Fri",
		"      o: 11",
		"      h: 13",
		"      l: 11",
		"      c: 13",
		"---",
		"[ @q | candlestick ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"+---------------------+",
		"|         |           |",
		"|         +  #        |",
		"|   |     +  #        |",
		"|   +  #  +  #  +     |",
		"|   +  #  +  #  +     |",
		"|   +  #  +           |",
		"|   |  |              |",
		"|   |                 |",
		"|   MonTueWedThuFri   |",
		"+---------------------+",
	}, "\n")
	if got != want {
		t.Errorf("candlestick ASCII theme mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderHist_Basic locks a small histogram fixture. Column width
// is driven by the footer (single-char labels → colW=1) and the
// block glyph fills each column from the bottom up.
func TestRenderHist_Basic(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  h:",
		"    - x: a",
		"      y: 1",
		"    - x: b",
		"      y: 3",
		"    - x: c",
		"      y: 5",
		"---",
		"[ @h | hist ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌─────────┐",
		"│     █   │",
		"│     █   │",
		"│     █   │",
		"│    ██   │",
		"│    ██   │",
		"│    ██   │",
		"│   ███   │",
		"│   ███   │",
		"│   abc   │",
		"└─────────┘",
	}, "\n")
	if got != want {
		t.Errorf("hist basic mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderStep_WorkedExample locks the exact 5x5 grid pinned in
// the contract: series y=[3,5,2] at H=5 normalises to rows [3,0,4],
// which draws the up-then-down staircase byte-for-byte below.
func TestRenderStep_WorkedExample(t *testing.T) {
	got := renderStep([]map[string]interface{}{
		{"x": 1.0, "y": 3.0},
		{"x": 2.0, "y": 5.0},
		{"x": 3.0, "y": 2.0},
	}, unicodeBox)
	want := []string{
		" ┌─┐ ",
		" │ │ ",
		" │ │ ",
		"─┘ │ ",
		"   └─",
	}
	if len(got) != len(want) {
		t.Fatalf("row count = %d, want %d; rows = %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRenderStep_Flat asserts all-equal y collapses every entry to
// row H-1 (bottom), rendering as a single full-width horizontal line.
func TestRenderStep_Flat(t *testing.T) {
	got := renderStep([]map[string]interface{}{
		{"x": 1.0, "y": 5.0},
		{"x": 2.0, "y": 5.0},
		{"x": 3.0, "y": 5.0},
	}, unicodeBox)
	// Width = 2n-1 = 5 cells; only bottom row non-blank.
	want := []string{
		"     ",
		"     ",
		"     ",
		"     ",
		"─────",
	}
	if len(got) != len(want) {
		t.Fatalf("row count = %d, want %d; rows = %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRenderGantt_Basic locks a small task series byte-for-byte.
// Budget 20, three tasks on maxEnd=10: the bar spans are 0-7, 4-15,
// 12-19 per the round(start/maxEnd*budget) / round(end/maxEnd*budget)
// -1 rule.
func TestRenderGantt_Basic(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"data:",
		"  tasks:",
		"    - x: spec",
		"      start: 0",
		"      end: 4",
		"    - x: build",
		"      start: 2",
		"      end: 8",
		"    - x: test",
		"      start: 6",
		"      end: 10",
		"---",
		"[ @tasks | gantt ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	want := strings.Join([]string{
		"┌────────────────────────────────┐",
		"│    spec ████████               │",
		"│   build     ████████████       │",
		"│    test             ████████   │",
		"└────────────────────────────────┘",
	}, "\n")
	if got != want {
		t.Errorf("gantt basic mismatch\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

// TestRenderGantt_Size exercises the size-aware budget: with
// `size: 40x6` and a 5-char widest label, budget = 40-5-4 = 31 (well
// above the 5-cell floor), so bars stretch further than the default
// 20-cell path.
func TestRenderGantt_Size(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"size: 40x6",
		"data:",
		"  tasks:",
		"    - x: spec",
		"      start: 0",
		"      end: 4",
		"    - x: build",
		"      start: 2",
		"      end: 8",
		"    - x: test",
		"      start: 6",
		"      end: 10",
		"---",
		"[ @tasks | gantt ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	// spec = round(4/10*31) = 12 cells; build = 24-6 = 19 cells;
	// test = 31-19 = 12 cells.
	if !strings.Contains(got, strings.Repeat(barGlyph, 19)) {
		t.Errorf("expected a 19-cell run (build task) under size:40x6; got:\n%s", got)
	}
	if strings.Contains(got, strings.Repeat(barGlyph, 20)) {
		t.Errorf("didn't expect a 20-cell run under size:40x6 (would mean default budget still applied):\n%s", got)
	}
}

// TestRenderGantt_SizeFloor pins the 5-cell minimum budget.
// With labels up to 6 chars ("deploy") and `size: 15x6`, the math
// gives max(5, 15 - 6 - 4) = 5 — the floor binds exactly while the
// outer box stays wide enough to hold 3 left-pad + 6 label + 1 gap
// + 5 bar cells. A narrower size would let the outer box clip the
// bar area regardless of what gantt computes; the floor is a
// gantt-side guarantee, not a layout guarantee.
func TestRenderGantt_SizeFloor(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"size: 15x6",
		"data:",
		"  tasks:",
		"    - x: design",
		"      start: 0",
		"      end: 10",
		"    - x: deploy",
		"      start: 3",
		"      end: 10",
		"---",
		"[ @tasks | gantt ]",
	}, "\n")
	got := RenderASCII(Parse(src))
	// Floor at 5 cells; design's 0..10 spans the entire maxEnd so it
	// occupies all 5 cells — a run of 5 block glyphs must appear.
	if !strings.Contains(got, strings.Repeat(barGlyph, 5)) {
		t.Errorf("expected at least one 5-cell run under size:15x6 floor; got:\n%s", got)
	}
	// Reject the default 20-cell budget slipping through.
	if strings.Contains(got, strings.Repeat(barGlyph, 20)) {
		t.Errorf("didn't expect a 20-cell run (would mean default budget still applied):\n%s", got)
	}
}

// TestRenderChart_DoesNotDisturbNormalBoxes is a regression guard:
// normal (non-chart) boxes must render exactly as they did before
// the chart dispatch was added.
func TestRenderChart_DoesNotDisturbNormalBoxes(t *testing.T) {
	cases := []string{
		"[ hello ]",
		"[ a ] -> [ b ]",
		"[ a :: x ] -> &x",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			got := RenderASCII(Parse(src))
			if strings.Contains(got, barGlyph) {
				t.Errorf("non-chart source produced block glyph (%q):\n%s", src, got)
			}
			if strings.Contains(got, "⚠") {
				t.Errorf("non-chart source produced warning glyph (%q):\n%s", src, got)
			}
		})
	}
}
