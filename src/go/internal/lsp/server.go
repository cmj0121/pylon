package lsp

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// lsName is the reported server name; clients show this in tracing
// output. Keep stable — editor configs may match on it.
const lsName = "pylon-lsp"

// lsVersion is advertised via InitializeResult.ServerInfo so clients
// know which Pylon protocol they're talking to.
const lsVersion = "0.1.0"

// Run starts the LSP server on stdio. It blocks until the client
// closes the connection, then returns. Logging is routed to stderr
// exclusively; stdout is reserved for LSP wire bytes.
//
// Run does not return until the server stops, so main.go's
// error-path is exercised only on startup failures.
func Run() error {
	// Pylon's own zerolog → stderr. glsp's commonlog has no backend
	// registered in this build, so its internal log calls are dropped
	// (no risk of stdout corruption). This function must be called
	// before any glsp entry point.
	configureLogging()

	store := NewStore()
	handlers := NewHandlers(store)
	handler := buildProtocolHandler(store, handlers)
	srv := server.NewServer(&handler, lsName, false)

	// glsp recovers from handler panics internally (errors are logged
	// and the server continues). We wrap RunStdio in a defer so a
	// deeper panic surfaces as a stderr log + non-zero exit rather
	// than a silent crash that confuses the client.
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("pylon-lsp: recovered from panic")
		}
	}()

	return srv.RunStdio()
}

// configureLogging pins zerolog's output writer to os.Stderr. stdout
// is the LSP transport and any byte written there corrupts the wire
// format. Uses ConsoleWriter so logs are readable when the server is
// launched by hand; LSP clients usually pipe stderr to a log file.
func configureLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
}

// buildProtocolHandler wires glsp's protocol.Handler slots to the
// server's lifecycle + document-sync callbacks. U4 advertises only
// textDocument/didOpen/didChange/didClose (full-text sync) — the
// feature-flag slots for documentSymbol, semanticTokens, etc. stay
// nil and so capabilities advertise false for them. U5/U6/U7 fill
// them in.
func buildProtocolHandler(store *Store, handlers *Handlers) protocol.Handler {
	return protocol.Handler{
		Initialize:  onInitialize,
		Initialized: onInitialized,
		Shutdown:    onShutdown,
		Exit:        onExit,

		TextDocumentDidOpen:   onDidOpen(store, handlers),
		TextDocumentDidChange: onDidChange(store, handlers),
		TextDocumentDidClose:  onDidClose(store, handlers),

		TextDocumentDocumentSymbol: onDocumentSymbol(handlers),
	}
}

func onInitialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	// Full-text sync plus the features U5+ flipped on: document
	// symbols. diagnosticProvider stays nil -- we push diagnostics
	// via the publish notification, not pull (3.17-era) queries.
	caps := protocol.ServerCapabilities{
		TextDocumentSync:       protocol.TextDocumentSyncKindFull,
		DocumentSymbolProvider: true,
	}
	version := lsVersion
	return protocol.InitializeResult{
		Capabilities: caps,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func onInitialized(ctx *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func onShutdown(ctx *glsp.Context) error {
	return nil
}

func onExit(ctx *glsp.Context) error {
	return nil
}

// onDidOpen returns a glsp-shaped callback that records the opened
// document in the store and publishes its diagnostics to the client.
// Returning a closure keeps *Handlers reachable for the publish step.
func onDidOpen(store *Store, h *Handlers) protocol.TextDocumentDidOpenFunc {
	return func(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		uri := string(params.TextDocument.URI)
		store.Open(uri, params.TextDocument.Version, params.TextDocument.Text)
		publishDiagnostics(ctx, h, uri)
		return nil
	}
}

// onDidChange refreshes the store from a full-sync replacement and
// republishes diagnostics. glsp's ContentChange is a union type
// (incremental vs whole) — we advertise Full in capabilities so the
// Whole arm is expected, but the type switch tolerates either shape
// for malformed clients.
func onDidChange(store *Store, h *Handlers) protocol.TextDocumentDidChangeFunc {
	return func(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		if len(params.ContentChanges) == 0 {
			return nil
		}
		// Full-sync contract: the last ContentChange carries the full
		// buffer.
		last := params.ContentChanges[len(params.ContentChanges)-1]
		var text string
		switch c := last.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			text = c.Text
		case protocol.TextDocumentContentChangeEvent:
			text = c.Text
		default:
			return nil
		}
		uri := string(params.TextDocument.URI)
		store.Change(uri, params.TextDocument.Version, text)
		publishDiagnostics(ctx, h, uri)
		return nil
	}
}

// onDidClose evicts the document and clears any diagnostics the
// client still thinks are live — LSP convention is that the server
// owns the set until it tells the client otherwise.
func onDidClose(store *Store, h *Handlers) protocol.TextDocumentDidCloseFunc {
	return func(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		uri := string(params.TextDocument.URI)
		store.Close(uri)
		// Publish an empty diagnostics array so the client drops any
		// markers it still has for this URI. Done AFTER the evict so
		// Handlers.Diagnostics observes the empty cache and returns
		// nil — we then send an explicit empty slice below to satisfy
		// LSP's "server is responsible for clearing state" contract.
		ctx.Notify(
			protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: []protocol.Diagnostic{},
			},
		)
		return nil
	}
}

// onDocumentSymbol answers textDocument/documentSymbol requests with
// the hierarchical DocumentSymbol form. glsp's callback returns any
// so clients that prefer SymbolInformation[] could be accommodated
// later; we always serve the richer DocumentSymbol tree.
func onDocumentSymbol(h *Handlers) protocol.TextDocumentDocumentSymbolFunc {
	return func(ctx *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
		uri := string(params.TextDocument.URI)
		syms := h.DocumentSymbols(uri)
		if syms == nil {
			return nil, nil
		}
		return syms, nil
	}
}

// publishDiagnostics computes the current diagnostic set for uri and
// pushes it to the client as a textDocument/publishDiagnostics
// notification. Called from didOpen and didChange closures after the
// store update.
func publishDiagnostics(ctx *glsp.Context, h *Handlers, uri string) {
	diags := h.Diagnostics(uri)
	if diags == nil {
		diags = []protocol.Diagnostic{}
	}
	ctx.Notify(
		protocol.ServerTextDocumentPublishDiagnostics,
		protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diags,
		},
	)
}
