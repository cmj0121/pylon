package pylon

import "testing"

// findDiag returns the first diagnostic with the given Code from a
// slice, or nil if none. Test helpers use this to pin expectations
// without asserting on incidental order-of-emission.
func findDiag(diags []Diagnostic, code Code) *Diagnostic {
	for i := range diags {
		if diags[i].Code == code {
			return &diags[i]
		}
	}
	return nil
}

func TestValidateEmpty(t *testing.T) {
	diags := Validate(Parse("[ hello ]"))
	if len(diags) != 0 {
		t.Errorf("Validate on clean source = %v, want empty", diags)
	}
}

func TestValidateUnsupportedData(t *testing.T) {
	// Tab-indented data: is the one toast-path error emitted today.
	src := "---\ndata:\n\t- x: 1\n\t  y: 10\n---\n[ x ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeUnsupportedData)
	if d == nil {
		t.Fatalf("no CodeUnsupportedData; got %+v", diags)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity=%d, want SeverityError", d.Severity)
	}
	if d.Message != "Unsupported data: frontmatter shape" {
		t.Errorf("Message=%q", d.Message)
	}
	if d.Span == (Span{}) {
		t.Error("Span is zero-value")
	}
}

func TestValidateDuplicateName(t *testing.T) {
	src := "[ x :: a ]\n[ y :: a ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeDuplicateName)
	if d == nil {
		t.Fatalf("no CodeDuplicateName; got %+v", diags)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity=%d, want SeverityError", d.Severity)
	}
	if d.Message != "Duplicate node name: a" {
		t.Errorf("Message=%q", d.Message)
	}
	// The duplicate's span should be on line 1 (the second declaration),
	// not line 0.
	if d.Span.Start.Line != 1 {
		t.Errorf("Span.Start.Line=%d, want 1 (second occurrence)", d.Span.Start.Line)
	}
}

func TestValidateUndefinedRef(t *testing.T) {
	src := "[ a ] -> &missing"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeUndefinedRef)
	if d == nil {
		t.Fatalf("no CodeUndefinedRef; got %+v", diags)
	}
	if d.Message != "Undefined ref: &missing" {
		t.Errorf("Message=%q", d.Message)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity=%d, want SeverityError", d.Severity)
	}
	if d.Span == (Span{}) {
		t.Error("Span is zero-value")
	}
}

func TestValidateUnknownRenderer(t *testing.T) {
	src := "[ hello | unknown ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeUnknownRenderer)
	if d == nil {
		t.Fatalf("no CodeUnknownRenderer; got %+v", diags)
	}
	if d.Message != "⚠ unknown renderer: unknown" {
		t.Errorf("Message=%q", d.Message)
	}
	if d.Severity != SeverityWarning {
		t.Errorf("Severity=%d, want SeverityWarning", d.Severity)
	}
}

func TestValidateUseAtRef(t *testing.T) {
	// Non-text renderer handed a raw string. `bar` keeps its literal
	// name (not hbar) in this specific message — JS's contract.
	src := "[ hello | bar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeUseAtRef)
	if d == nil {
		t.Fatalf("no CodeUseAtRef; got %+v", diags)
	}
	if d.Message != "⚠ bar: use @ref" {
		t.Errorf("Message=%q, want %q", d.Message, "⚠ bar: use @ref")
	}
}

func TestValidateBareDataRef(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeBareDataRef)
	if d == nil {
		t.Fatalf("no CodeBareDataRef; got %+v", diags)
	}
	if d.Message != "⚠ @data: requires | renderer" {
		t.Errorf("Message=%q, want %q", d.Message, "⚠ @data: requires | renderer")
	}
	if d.Severity != SeverityWarning {
		t.Errorf("Severity=%d, want SeverityWarning", d.Severity)
	}
}

func TestValidateDataNotFound(t *testing.T) {
	// A @ref that doesn't exist in meta.Data — either because Data is
	// nil, or because the named series isn't present.
	src := "[ @missing | bar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeDataNotFound)
	if d == nil {
		t.Fatalf("no CodeDataNotFound; got %+v", diags)
	}
	if d.Message != "⚠ @missing not found" {
		t.Errorf("Message=%q", d.Message)
	}
}

func TestValidateBarShape(t *testing.T) {
	// The YAML subset the frontmatter parser accepts can't express a
	// scalar series value, so we construct the offending Meta.Data
	// directly and re-run Validate — Validate is the subject here,
	// not parseFrontmatter.
	ast := Parse("[ @myseries | bar ]")
	ast.Meta.Data = map[string]interface{}{
		"myseries": "not an array",
	}
	diags := Validate(ast)
	d := findDiag(diags, CodeBarShape)
	if d == nil {
		t.Fatalf("no CodeBarShape; got %+v", diags)
	}
	if d.Message != "⚠ bar: expected [{x,y}]" {
		t.Errorf("Message=%q (bar alias threads its literal name through)", d.Message)
	}
}

func TestValidateBarEmpty(t *testing.T) {
	// Same path as BarShape: the YAML subset can't express an empty
	// series directly (the parser rejects empty map values). Inject
	// the shape manually and exercise the CodeBarEmpty emission.
	ast := Parse("[ @empty | bar ]")
	ast.Meta.Data = map[string]interface{}{
		"empty": []map[string]interface{}{},
	}
	diags := Validate(ast)
	d := findDiag(diags, CodeBarEmpty)
	if d == nil {
		t.Fatalf("no CodeBarEmpty; got %+v", diags)
	}
	if d.Message != "⚠ bar: empty series" {
		t.Errorf("Message=%q", d.Message)
	}
}

func TestValidateBarNegativeY(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | bar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeBarNegativeY)
	if d == nil {
		t.Fatalf("no CodeBarNegativeY; got %+v", diags)
	}
	if d.Message != "⚠ bar: negative y" {
		t.Errorf("Message=%q", d.Message)
	}
}

func TestValidateBarDuplicateX(t *testing.T) {
	src := "---\ndata:\n  - x: a\n    y: 2\n  - x: a\n    y: 10\n---\n[ @data | bar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeBarDuplicateX)
	if d == nil {
		t.Fatalf("no CodeBarDuplicateX; got %+v", diags)
	}
	if d.Message != `⚠ bar: duplicate x "a"` {
		t.Errorf("Message=%q", d.Message)
	}
}

func TestValidateBarAliasKeepsLiteralName(t *testing.T) {
	// JS threads the literal renderer name into validateBarSeries
	// (renderHBar(_, _, _, "bar")), so `bar` stays `bar` in diagnostics
	// instead of resolving to the internal `hbar`. This matches what
	// the user actually typed.
	src := "---\ndata:\n  - x: a\n    y: -1\n---\n[ @data | bar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeBarNegativeY)
	if d == nil {
		t.Fatalf("no CodeBarNegativeY; got %+v", diags)
	}
	if d.Message != "⚠ bar: negative y" {
		t.Errorf("Message=%q, want bar prefix (literal name, not alias target)", d.Message)
	}
}

func TestValidateVbar(t *testing.T) {
	src := "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | vbar ]"
	diags := Validate(Parse(src))
	d := findDiag(diags, CodeBarNegativeY)
	if d == nil {
		t.Fatalf("no CodeBarNegativeY; got %+v", diags)
	}
	if d.Message != "⚠ vbar: negative y" {
		t.Errorf("Message=%q, want vbar prefix", d.Message)
	}
}

func TestValidateTextRendererAcceptsAnything(t *testing.T) {
	// text renderer gets raw strings or refs — no use-@-ref complaint.
	src := "[ hello | text ]"
	diags := Validate(Parse(src))
	if len(diags) != 0 {
		t.Errorf("text renderer with raw string produced diagnostics: %v", diags)
	}
}

func TestValidateOrdering(t *testing.T) {
	// Source with multiple issues from different classes. Emission
	// order is: frontmatter → names → renderers → bar data.
	src := "---\ndata:\n\t- x: 1\n---\n[ x :: dup ]\n[ y :: dup ]\n[ @ref | unknown ]"
	diags := Validate(Parse(src))
	if len(diags) < 2 {
		t.Fatalf("expected multiple diagnostics; got %+v", diags)
	}
	// Frontmatter error must come before non-frontmatter ones.
	idxFront := -1
	idxName := -1
	idxRenderer := -1
	for i, d := range diags {
		switch d.Code {
		case CodeUnsupportedData:
			if idxFront == -1 {
				idxFront = i
			}
		case CodeDuplicateName:
			if idxName == -1 {
				idxName = i
			}
		case CodeUnknownRenderer:
			if idxRenderer == -1 {
				idxRenderer = i
			}
		}
	}
	if idxFront == -1 || idxName == -1 || idxRenderer == -1 {
		t.Fatalf("missing a class; got %+v", diags)
	}
	if !(idxFront < idxName && idxName < idxRenderer) {
		t.Errorf("ordering violation: front=%d name=%d renderer=%d", idxFront, idxName, idxRenderer)
	}
}

func TestValidateNilAST(t *testing.T) {
	// Defensive: Validate(nil) returns empty, not a panic.
	if got := Validate(nil); got != nil {
		t.Errorf("Validate(nil) = %v, want nil", got)
	}
}

// TestRendererInlineErrorCatalogue locks every renderer-level wire
// message byte-for-byte. The validator and the chart renderer both
// flow through rendererInlineError, so if one of these strings
// changes without this test being updated, the two surfaces would
// drift. Keep this test expensive to break.
func TestRendererInlineErrorCatalogue(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		dataHack interface{} // optional override for ast.Meta.Data
		wantCode Code
		wantMsg  string
	}{
		{
			name:     "bare-ref",
			src:      "---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data ]",
			wantCode: CodeBareDataRef,
			wantMsg:  "⚠ @data: requires | renderer",
		},
		{
			name:     "unknown-renderer",
			src:      "[ hello | foo ]",
			wantCode: CodeUnknownRenderer,
			wantMsg:  "⚠ unknown renderer: foo",
		},
		{
			name:     "use-at-ref-bar-alias",
			src:      "[ hello | bar ]",
			wantCode: CodeUseAtRef,
			wantMsg:  "⚠ bar: use @ref",
		},
		{
			name:     "use-at-ref-hbar",
			src:      "[ hello | hbar ]",
			wantCode: CodeUseAtRef,
			wantMsg:  "⚠ hbar: use @ref",
		},
		{
			name:     "use-at-ref-vbar",
			src:      "[ hello | vbar ]",
			wantCode: CodeUseAtRef,
			wantMsg:  "⚠ vbar: use @ref",
		},
		{
			name:     "data-not-found",
			src:      "[ @missing | bar ]",
			wantCode: CodeDataNotFound,
			wantMsg:  "⚠ @missing not found",
		},
		{
			name:     "bar-shape-bar-alias",
			src:      "[ @s | bar ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ bar: expected [{x,y}]",
		},
		{
			name:     "bar-shape-hbar",
			src:      "[ @s | hbar ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ hbar: expected [{x,y}]",
		},
		{
			name:     "bar-shape-vbar",
			src:      "[ @s | vbar ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ vbar: expected [{x,y}]",
		},
		{
			name:     "bar-empty-bar-alias",
			src:      "[ @s | bar ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ bar: empty series",
		},
		{
			name:     "bar-empty-hbar",
			src:      "[ @s | hbar ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ hbar: empty series",
		},
		{
			name:     "bar-empty-vbar",
			src:      "[ @s | vbar ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ vbar: empty series",
		},
		{
			name:     "bar-negative-y-bar-alias",
			src:      "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | bar ]",
			wantCode: CodeBarNegativeY,
			wantMsg:  "⚠ bar: negative y",
		},
		{
			name:     "bar-negative-y-hbar",
			src:      "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | hbar ]",
			wantCode: CodeBarNegativeY,
			wantMsg:  "⚠ hbar: negative y",
		},
		{
			name:     "bar-negative-y-vbar",
			src:      "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | vbar ]",
			wantCode: CodeBarNegativeY,
			wantMsg:  "⚠ vbar: negative y",
		},
		{
			name:     "bar-duplicate-x-bar-alias",
			src:      "---\ndata:\n  - x: a\n    y: 1\n  - x: a\n    y: 2\n---\n[ @data | bar ]",
			wantCode: CodeBarDuplicateX,
			wantMsg:  `⚠ bar: duplicate x "a"`,
		},
		{
			name:     "bar-duplicate-x-hbar",
			src:      "---\ndata:\n  - x: a\n    y: 1\n  - x: a\n    y: 2\n---\n[ @data | hbar ]",
			wantCode: CodeBarDuplicateX,
			wantMsg:  `⚠ hbar: duplicate x "a"`,
		},
		{
			name:     "bar-duplicate-x-vbar",
			src:      "---\ndata:\n  - x: a\n    y: 1\n  - x: a\n    y: 2\n---\n[ @data | vbar ]",
			wantCode: CodeBarDuplicateX,
			wantMsg:  `⚠ vbar: duplicate x "a"`,
		},
		{
			name:     "progress-not-number",
			src:      "[ hello | progress ]",
			wantCode: CodeProgressNotNumber,
			wantMsg:  "⚠ progress: expected number",
		},
		{
			name:     "progress-series-shape",
			src:      "[ @s | progress ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ progress: expected [{x,y}]",
		},
		{
			name:     "progress-series-empty",
			src:      "[ @s | progress ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ progress: empty series",
		},
		{
			name:     "progress-series-negative-y",
			src:      "---\ndata:\n  - x: 1\n    y: -5\n---\n[ @data | progress ]",
			wantCode: CodeBarNegativeY,
			wantMsg:  "⚠ progress: negative y",
		},
		{
			name:     "sparkline-shape",
			src:      "[ @s | sparkline ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ sparkline: expected [{x,y}]",
		},
		{
			name:     "sparkline-empty",
			src:      "[ @s | sparkline ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ sparkline: empty series",
		},
		{
			name:     "candlestick-shape",
			src:      "[ @s | candlestick ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ candlestick: expected [{x, o, h, l, c}]",
		},
		{
			name:     "candlestick-empty",
			src:      "[ @s | candlestick ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ candlestick: empty series",
		},
		{
			name:     "candlestick-invalid-ohlc",
			src:      "---\ndata:\n  - x: Mon\n    o: 10\n    h: 11\n    l: 8\n    c: 15\n---\n[ @data | candlestick ]",
			wantCode: CodeCandlestickOHLC,
			wantMsg:  "⚠ candlestick: invalid ohlc",
		},
		{
			name:     "hist-shape",
			src:      "[ @s | hist ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ hist: expected [{x,y}]",
		},
		{
			name:     "hist-empty",
			src:      "[ @s | hist ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ hist: empty series",
		},
		{
			name:     "step-shape",
			src:      "[ @s | step ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ step: expected [{x,y}]",
		},
		{
			name:     "step-empty",
			src:      "[ @s | step ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ step: empty series",
		},
		{
			name:     "gantt-shape",
			src:      "[ @s | gantt ]",
			dataHack: map[string]interface{}{"s": "not an array"},
			wantCode: CodeBarShape,
			wantMsg:  "⚠ gantt: expected [{x, start, end}]",
		},
		{
			name:     "gantt-empty",
			src:      "[ @s | gantt ]",
			dataHack: map[string]interface{}{"s": []map[string]interface{}{}},
			wantCode: CodeBarEmpty,
			wantMsg:  "⚠ gantt: empty series",
		},
		{
			name:     "gantt-invalid-range",
			src:      "---\ndata:\n  - x: spec\n    start: 5\n    end: 2\n---\n[ @data | gantt ]",
			wantCode: CodeGanttRange,
			wantMsg:  "⚠ gantt: invalid range",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ast := Parse(tc.src)
			if tc.dataHack != nil {
				ast.Meta.Data = tc.dataHack
			}
			b := firstErroringBox(ast)
			if b == nil {
				t.Fatalf("no box produced a renderer-level error in %q", tc.src)
			}
			code, msg, ok := rendererInlineError(b, ast.Meta)
			if !ok {
				t.Fatalf("rendererInlineError returned ok=false for picked box")
			}
			if code != tc.wantCode {
				t.Errorf("code = %q, want %q", code, tc.wantCode)
			}
			if msg != tc.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}

// TestRendererInlineErrorCleanBoxes asserts that boxes with no
// renderer-level issue return ok=false. Locks the contract that the
// helper never produces a spurious ⚠ for healthy sources.
func TestRendererInlineErrorCleanBoxes(t *testing.T) {
	sources := []string{
		"[ hello ]",
		"[ hello | text ]",
		"---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data | bar ]",
		"---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data | hbar ]",
		"---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data | vbar ]",
		"[ 75 | progress ]",
		"[ 150 | progress ]",
		"---\ndata:\n  - x: a\n    y: 50\n---\n[ @data | progress ]",
		"---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: -5\n---\n[ @data | sparkline ]",
		"---\ndata:\n  - x: 1\n    y: 5\n  - x: 1\n    y: 5\n---\n[ @data | sparkline ]",
	}
	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			ast := Parse(src)
			walkBoxes(ast, func(b *Box) {
				if _, _, ok := rendererInlineError(b, ast.Meta); ok {
					t.Errorf("unexpected renderer error on clean source %q (box span %+v)",
						src, b.Span)
				}
			})
		})
	}
}

// firstErroringBox returns the first Box (pre-order) that yields a
// renderer-level error. Test-only helper.
func firstErroringBox(ast *Box) *Box {
	var out *Box
	walkBoxes(ast, func(b *Box) {
		if out != nil {
			return
		}
		if _, _, ok := rendererInlineError(b, ast.Meta); ok {
			out = b
		}
	})
	return out
}
