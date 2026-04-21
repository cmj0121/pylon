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

	if got := h.Diagnostics(uri); got != nil {
		t.Errorf("Diagnostics(missing)=%v, want nil", got)
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
	// Clean source produces no diagnostics. U6 / U7 stubs still
	// return empty; tighten those when the feature bodies land.
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
	if got := h.SemanticTokens(uri); got == nil || len(got.Data) != 0 {
		t.Errorf("SemanticTokens(open)=%v, want empty-Data stub", got)
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
