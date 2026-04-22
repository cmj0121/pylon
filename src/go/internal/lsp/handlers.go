package lsp

import (
	"sort"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
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

// DocumentSymbols returns the outline for uri: every `:: name`
// declaration as a Variable symbol, plus each frontmatter `data:`
// series as a Constant named `@key`. Returns nil for unknown URIs;
// returns an empty slice (not nil) when the document is known but
// carries no symbols, so callers can distinguish "clean outline"
// from "unknown document".
//
// Per-series-key spans are not tracked by the frontmatter parser
// today (Meta.DataSpan covers the whole `data:` section), so every
// series symbol shares that range. Narrowing is a future refinement
// that can ride on a later parser change.
func (h *Handlers) DocumentSymbols(uri string) []protocol.DocumentSymbol {
	doc, ok := h.Store.Get(uri)
	if !ok {
		return nil
	}
	out := []protocol.DocumentSymbol{}
	out = append(out, namedNodeSymbols(doc.AST)...)
	out = append(out, dataSeriesSymbols(doc.AST)...)
	sortSymbolsByStart(out)
	return out
}

// namedNodeSymbols walks the AST collecting one symbol per `:: name`
// declaration. Named Boxes can be nested and can appear inside edge
// labels; the traversal visits every reachable Box.
func namedNodeSymbols(root pylon.AST) []protocol.DocumentSymbol {
	if root == nil {
		return nil
	}
	var out []protocol.DocumentSymbol
	var visit func(n pylon.Node)
	visit = func(n pylon.Node) {
		if n == nil {
			return
		}
		switch x := n.(type) {
		case *pylon.Box:
			if x.Name != "" {
				r := spanToRange(x.Span)
				out = append(out, protocol.DocumentSymbol{
					Name:           x.Name,
					Kind:           protocol.SymbolKindVariable,
					Range:          r,
					SelectionRange: r,
				})
			}
			for _, c := range x.Items {
				visit(c)
			}
		case *pylon.Row:
			for _, c := range x.Items {
				visit(c)
			}
		case *pylon.Edge:
			if x.Label != nil {
				visit(x.Label)
			}
		}
	}
	visit(root)
	return out
}

// dataSeriesSymbols emits one symbol per resolvable `@series`
// reference in the frontmatter `data:` block. A flat list surfaces as
// a single `@data`; a map-keyed data block exposes each key as
// `@<key>`. Each symbol takes the whole DataSpan as its range for v1
// — per-key spans are a future refinement.
func dataSeriesSymbols(root pylon.AST) []protocol.DocumentSymbol {
	if root == nil {
		return nil
	}
	if root.Meta.Data == nil {
		return nil
	}
	r := spanToRange(root.Meta.DataSpan)
	mk := func(name string) protocol.DocumentSymbol {
		return protocol.DocumentSymbol{
			Name:           name,
			Kind:           protocol.SymbolKindConstant,
			Range:          r,
			SelectionRange: r,
		}
	}
	switch d := root.Meta.Data.(type) {
	case []map[string]interface{}:
		return []protocol.DocumentSymbol{mk("@data")}
	case map[string]interface{}:
		out := make([]protocol.DocumentSymbol, 0, len(d))
		for k := range d {
			out = append(out, mk("@"+k))
		}
		return out
	}
	return nil
}

// sortSymbolsByStart stabilises the symbol slice by Range.Start so
// tests and editors see deterministic ordering.
func sortSymbolsByStart(syms []protocol.DocumentSymbol) {
	sort.SliceStable(syms, func(i, j int) bool {
		a, b := syms[i].Range.Start, syms[j].Range.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Character != b.Character {
			return a.Character < b.Character
		}
		return syms[i].Name < syms[j].Name
	})
}

// SemanticTokens returns the LSP-wire-encoded token stream for uri.
// Partial implementation: brackets plus &/@ references only; see
// internal/lsp/tokens.go for the legend and deferred-class list.
// Keeps the U5/U6 "non-nil empty for known clean doc, nil-ish for
// unknown URI" convention — but LSP clients expect a value, so
// unknown URIs still get an empty Data slice instead of nil.
func (h *Handlers) SemanticTokens(uri string) *protocol.SemanticTokens {
	doc, ok := h.Store.Get(uri)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}
	}
	raw := collectSemanticTokens(doc.AST)
	raw = sortAndValidateTokens(raw)
	return &protocol.SemanticTokens{Data: encodeTokens(raw)}
}
