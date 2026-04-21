// Package lsp implements the pylon-lsp Language Server.
//
// This is an internal/ package — Go's import rules keep it unreachable
// from outside the src/go module. Public surface: Run (server.go),
// wired from src/go/cmd/pylon-lsp/main.go.
package lsp

import (
	"sync"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// Document is a single open source document cached by the server.
// Version tracks the LSP text-document version counter; Text is the
// latest full-sync content.
//
// AST and Diagnostics are computed once (at Open / Change time) and
// cached so feature handlers stay transport-free: they translate from
// this already-parsed state rather than re-running the pipeline.
type Document struct {
	URI         string
	Version     int32
	Text        string
	AST         pylon.AST
	Diagnostics []pylon.Diagnostic
}

// Store is the URI-keyed document cache. Handlers read under RLock;
// Open / Change / Close mutate under Lock. Parse + Validate run inside
// the mutation's critical section — cheap on pylon-scale files
// (microseconds) and keeps the Document snapshot self-consistent for
// every reader.
type Store struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewStore returns an empty Store ready for didOpen notifications.
func NewStore() *Store {
	return &Store{docs: map[string]*Document{}}
}

// Open records a newly-opened document, parsing and validating under
// Lock so Document.AST and Document.Diagnostics are ready for the
// first handler read.
func (s *Store) Open(uri string, version int32, text string) {
	doc := buildDocument(uri, version, text)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = doc
}

// Change replaces the full text of an open document. v1 uses full-
// text sync (no incremental patches), so text is always the complete
// buffer. A Change on a URI that was never Open'd is treated as an
// implicit Open — malformed clients shouldn't crash the server.
func (s *Store) Change(uri string, version int32, text string) {
	doc := buildDocument(uri, version, text)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = doc
}

// Close evicts a document. Subsequent Get returns (nil, false).
func (s *Store) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

// Get returns a pointer to the cached document for uri, or (nil, false)
// if none is open. The returned pointer aliases the live entry;
// handlers must treat it as read-only.
func (s *Store) Get(uri string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[uri]
	return doc, ok
}

// buildDocument parses and validates the source, returning a Document
// with AST + Diagnostics populated. Runs outside the Store mutex so
// the critical section stays short.
func buildDocument(uri string, version int32, text string) *Document {
	ast := pylon.Parse(text)
	diags := pylon.Validate(ast)
	return &Document{
		URI:         uri,
		Version:     version,
		Text:        text,
		AST:         ast,
		Diagnostics: diags,
	}
}
