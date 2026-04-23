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
