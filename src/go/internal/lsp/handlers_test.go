package lsp

import "testing"

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

func TestHandlersPopulatedDoc(t *testing.T) {
	store := NewStore()
	const uri = "file:///tmp/c.pylon"
	store.Open(uri, 1, "[ hello ]")
	h := NewHandlers(store)

	// U4 stubs return empty; the contract is "does not panic and types
	// line up with the protocol package." U5/U6/U7 tighten the
	// assertions when they fill the bodies in.
	if got := h.Diagnostics(uri); len(got) != 0 {
		t.Errorf("Diagnostics(open)=%v, want empty stub", got)
	}
	if got := h.DocumentSymbols(uri); len(got) != 0 {
		t.Errorf("DocumentSymbols(open)=%v, want empty stub", got)
	}
	if got := h.SemanticTokens(uri); got == nil || len(got.Data) != 0 {
		t.Errorf("SemanticTokens(open)=%v, want empty-Data stub", got)
	}
}
