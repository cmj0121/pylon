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
