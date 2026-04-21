package lsp

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Handlers is the transport-free surface every LSP feature attaches to.
// Each method takes a URI string, reads from the Store, and returns a
// protocol-shaped response. No glsp.Context reaches here — tests seed
// a Store and call the methods directly.
//
// The method set is deliberately over-provisioned for U4: the feature
// units (U5 diagnostics, U6 document symbols, U7 semantic tokens) each
// replace one method's body. For U4 every method returns an empty /
// nil result so the server stays a correct but feature-less scaffold.
type Handlers struct {
	Store *Store
}

// NewHandlers constructs a Handlers over the given Store.
func NewHandlers(store *Store) *Handlers {
	return &Handlers{Store: store}
}

// Diagnostics returns the cached pylon diagnostics for uri translated
// into the LSP protocol shape. Translation only — Parse + Validate
// ran when the Store.Open / Change notification landed. Returns nil
// for unknown URIs; returns an empty slice (not nil) when the cached
// document has no diagnostics, so callers can distinguish "clean
// state" from "unknown document" without an extra probe.
func (h *Handlers) Diagnostics(uri string) []protocol.Diagnostic {
	doc, ok := h.Store.Get(uri)
	if !ok {
		return nil
	}
	out := make([]protocol.Diagnostic, 0, len(doc.Diagnostics))
	for _, d := range doc.Diagnostics {
		out = append(out, toProtocolDiagnostic(d))
	}
	return out
}

// DocumentSymbols returns the symbols for uri. U4 stub returns nil;
// U6 fills this with named-node and data-series collection.
func (h *Handlers) DocumentSymbols(uri string) []protocol.DocumentSymbol {
	if _, ok := h.Store.Get(uri); !ok {
		return nil
	}
	return nil
}

// SemanticTokens returns the tokens for uri. U4 stub returns an empty
// SemanticTokens (Data: []uint32{}); U7 swaps in a real encoder.
func (h *Handlers) SemanticTokens(uri string) *protocol.SemanticTokens {
	if _, ok := h.Store.Get(uri); !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}
	}
	return &protocol.SemanticTokens{Data: []uint32{}}
}
