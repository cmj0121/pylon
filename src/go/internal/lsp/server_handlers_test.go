package lsp

import (
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// captureCtx returns a glsp.Context whose Notify callback appends every
// (method, params) tuple into the supplied notes slice. Lets tests
// observe publishDiagnostics fan-out without standing up a real server.
func captureCtx(notes *[]capturedNote) *glsp.Context {
	return &glsp.Context{
		Notify: func(method string, params any) {
			*notes = append(*notes, capturedNote{Method: method, Params: params})
		},
	}
}

type capturedNote struct {
	Method string
	Params any
}

func TestOnInitializeAdvertisesCapabilities(t *testing.T) {
	res, err := onInitialize(&glsp.Context{}, &protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("onInitialize err = %v", err)
	}
	r, ok := res.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("onInitialize result type = %T, want InitializeResult", res)
	}
	if r.ServerInfo == nil || r.ServerInfo.Name != lsName {
		t.Errorf("ServerInfo.Name = %v, want %q", r.ServerInfo, lsName)
	}
	if r.ServerInfo.Version == nil || *r.ServerInfo.Version != lsVersion {
		t.Errorf("ServerInfo.Version = %v, want %q", r.ServerInfo.Version, lsVersion)
	}
	sync, ok := r.Capabilities.TextDocumentSync.(protocol.TextDocumentSyncKind)
	if !ok || sync != protocol.TextDocumentSyncKindFull {
		t.Errorf("TextDocumentSync = %v, want Full", r.Capabilities.TextDocumentSync)
	}
	if dsp, ok := r.Capabilities.DocumentSymbolProvider.(bool); !ok || !dsp {
		t.Errorf("DocumentSymbolProvider = %v, want true", r.Capabilities.DocumentSymbolProvider)
	}
	stp, ok := r.Capabilities.SemanticTokensProvider.(*protocol.SemanticTokensOptions)
	if !ok || stp == nil {
		t.Fatalf("SemanticTokensProvider = %T, want *SemanticTokensOptions", r.Capabilities.SemanticTokensProvider)
	}
	if full, ok := stp.Full.(bool); !ok || !full {
		t.Errorf("SemanticTokensProvider.Full = %v, want true", stp.Full)
	}
	if len(stp.Legend.TokenTypes) == 0 || len(stp.Legend.TokenModifiers) == 0 {
		t.Errorf("Legend incomplete: types=%v mods=%v", stp.Legend.TokenTypes, stp.Legend.TokenModifiers)
	}
}

func TestLifecycleNoOps(t *testing.T) {
	if err := onInitialized(&glsp.Context{}, &protocol.InitializedParams{}); err != nil {
		t.Errorf("onInitialized err = %v", err)
	}
	if err := onShutdown(&glsp.Context{}); err != nil {
		t.Errorf("onShutdown err = %v", err)
	}
	if err := onExit(&glsp.Context{}); err != nil {
		t.Errorf("onExit err = %v", err)
	}
}

func TestOnDidOpenStoresDocAndPublishes(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	cb := onDidOpen(store, h)

	var notes []capturedNote
	ctx := captureCtx(&notes)
	const uri = "file:///tmp/o.pylon"
	err := cb(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "pylon",
			Version:    1,
			Text:       "[ a ] -> &missing",
		},
	})
	if err != nil {
		t.Fatalf("onDidOpen err = %v", err)
	}
	if _, ok := store.Get(uri); !ok {
		t.Fatalf("store missing doc after didOpen")
	}
	if len(notes) != 1 {
		t.Fatalf("notify calls = %d, want 1", len(notes))
	}
	if notes[0].Method != protocol.ServerTextDocumentPublishDiagnostics {
		t.Errorf("notify method = %q, want publishDiagnostics", notes[0].Method)
	}
	pub, ok := notes[0].Params.(protocol.PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("notify params type = %T, want PublishDiagnosticsParams", notes[0].Params)
	}
	if pub.URI != uri {
		t.Errorf("publish URI = %q, want %q", pub.URI, uri)
	}
	if len(pub.Diagnostics) != 1 {
		t.Errorf("publish diagnostics = %d, want 1 (undefined ref)", len(pub.Diagnostics))
	}
}

func TestOnDidChangeFullSyncReplacement(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/c.pylon"
	store.Open(uri, 1, "[ x ]")

	cb := onDidChange(store, h)
	var notes []capturedNote
	ctx := captureCtx(&notes)

	// Whole-document content change (matches advertised Full sync).
	err := cb(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEventWhole{Text: "[ a ] -> &nope"},
		},
	})
	if err != nil {
		t.Fatalf("onDidChange err = %v", err)
	}
	doc, _ := store.Get(uri)
	if doc.Version != 2 {
		t.Errorf("doc.Version = %d, want 2", doc.Version)
	}
	if doc.Text != "[ a ] -> &nope" {
		t.Errorf("doc.Text = %q", doc.Text)
	}
	if len(notes) != 1 {
		t.Fatalf("notify calls = %d, want 1", len(notes))
	}
	pub := notes[0].Params.(protocol.PublishDiagnosticsParams)
	if len(pub.Diagnostics) != 1 {
		t.Errorf("expected 1 undefined-ref diag, got %d", len(pub.Diagnostics))
	}
}

func TestOnDidChangeIncrementalEvent(t *testing.T) {
	// glsp tolerates incremental events even though we advertise Full —
	// the type switch's TextDocumentContentChangeEvent arm runs.
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/inc.pylon"
	store.Open(uri, 1, "[ a ]")

	cb := onDidChange(store, h)
	var notes []capturedNote
	ctx := captureCtx(&notes)

	err := cb(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                3,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{Text: "[ b ]"},
		},
	})
	if err != nil {
		t.Fatalf("onDidChange err = %v", err)
	}
	doc, _ := store.Get(uri)
	if doc.Text != "[ b ]" {
		t.Errorf("doc.Text = %q, want '[ b ]'", doc.Text)
	}
	if len(notes) != 1 {
		t.Errorf("notify calls = %d, want 1", len(notes))
	}
}

func TestOnDidChangeEmptyChangeListNoOps(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/e.pylon"
	store.Open(uri, 1, "[ a ]")

	cb := onDidChange(store, h)
	var notes []capturedNote
	ctx := captureCtx(&notes)

	err := cb(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{},
	})
	if err != nil {
		t.Fatalf("onDidChange err = %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("empty change list should not publish; got %d notes", len(notes))
	}
	doc, _ := store.Get(uri)
	if doc.Version != 1 {
		t.Errorf("Version unchanged = %d, want 1", doc.Version)
	}
}

// unknownChange is an arbitrary type implementing nothing — exercises
// the default arm of onDidChange's content-change type switch.
type unknownChange struct{}

func TestOnDidChangeUnknownPayloadDefault(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/u.pylon"
	store.Open(uri, 1, "[ a ]")

	cb := onDidChange(store, h)
	var notes []capturedNote
	ctx := captureCtx(&notes)

	err := cb(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{unknownChange{}},
	})
	if err != nil {
		t.Fatalf("onDidChange err = %v", err)
	}
	// Default arm returns nil without a publish — store unchanged.
	if len(notes) != 0 {
		t.Errorf("unknown payload should not publish; got %d notes", len(notes))
	}
	doc, _ := store.Get(uri)
	if doc.Version != 1 {
		t.Errorf("Version = %d, want unchanged 1", doc.Version)
	}
}

func TestOnDidCloseEvictsAndClearsDiagnostics(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/z.pylon"
	store.Open(uri, 1, "[ a ] -> &gone")

	cb := onDidClose(store, h)
	var notes []capturedNote
	ctx := captureCtx(&notes)

	err := cb(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("onDidClose err = %v", err)
	}
	if _, ok := store.Get(uri); ok {
		t.Errorf("doc still in store after close")
	}
	if len(notes) != 1 {
		t.Fatalf("notify calls = %d, want 1 (clearing)", len(notes))
	}
	pub := notes[0].Params.(protocol.PublishDiagnosticsParams)
	if pub.URI != uri {
		t.Errorf("publish URI = %q, want %q", pub.URI, uri)
	}
	if len(pub.Diagnostics) != 0 {
		t.Errorf("publish should clear diagnostics; got %d", len(pub.Diagnostics))
	}
	if pub.Diagnostics == nil {
		t.Error("publish.Diagnostics = nil; LSP requires an empty slice (not nil) to clear markers")
	}
}

func TestOnDocumentSymbolKnownAndUnknown(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const known = "file:///tmp/k.pylon"
	store.Open(known, 1, "[ x :: tagged ]")

	cb := onDocumentSymbol(h)

	res, err := cb(&glsp.Context{}, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: known},
	})
	if err != nil {
		t.Fatalf("onDocumentSymbol(known) err = %v", err)
	}
	syms, ok := res.([]protocol.DocumentSymbol)
	if !ok || len(syms) != 1 {
		t.Fatalf("syms = %#v, want 1 entry", res)
	}
	if syms[0].Name != "tagged" {
		t.Errorf("syms[0].Name = %q, want 'tagged'", syms[0].Name)
	}

	// Unknown URI: handler receives nil from h.DocumentSymbols and
	// returns (nil, nil). Exercises the early-return branch.
	res, err = cb(&glsp.Context{}, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nope"},
	})
	if err != nil {
		t.Fatalf("onDocumentSymbol(unknown) err = %v", err)
	}
	if res != nil {
		t.Errorf("onDocumentSymbol(unknown) = %v, want nil", res)
	}
}

func TestOnDocumentSymbolKnownEmpty(t *testing.T) {
	// Known doc with zero symbols: handler returns the empty slice
	// (not nil) so editors distinguish "no outline" from "unknown URI".
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/empty.pylon"
	store.Open(uri, 1, "[ plain ]")

	cb := onDocumentSymbol(h)
	res, err := cb(&glsp.Context{}, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	syms, ok := res.([]protocol.DocumentSymbol)
	if !ok {
		t.Fatalf("res type = %T, want []DocumentSymbol", res)
	}
	if len(syms) != 0 {
		t.Errorf("syms = %+v, want empty", syms)
	}
}

func TestOnSemanticTokensFull(t *testing.T) {
	store := NewStore()
	h := NewHandlers(store)
	const uri = "file:///tmp/s.pylon"
	store.Open(uri, 1, "[ hi ]")

	cb := onSemanticTokensFull(h)
	st, err := cb(&glsp.Context{}, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if st == nil {
		t.Fatal("SemanticTokens = nil")
	}
	// Two brackets × 5 uint32 each.
	if len(st.Data) != 10 {
		t.Errorf("Data len = %d, want 10", len(st.Data))
	}
}

func TestPublishDiagnosticsUnknownURIClears(t *testing.T) {
	// h.Diagnostics returns []protocol.Diagnostic{} for unknown URIs —
	// publishDiagnostics must still fire a notification carrying an
	// empty (non-nil) slice so the client clears any stale markers.
	store := NewStore()
	h := NewHandlers(store)

	var notes []capturedNote
	ctx := captureCtx(&notes)
	publishDiagnostics(ctx, h, "file:///never-opened")

	if len(notes) != 1 {
		t.Fatalf("notify calls = %d, want 1", len(notes))
	}
	pub := notes[0].Params.(protocol.PublishDiagnosticsParams)
	if pub.Diagnostics == nil {
		t.Error("Diagnostics = nil, want empty slice")
	}
	if len(pub.Diagnostics) != 0 {
		t.Errorf("len(Diagnostics) = %d, want 0", len(pub.Diagnostics))
	}
}

func TestBuildProtocolHandlerWiresAllSlots(t *testing.T) {
	// Every slot the server advertises must be non-nil; a dropped
	// handler shows up as a "method not found" error to the client.
	store := NewStore()
	h := NewHandlers(store)
	ph := buildProtocolHandler(store, h)

	if ph.Initialize == nil {
		t.Error("Initialize unwired")
	}
	if ph.Initialized == nil {
		t.Error("Initialized unwired")
	}
	if ph.Shutdown == nil {
		t.Error("Shutdown unwired")
	}
	if ph.Exit == nil {
		t.Error("Exit unwired")
	}
	if ph.TextDocumentDidOpen == nil {
		t.Error("TextDocumentDidOpen unwired")
	}
	if ph.TextDocumentDidChange == nil {
		t.Error("TextDocumentDidChange unwired")
	}
	if ph.TextDocumentDidClose == nil {
		t.Error("TextDocumentDidClose unwired")
	}
	if ph.TextDocumentDocumentSymbol == nil {
		t.Error("TextDocumentDocumentSymbol unwired")
	}
	if ph.TextDocumentSemanticTokensFull == nil {
		t.Error("TextDocumentSemanticTokensFull unwired")
	}
}
