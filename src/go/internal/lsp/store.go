// Package lsp implements the pylon-lsp Language Server.
//
// This is an internal/ package — Go's import rules keep it unreachable
// from outside the src/go module. Public surface: Run (server.go),
// wired from src/go/cmd/pylon-lsp/main.go.
package lsp

import "sync"

// Document is a single open source document cached by the server.
// Version tracks the LSP text-document version counter; Text is the
// latest full-sync content.
//
// AST and Diagnostics slots are added when U5 wires Validate() in.
type Document struct {
	URI     string
	Version int32
	Text    string
}

// Store is the URI-keyed document cache. Handlers read under RLock;
// Open / Change / Close mutate under Lock.
type Store struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewStore returns an empty Store ready for didOpen notifications.
func NewStore() *Store {
	return &Store{docs: map[string]*Document{}}
}

// Open records a newly-opened document. A didOpen for a URI already in
// the store is treated as a reset — the new contents replace the old.
func (s *Store) Open(uri string, version int32, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = &Document{URI: uri, Version: version, Text: text}
}

// Change replaces the full text of an open document. v1 uses full-text
// sync (no incremental patches), so text is always the complete buffer.
// A Change on a URI that was never Open'd is treated as an implicit
// Open — clients that send didChange before didOpen are malformed but
// we shouldn't crash on them.
func (s *Store) Change(uri string, version int32, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = &Document{URI: uri, Version: version, Text: text}
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
