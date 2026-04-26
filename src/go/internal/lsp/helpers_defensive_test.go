package lsp

import (
	"testing"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// TestCollectSemanticTokensNilRoot exercises the early-return guard.
func TestCollectSemanticTokensNilRoot(t *testing.T) {
	if got := collectSemanticTokens(nil); got != nil {
		t.Errorf("collectSemanticTokens(nil) = %v, want nil", got)
	}
}

// TestEmitBracketTokensZeroSpan covers the synthetic-root guard. A Box
// with the zero Span (the synthetic default-example fallback) emits no
// tokens — the parser never builds one with a real source location of
// (0,0)-(0,0), so emitting brackets there would mis-highlight the
// origin.
func TestEmitBracketTokensZeroSpan(t *testing.T) {
	var out []semanticToken
	emitBracketTokens(&out, &pylon.Box{Span: pylon.Span{}})
	if len(out) != 0 {
		t.Errorf("zero-span box emitted %d tokens, want 0", len(out))
	}
}

// TestEmitBracketTokensZeroEndColumn covers the End.Column == 0 guard.
// A box with non-zero start but End at column 0 means the closer is at
// "column -1" — the function bails after emitting only the opener.
func TestEmitBracketTokensZeroEndColumn(t *testing.T) {
	var out []semanticToken
	b := &pylon.Box{Span: pylon.Span{
		Start: pylon.Position{Line: 1, Column: 5},
		End:   pylon.Position{Line: 1, Column: 0},
	}}
	emitBracketTokens(&out, b)
	if len(out) != 1 {
		t.Fatalf("End.Column==0 path emitted %d tokens, want 1 (opener only)", len(out))
	}
	if out[0].StartChar != 5 {
		t.Errorf("opener StartChar = %d, want 5", out[0].StartChar)
	}
}

// TestEmitVariableTokenZeroSpan covers the zero-span guard.
func TestEmitVariableTokenZeroSpan(t *testing.T) {
	var out []semanticToken
	emitVariableToken(&out, pylon.Span{}, 0)
	if len(out) != 0 {
		t.Errorf("zero-span emitted %d tokens, want 0", len(out))
	}
}

// TestEmitVariableTokenMultiLineSpan covers the multi-line guard.
// Identifiers can't legally span lines in the grammar, but the helper
// drops any cross-line span defensively.
func TestEmitVariableTokenMultiLineSpan(t *testing.T) {
	var out []semanticToken
	emitVariableToken(&out, pylon.Span{
		Start: pylon.Position{Line: 0, Column: 5},
		End:   pylon.Position{Line: 1, Column: 2},
	}, modReadonly)
	if len(out) != 0 {
		t.Errorf("multi-line span emitted %d tokens, want 0", len(out))
	}
}

// TestNamedNodeSymbolsNilRoot covers the early-return guard in
// namedNodeSymbols. Calling with nil returns nil, not an empty slice.
func TestNamedNodeSymbolsNilRoot(t *testing.T) {
	if got := namedNodeSymbols(nil); got != nil {
		t.Errorf("namedNodeSymbols(nil) = %v, want nil", got)
	}
}

// TestDataSeriesSymbolsNilRoot covers the early-return guard.
func TestDataSeriesSymbolsNilRoot(t *testing.T) {
	if got := dataSeriesSymbols(nil); got != nil {
		t.Errorf("dataSeriesSymbols(nil) = %v, want nil", got)
	}
}

// TestDataSeriesSymbolsUnknownDataShape covers the default arm of the
// type switch. The frontmatter parser only ever produces []map or map
// shapes; an unknown shape (e.g. a bare scalar from a malformed parse)
// returns nil.
func TestDataSeriesSymbolsUnknownDataShape(t *testing.T) {
	root := &pylon.Box{}
	root.Meta.Data = "not-a-list-or-map"
	if got := dataSeriesSymbols(root); got != nil {
		t.Errorf("default-arm = %v, want nil", got)
	}
}

// TestNamedNodeSymbolsViaEdgeLabel exercises the *Edge case in
// namedNodeSymbols by parsing a source where a named box rides as an
// edge label. The named-symbol traversal must descend into the label.
func TestNamedNodeSymbolsViaEdgeLabel(t *testing.T) {
	// Edge label syntax: `[ a ] -[ x :: tagged ]-> [ b ]` — the middle
	// box becomes the edge's label. Whether tagged surfaces as a
	// symbol depends on parser handling, but the traversal path runs
	// either way.
	store := NewStore()
	const uri = "file:///tmp/edge.pylon"
	store.Open(uri, 1, "[ a ] -[ x :: tagged ]-> [ b ]")
	h := NewHandlers(store)

	// Just exercise the path — no assertion on count, since the parser
	// may or may not promote the label box to a named declaration.
	_ = h.DocumentSymbols(uri)
}
