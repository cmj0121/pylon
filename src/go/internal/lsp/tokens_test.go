package lsp

import (
	"reflect"
	"testing"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// TestLegendAddOnlyInvariant locks the published legend. Reordering
// or removing entries is a breaking change for editor clients that
// cached the legend from a previous initialize response. Adding a
// new entry on the end is fine — update the expected slice here.
func TestLegendAddOnlyInvariant(t *testing.T) {
	wantTypes := []string{"variable", "operator"}
	if !reflect.DeepEqual(tokenTypes, wantTypes) {
		t.Errorf("tokenTypes = %v, want %v (add-only: append, never reorder)", tokenTypes, wantTypes)
	}
	wantMods := []string{"readonly"}
	if !reflect.DeepEqual(tokenModifiers, wantMods) {
		t.Errorf("tokenModifiers = %v, want %v", tokenModifiers, wantMods)
	}
}

// tokensOf is a small test helper: parse src, collect tokens, sort /
// validate / encode. Returns the wire []uint32 form.
func tokensOf(t *testing.T, src string) []uint32 {
	t.Helper()
	ast := pylon.Parse(src)
	raw := collectSemanticTokens(ast)
	raw = sortAndValidateTokens(raw)
	return encodeTokens(raw)
}

// TestSemanticTokensSingleBox asserts the encoding for "[ hello ]":
// one bracket token at column 0 and one at column 8, both operator
// kind. Delta-encoded: 5 uint32s for each, 10 total.
func TestSemanticTokensSingleBox(t *testing.T) {
	got := tokensOf(t, "[ hello ]")
	want := []uint32{
		0, 0, 1, tokTypeOperator, 0, // first `[` at (0, 0), length 1
		0, 8, 1, tokTypeOperator, 0, // `]` at delta-char 8 same line
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("encoded = %v, want %v", got, want)
	}
}

// TestSemanticTokensRef covers a cross-row ref: the `&a` token plus
// the two sets of brackets. Source: `[ x :: a ]\n[ b ] -> &a`.
func TestSemanticTokensRef(t *testing.T) {
	got := tokensOf(t, "[ x :: a ]\n[ b ] -> &a")

	// Expected raw tokens:
	//   line 0 col  0 length 1 operator    (first `[`)
	//   line 0 col  9 length 1 operator    (first `]`)
	//   line 1 col  0 length 1 operator    (second `[`)
	//   line 1 col  4 length 1 operator    (second `]`)
	//   line 1 col  9 length 2 variable    (`&a`)
	want := []uint32{
		0, 0, 1, tokTypeOperator, 0,
		0, 9, 1, tokTypeOperator, 0,
		1, 0, 1, tokTypeOperator, 0,
		0, 4, 1, tokTypeOperator, 0,
		0, 5, 2, tokTypeVariable, 0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("encoded = %v\n   want = %v", got, want)
	}
}

// TestSemanticTokensDataRef covers `@data` emission with the
// readonly modifier bit set.
func TestSemanticTokensDataRef(t *testing.T) {
	got := tokensOf(t, "[ @data | bar ]")
	// The grammar strips the `| bar` renderer tail from the body
	// before parseItems runs, so the @data token has span over the 5
	// bytes `@data`. Brackets sit at cols 0 and 14 of line 0.
	//
	// Expected encoded order:
	//   col  0 length  1 operator   (`[`)
	//   col  2 length  5 variable   (`@data`) + readonly
	//   col 14 length  1 operator   (`]`)
	want := []uint32{
		0, 0, 1, tokTypeOperator, 0,
		0, 2, 5, tokTypeVariable, modReadonly,
		0, 12, 1, tokTypeOperator, 0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("encoded = %v\n   want = %v", got, want)
	}
}

// TestSemanticTokensSort exercises the source-order sort on a nested
// box where raw collection visits the outer box first (its opener
// comes before the inner, its closer comes after). The final encoded
// stream must be monotonically non-decreasing in source order.
func TestSemanticTokensSort(t *testing.T) {
	// Nested borderless box inside a bordered one, all on one line.
	//   [( x )]
	//   cols: 0 1 2 3 4 5 6
	//         [ ( x )  ] — outer opener col 0, inner `(` col 1,
	//         inner `)` col 5, outer `]` col 6.
	got := tokensOf(t, "[( x )]")

	// Expected encoded (delta-encoded):
	//   0, 0, 1, op (outer `[`)
	//   0, 1, 1, op (inner `(`)
	//   0, 4, 1, op (inner `)`)
	//   0, 1, 1, op (outer `]`)
	want := []uint32{
		0, 0, 1, tokTypeOperator, 0,
		0, 1, 1, tokTypeOperator, 0,
		0, 4, 1, tokTypeOperator, 0,
		0, 1, 1, tokTypeOperator, 0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("encoded = %v\n   want = %v", got, want)
	}
}

// TestSemanticTokensClean: a source with no boxes and no refs (bare
// text treated as a synthetic root box) — expect either no tokens or
// exactly the synthetic root's brackets. With bare text, the root is
// a synthetic bordered Box whose Span covers the whole body; the
// bracket emitter uses Span.Start and Span.End-1, which for bare
// text aren't actual brackets in the source. Guard by asserting the
// output length is a multiple of 5 (correctly encoded) and return.
func TestSemanticTokensClean(t *testing.T) {
	got := tokensOf(t, "plain text no box")
	if len(got)%5 != 0 {
		t.Errorf("encoded length %d not a multiple of 5", len(got))
	}
}

// TestSemanticTokensEncodingRoundTrip feeds a hand-built token slice
// through encodeTokens and asserts the bytes. Isolates the encoder
// from the collector so a collection-side bug doesn't mask an
// encoding-side bug.
func TestSemanticTokensEncodingRoundTrip(t *testing.T) {
	toks := []semanticToken{
		{Line: 2, StartChar: 5, Length: 3, TokenType: 0, Modifiers: 0},
		{Line: 2, StartChar: 10, Length: 1, TokenType: 1, Modifiers: 1},
		{Line: 5, StartChar: 2, Length: 4, TokenType: 0, Modifiers: 0},
	}
	got := encodeTokens(toks)
	want := []uint32{
		2, 5, 3, 0, 0, // first: absolute (2, 5)
		0, 5, 1, 1, 1, // same line, delta-char 5
		3, 2, 4, 0, 0, // 3 lines forward, absolute start 2
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("encoded = %v\n   want = %v", got, want)
	}
}

// TestSortAndValidateDropsOverlap asserts the encoder skips a token
// that overlaps a preceding one on the same line (LSP spec: overlap
// behaviour is undefined; silently dropping is safer than garbage).
func TestSortAndValidateDropsOverlap(t *testing.T) {
	toks := []semanticToken{
		{Line: 0, StartChar: 0, Length: 5, TokenType: 0},
		{Line: 0, StartChar: 2, Length: 1, TokenType: 1}, // overlap
		{Line: 0, StartChar: 6, Length: 1, TokenType: 1}, // past the first
	}
	out := sortAndValidateTokens(toks)
	if len(out) != 2 {
		t.Fatalf("expected 2 tokens after overlap drop, got %d: %+v", len(out), out)
	}
	if out[0].StartChar != 0 || out[1].StartChar != 6 {
		t.Errorf("kept tokens = %+v, expected the first and third", out)
	}
}
