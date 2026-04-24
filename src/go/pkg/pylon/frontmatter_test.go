package pylon

import "testing"

// TestFrontmatterSizeSpan asserts Meta.SizeSpan covers the source line
// containing the `size:` declaration.
func TestFrontmatterSizeSpan(t *testing.T) {
	src := "---\nsize: 30x5\n---\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Meta.Size == nil {
		t.Fatal("Meta.Size not populated")
	}
	if got := root.Meta.SizeSpan; got.Start.Line != 1 || got.Start.Column != 0 {
		t.Errorf("SizeSpan.Start = %+v, want line 1 col 0", got.Start)
	}
	wantEndCol := len("size: 30x5")
	if got := root.Meta.SizeSpan; got.End.Line != 1 || got.End.Column != wantEndCol {
		t.Errorf("SizeSpan.End = %+v, want line 1 col %d", got.End, wantEndCol)
	}
}

// TestFrontmatterThemeSpan asserts Meta.ThemeSpan covers the source
// line containing the `theme:` declaration.
func TestFrontmatterThemeSpan(t *testing.T) {
	src := "---\ntheme: ascii\n---\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Meta.Theme != "ascii" {
		t.Fatalf("Meta.Theme = %q, want ascii", root.Meta.Theme)
	}
	if got := root.Meta.ThemeSpan; got.Start.Line != 1 || got.Start.Column != 0 {
		t.Errorf("ThemeSpan.Start = %+v, want line 1 col 0", got.Start)
	}
	wantEndCol := len("theme: ascii")
	if got := root.Meta.ThemeSpan; got.End.Line != 1 || got.End.Column != wantEndCol {
		t.Errorf("ThemeSpan.End = %+v, want line 1 col %d", got.End, wantEndCol)
	}
}

// TestFrontmatterDataSpan asserts Meta.DataSpan covers the header line
// through the end of the last accumulated section line.
func TestFrontmatterDataSpan(t *testing.T) {
	// Line 0: ---
	// Line 1: data:
	// Line 2:   - x: 1
	// Line 3:     y: 10
	// Line 4: ---
	// Line 5: [ x ]
	src := "---\ndata:\n  - x: 1\n    y: 10\n---\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Meta.Data == nil {
		t.Fatalf("Meta.Data not populated; Errors=%v", root.Meta.Errors)
	}
	ds := root.Meta.DataSpan
	if ds.Start.Line != 1 || ds.Start.Column != 0 {
		t.Errorf("DataSpan.Start = %+v, want line 1 col 0", ds.Start)
	}
	// End points past the last byte of line 3 ("    y: 10").
	wantEndCol := len("    y: 10")
	if ds.End.Line != 3 || ds.End.Column != wantEndCol {
		t.Errorf("DataSpan.End = %+v, want line 3 col %d", ds.End, wantEndCol)
	}
}

// TestFrontmatterUnsupportedDataError asserts that a tab-indented
// data: section produces exactly one MetaError with the SPEC-canonical
// wording and a Span over the data: section.
func TestFrontmatterUnsupportedDataError(t *testing.T) {
	// Line 0: ---
	// Line 1: data:
	// Line 2: \t- x: 1        (tab indent — rejects the whole section)
	// Line 3: \t  y: 10
	// Line 4: ---
	// Line 5: [ x ]
	src := "---\ndata:\n\t- x: 1\n\t  y: 10\n---\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if got, want := len(root.Meta.Errors), 1; got != want {
		t.Fatalf("len(Meta.Errors) = %d, want %d (%#v)", got, want, root.Meta.Errors)
	}
	e := root.Meta.Errors[0]
	if want := "Unsupported data: frontmatter shape"; e.Message != want {
		t.Errorf("Error Message = %q, want %q", e.Message, want)
	}
	if e.Span.Start.Line != 1 || e.Span.Start.Column != 0 {
		t.Errorf("Error Span.Start = %+v, want line 1 col 0", e.Span.Start)
	}
	// End past line 3 ("\t  y: 10", 8 bytes which count as 8 UTF-16 units
	// since tab and ASCII are each 1 unit).
	wantEndCol := len("\t  y: 10")
	if e.Span.End.Line != 3 || e.Span.End.Column != wantEndCol {
		t.Errorf("Error Span.End = %+v, want line 3 col %d", e.Span.End, wantEndCol)
	}
	// Meta.Data stays nil on the error path.
	if root.Meta.Data != nil {
		t.Errorf("Meta.Data = %v, want nil on error path", root.Meta.Data)
	}
	// The error message is also surfaced on root.Errors (string form) so
	// the existing toast pipeline keeps working.
	found := false
	for _, s := range root.Errors {
		if s == "Unsupported data: frontmatter shape" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("root.Errors did not carry the toast message; got %v", root.Errors)
	}
}

// TestFrontmatterKeySpansMultipleKeys asserts size and theme spans
// land on their respective distinct lines without overlap.
func TestFrontmatterKeySpansMultipleKeys(t *testing.T) {
	src := "---\nsize: 20x4\ntheme: ascii\n---\n[ x ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Meta.Size == nil {
		t.Fatal("Meta.Size not populated")
	}
	if root.Meta.Theme != "ascii" {
		t.Fatalf("Meta.Theme = %q, want ascii", root.Meta.Theme)
	}
	sz := root.Meta.SizeSpan
	th := root.Meta.ThemeSpan
	if sz == (Span{}) || th == (Span{}) {
		t.Fatalf("zero-value span (Size=%+v Theme=%+v)", sz, th)
	}
	if sz.Start.Line != 1 || th.Start.Line != 2 {
		t.Errorf("lines: size=%d theme=%d; want 1, 2", sz.Start.Line, th.Start.Line)
	}
	// The two spans must not overlap.
	if sz.End.Line == th.Start.Line && sz.End.Column > th.Start.Column {
		t.Errorf("SizeSpan %+v overlaps ThemeSpan %+v", sz, th)
	}
	// Sanity: neither span extends beyond its own line end in column.
	if sz.End.Column > len("size: 20x4") {
		t.Errorf("SizeSpan.End.Column too large: %d", sz.End.Column)
	}
	if th.End.Column > len("theme: ascii") {
		t.Errorf("ThemeSpan.End.Column too large: %d", th.End.Column)
	}
}

// TestFrontmatterNoFrontmatter asserts zero-value Meta spans when no
// frontmatter block is present — guards against false positives.
func TestFrontmatterNoFrontmatter(t *testing.T) {
	src := "[ hello ]"
	root := Parse(src)
	if root == nil {
		t.Fatal("Parse returned nil")
	}
	if root.Meta.SizeSpan != (Span{}) || root.Meta.ThemeSpan != (Span{}) || root.Meta.DataSpan != (Span{}) {
		t.Errorf("non-zero key spans without a frontmatter block: %+v", root.Meta)
	}
	if len(root.Meta.Errors) != 0 {
		t.Errorf("unexpected Meta.Errors without frontmatter: %v", root.Meta.Errors)
	}
}

// TestFrontmatterColor asserts Meta.Color picks up explicit
// `color: true` / `color: false` and stays nil when the key is
// absent or malformed. Other values are silently ignored (no
// toast) to match how `theme:` and `size:` handle bad input.
func TestFrontmatterColor(t *testing.T) {
	cases := []struct {
		src     string
		want    *bool
		wantSet bool
	}{
		{"---\ncolor: true\n---\n[- x -]", boolPtr(true), true},
		{"---\ncolor: false\n---\n[- x -]", boolPtr(false), true},
		{"---\ncolor: maybe\n---\n[- x -]", nil, false},
		{"---\n---\n[- x -]", nil, false},
		{"[- x -]", nil, false},
	}
	for _, tc := range cases {
		root := Parse(tc.src)
		got := root.Meta.Color
		if (got == nil) != (tc.want == nil) {
			t.Errorf("Meta.Color nil-ness mismatch for %q: got=%v want=%v",
				tc.src, got, tc.want)
			continue
		}
		if got != nil && *got != *tc.want {
			t.Errorf("Meta.Color = %v, want %v for %q",
				*got, *tc.want, tc.src)
		}
		if tc.wantSet && root.Meta.ColorSpan == (Span{}) {
			t.Errorf("expected non-zero ColorSpan for %q", tc.src)
		}
	}
}

func boolPtr(b bool) *bool { return &b }
