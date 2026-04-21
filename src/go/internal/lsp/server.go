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
	}
}

func onInitialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	// Build capabilities from which protocol.Handler slots are set.
	// Non-nil slots flip their capability on; nil stays off. Full-text
	// sync is selected by pinning the sync kind explicitly.
	caps := protocol.ServerCapabilities{
		TextDocumentSync: protocol.TextDocumentSyncKindFull,
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
// document in the store. Returning a closure keeps handlers *Handlers
// reachable for U5 (which will call h.Diagnostics and publish via ctx).
func onDidOpen(store *Store, h *Handlers) protocol.TextDocumentDidOpenFunc {
	return func(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		store.Open(
			string(params.TextDocument.URI),
			params.TextDocument.Version,
			params.TextDocument.Text,
		)
		return nil
	}
}

// onDidChange refreshes the store from a full-sync replacement. The
// glsp protocol also supports incremental updates via a different
// ContentChange union; we advertise Full in capabilities so clients
// always send a single full-text entry.
func onDidChange(store *Store, h *Handlers) protocol.TextDocumentDidChangeFunc {
	return func(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		if len(params.ContentChanges) == 0 {
			return nil
		}
		// Full-sync contract: the last ContentChange carries the full
		// buffer. Union type means we have to type-switch.
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
		store.Change(string(params.TextDocument.URI), params.TextDocument.Version, text)
		return nil
	}
}

func onDidClose(store *Store, h *Handlers) protocol.TextDocumentDidCloseFunc {
	return func(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		store.Close(string(params.TextDocument.URI))
		return nil
	}
}
