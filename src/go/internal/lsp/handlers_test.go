package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

func TestHandlersEmpty(t *testing.T) {
	h := NewHandlers(NewStore())
	const uri = "file:///does-not-exist"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handler panicked on empty store: %v", r)
		}
	}()

	if got := h.Diagnostics(uri); got == nil || len(got) != 0 {
		t.Errorf("Diagnostics(missing)=%v, want empty slice", got)
	}
	if got := h.DocumentSymbols(uri); got != nil {
		t.Errorf("DocumentSymbols(missing)=%v, want nil", got)
	}
	if got := h.SemanticTokens(uri); got == nil {
		t.Error("SemanticTokens(missing)=nil, want non-nil empty tokens")
	} else if len(got.Data) != 0 {
		t.Errorf("SemanticTokens(missing).Data=%v, want empty", got.Data)
	}
}

func TestHandlersPopulatedCleanDoc(t *testing.T) {
	// Clean source produces no diagnostics and no document symbols.
	// Semantic tokens DO emit for "[ hello ]" — two bracket tokens
	// (opener + closer). Token coverage today: brackets + refs /
	// datarefs only; see tokens.go for the deferred-class list.
	store := NewStore()
	const uri = "file:///tmp/c.pylon"
	store.Open(uri, 1, "[ hello ]")
	h := NewHandlers(store)

	if got := h.Diagnostics(uri); len(got) != 0 {
		t.Errorf("Diagnostics(clean)=%v, want empty", got)
	}
	if got := h.DocumentSymbols(uri); len(got) != 0 {
		t.Errorf("DocumentSymbols(open)=%v, want empty stub", got)
	}
	if got := h.SemanticTokens(uri); got == nil {
		t.Errorf("SemanticTokens(open)=nil, want non-nil")
	} else if len(got.Data) != 10 {
		// 2 tokens × 5 uint32 each — the `[` and `]` operator tokens.
		t.Errorf("SemanticTokens(open).Data len=%d, want 10", len(got.Data))
	}
}

func TestHandlersDocumentSymbolsNamedNodes(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/named.pylon"
	store.Open(uri, 1, "[ x :: a ]\n[ y :: b ]")
	h := NewHandlers(store)

	syms := h.DocumentSymbols(uri)
	if len(syms) != 2 {
		t.Fatalf("len(syms) = %d, want 2 (%+v)", len(syms), syms)
	}
	if syms[0].Name != "a" || syms[1].Name != "b" {
		t.Errorf("names = %q, %q; want a, b", syms[0].Name, syms[1].Name)
	}
	for i, s := range syms {
		if s.Kind != protocol.SymbolKindVariable {
			t.Errorf("syms[%d].Kind = %d, want Variable", i, s.Kind)
		}
	}
	if syms[0].Range.Start.Line != 0 {
		t.Errorf("syms[0].Range.Start.Line = %d, want 0", syms[0].Range.Start.Line)
	}
	if syms[1].Range.Start.Line != 1 {
		t.Errorf("syms[1].Range.Start.Line = %d, want 1", syms[1].Range.Start.Line)
	}
}

func TestHandlersDocumentSymbolsDataSeries(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/keyed.pylon"
	src := "---\ndata:\n  counter:\n    - x: 1\n      y: 10\n  sales:\n    - x: 1\n      y: 100\n---\n[ x ]"
	store.Open(uri, 1, src)
	h := NewHandlers(store)

	syms := h.DocumentSymbols(uri)
	names := map[string]protocol.SymbolKind{}
	for _, s := range syms {
		names[s.Name] = s.Kind
	}
	if kind, ok := names["@counter"]; !ok || kind != protocol.SymbolKindConstant {
		t.Errorf("@counter = %v (ok=%v), want SymbolKindConstant", kind, ok)
	}
	if kind, ok := names["@sales"]; !ok || kind != protocol.SymbolKindConstant {
		t.Errorf("@sales = %v (ok=%v), want SymbolKindConstant", kind, ok)
	}
	// Both series share the DataSpan until per-key spans land in a
	// later refinement. Assert ranges line up with the `data:` block.
	for _, s := range syms {
		if s.Name[0] == '@' && s.Range.Start.Line == 0 {
			t.Errorf("%s.Range.Start.Line = 0; expected non-zero (inside data: section)", s.Name)
		}
	}
}

func TestHandlersDocumentSymbolsFlatData(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/flat.pylon"
	src := "---\ndata:\n  - x: 1\n    y: 10\n---\n[ @data | bar ]"
	store.Open(uri, 1, src)
	h := NewHandlers(store)

	syms := h.DocumentSymbols(uri)
	// Flat-list data exposes exactly one @data series (plus no named
	// nodes here).
	found := false
	for _, s := range syms {
		if s.Name == "@data" {
			found = true
			if s.Kind != protocol.SymbolKindConstant {
				t.Errorf("@data.Kind = %d, want Constant", s.Kind)
			}
		}
	}
	if !found {
		t.Fatalf("no @data symbol in %+v", syms)
	}
}

func TestHandlersDocumentSymbolsOrdering(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/mixed.pylon"
	// Declarations on lines 1, 5, 7 (after the data: section).
	src := "---\ndata:\n  counter:\n    - x: 1\n      y: 10\n---\n[ x :: first ]\n[ y :: second ]"
	store.Open(uri, 1, src)
	h := NewHandlers(store)

	syms := h.DocumentSymbols(uri)
	if len(syms) < 2 {
		t.Fatalf("expected at least 2 symbols, got %+v", syms)
	}
	// Output must be monotonically non-decreasing on (Line, Character).
	for i := 1; i < len(syms); i++ {
		prev := syms[i-1].Range.Start
		cur := syms[i].Range.Start
		if cur.Line < prev.Line || (cur.Line == prev.Line && cur.Character < prev.Character) {
			t.Errorf("ordering violation at %d: prev=%+v cur=%+v", i, prev, cur)
		}
	}
}

func TestHandlersDocumentSymbolsEmpty(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/plain.pylon"
	store.Open(uri, 1, "[ hello ]")
	h := NewHandlers(store)

	syms := h.DocumentSymbols(uri)
	if syms == nil {
		t.Fatal("DocumentSymbols on clean doc = nil; want empty slice")
	}
	if len(syms) != 0 {
		t.Errorf("DocumentSymbols on clean doc = %+v, want empty", syms)
	}
}

func TestHandlersDiagnosticsReturnsProtocolShape(t *testing.T) {
	// "[ a ] -> &missing" triggers CodeUndefinedRef — one diagnostic.
	store := NewStore()
	const uri = "file:///tmp/bad.pylon"
	store.Open(uri, 1, "[ a ] -> &missing")
	h := NewHandlers(store)

	diags := h.Diagnostics(uri)
	if len(diags) != 1 {
		t.Fatalf("len(Diagnostics) = %d, want 1 (got %+v)", len(diags), diags)
	}
	d := diags[0]
	if d.Code == nil {
		t.Fatal("Code is nil")
	}
	if d.Code.Value != string(pylon.CodeUndefinedRef) {
		t.Errorf("Code.Value = %v, want %q", d.Code.Value, pylon.CodeUndefinedRef)
	}
	if d.Source == nil || *d.Source != diagnosticSource {
		t.Errorf("Source = %v, want %q", d.Source, diagnosticSource)
	}
	if d.Severity == nil || *d.Severity != protocol.DiagnosticSeverityError {
		t.Errorf("Severity = %v, want Error", d.Severity)
	}
	if d.Message != "Undefined ref: &missing" {
		t.Errorf("Message = %q", d.Message)
	}
	// The &missing token starts at column 9 (0-based) of line 0.
	if d.Range.Start.Line != 0 || d.Range.Start.Character != 9 {
		t.Errorf("Range.Start = %+v, want line 0 col 9", d.Range.Start)
	}
}
