package pylon

import (
	"os"
	"testing"
)

// TestSpansIncompleteBracket asserts the parser stays resilient under
// malformed input — a truncated source like "[ Start" with no matching
// close bracket must not panic and must produce a usable AST.
func TestSpansIncompleteBracket(t *testing.T) {
	src, err := os.ReadFile("testdata/incomplete-bracket.pylon")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var root *Box
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Parse panicked on incomplete bracket: %v", r)
			}
		}()
		root = Parse(string(src))
	}()
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	// The `[` is unmatched, so it falls through to textual content.
	// The synthetic bare-text root wraps it in a bordered box with a
	// single Text child whose content begins with '['.
	if len(root.Items) == 0 {
		t.Fatal("root has no items")
	}
	txt, ok := root.Items[0].(*Text)
	if !ok {
		t.Fatalf("root.Items[0] type = %T, want *Text", root.Items[0])
	}
	if len(txt.Content) == 0 || txt.Content[0] != '[' {
		t.Errorf("text content %q does not start with '['", txt.Content)
	}
}

// TestSpansUTF16Columns asserts that the Span end column for a box
// containing CJK characters is counted in UTF-16 code units — the LSP
// wire format — rather than bytes. "[ 你好世界 ]" has 8 UTF-16 code
// units (`[`, space, 4 CJK chars × 1 unit each, space, `]`), so the
// closing bracket's End.Column must be 8. The byte length of the same
// source is 15 (each CJK char = 3 UTF-8 bytes), which would be wrong.
func TestSpansUTF16Columns(t *testing.T) {
	src, err := os.ReadFile("testdata/utf8-label.pylon")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	root := Parse(string(src))
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	// The outer root IS the box (single-item shortcut — Parse returns
	// the inner box directly). Its Span must cover the whole source.
	if root.Span.Start.Line != 0 || root.Span.Start.Column != 0 {
		t.Errorf("root Span.Start = %+v, want line 0 col 0", root.Span.Start)
	}
	if root.Span.End.Line != 0 {
		t.Errorf("root Span.End.Line = %d, want 0", root.Span.End.Line)
	}
	// "[ 你好世界 ]" → 8 UTF-16 code units total. End.Column is exclusive
	// end position, so End.Column should equal 8.
	if root.Span.End.Column != 8 {
		t.Errorf("root Span.End.Column = %d, want 8 (UTF-16 units, not bytes)", root.Span.End.Column)
	}
}

// TestSpansFlowChain exercises a multi-node row with refs to confirm
// the span machinery handles nested parseItems sub-sources correctly.
func TestSpansFlowChain(t *testing.T) {
	// "[ a :: x ]\n[ b ] -> &x"
	//   Box a at line 0, cols 0..10
	//   Box b at line 1, cols 0..5
	//   Ref &x at line 1, cols 9..11
	src := "[ a :: x ]\n[ b ] -> &x"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	// Root has two top-level items: the named box and a Row.
	if got, want := len(root.Items), 2; got != want {
		t.Fatalf("len(root.Items) = %d, want %d (%#v)", got, want, root.Items)
	}
	box0, ok := root.Items[0].(*Box)
	if !ok {
		t.Fatalf("root.Items[0] type = %T, want *Box", root.Items[0])
	}
	if box0.Span.Start.Line != 0 || box0.Span.Start.Column != 0 {
		t.Errorf("box0 Span.Start = %+v, want line 0 col 0", box0.Span.Start)
	}
	if box0.Span.End.Line != 0 || box0.Span.End.Column != 10 {
		t.Errorf("box0 Span.End = %+v, want line 0 col 10", box0.Span.End)
	}

	row, ok := root.Items[1].(*Row)
	if !ok {
		t.Fatalf("root.Items[1] type = %T, want *Row", root.Items[1])
	}
	// Row contains: box b, edge, ref &x.
	var ref *Ref
	for _, it := range row.Items {
		if r, ok := it.(*Ref); ok {
			ref = r
			break
		}
	}
	if ref == nil {
		t.Fatalf("no Ref in row.Items (%#v)", row.Items)
	}
	if ref.Name != "x" {
		t.Errorf("ref.Name = %q, want %q", ref.Name, "x")
	}
	if ref.Span.Start.Line != 1 || ref.Span.Start.Column != 9 {
		t.Errorf("ref Span.Start = %+v, want line 1 col 9", ref.Span.Start)
	}
	if ref.Span.End.Line != 1 || ref.Span.End.Column != 11 {
		t.Errorf("ref Span.End = %+v, want line 1 col 11", ref.Span.End)
	}
}

// TestSpansDataRef asserts @ident spans cover the sigil + identifier.
func TestSpansDataRef(t *testing.T) {
	src := "[ @data ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	// Root is the box. Find the DataRef inside.
	var dref *DataRef
	for _, it := range root.Items {
		if d, ok := it.(*DataRef); ok {
			dref = d
			break
		}
	}
	if dref == nil {
		t.Fatalf("no DataRef in root.Items (%#v)", root.Items)
	}
	if dref.Name != "data" {
		t.Errorf("dref.Name = %q, want %q", dref.Name, "data")
	}
	// "@data" starts at column 2 (`[`, space, `@`), spans 5 UTF-16 units.
	if dref.Span.Start.Column != 2 {
		t.Errorf("dref Span.Start.Column = %d, want 2", dref.Span.Start.Column)
	}
	if dref.Span.End.Column != 7 {
		t.Errorf("dref Span.End.Column = %d, want 7", dref.Span.End.Column)
	}
}

// TestSpansPositionFromTrimmedLead asserts the parser preserves the
// absolute line/column of source content even when the document has
// leading whitespace that TrimSpace would strip.
func TestSpansPositionFromTrimmedLead(t *testing.T) {
	src := "\n\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Span.Start.Line != 2 || root.Span.Start.Column != 0 {
		t.Errorf("root Span.Start = %+v, want line 2 col 0", root.Span.Start)
	}
}
