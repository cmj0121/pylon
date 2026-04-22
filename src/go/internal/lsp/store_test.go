package lsp

import (
	"testing"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

func TestStoreOpenChangeClose(t *testing.T) {
	s := NewStore()
	const uri = "file:///tmp/a.pylon"

	if _, ok := s.Get(uri); ok {
		t.Fatal("empty store returned ok=true")
	}

	s.Open(uri, 1, "[ hello ]")
	doc, ok := s.Get(uri)
	if !ok {
		t.Fatal("Get after Open returned ok=false")
	}
	if doc.Version != 1 || doc.Text != "[ hello ]" {
		t.Errorf("doc=%+v, want {Version:1 Text:\"[ hello ]\"}", doc)
	}

	s.Change(uri, 2, "[ world ]")
	doc, ok = s.Get(uri)
	if !ok {
		t.Fatal("Get after Change returned ok=false")
	}
	if doc.Version != 2 || doc.Text != "[ world ]" {
		t.Errorf("doc=%+v after Change, want {Version:2 Text:\"[ world ]\"}", doc)
	}

	s.Close(uri)
	if _, ok := s.Get(uri); ok {
		t.Error("Get after Close returned ok=true")
	}
}

func TestStoreChangeImpliesOpen(t *testing.T) {
	// A malformed client sending didChange before didOpen must not
	// crash the server; Change falls through to Open semantics.
	s := NewStore()
	const uri = "file:///tmp/b.pylon"
	s.Change(uri, 5, "implicit open")
	doc, ok := s.Get(uri)
	if !ok || doc.Version != 5 {
		t.Errorf("implicit-open Get=%+v ok=%v, want Version:5", doc, ok)
	}
}

func TestStoreOpenComputesDiagnostics(t *testing.T) {
	// Open caches AST + Diagnostics so handler reads don't re-parse.
	s := NewStore()
	const uri = "file:///tmp/bad.pylon"
	s.Open(uri, 1, "[ a ] -> &missing")
	doc, ok := s.Get(uri)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if doc.AST == nil {
		t.Fatal("AST is nil")
	}
	if len(doc.Diagnostics) != 1 {
		t.Fatalf("len(Diagnostics) = %d, want 1 (%+v)", len(doc.Diagnostics), doc.Diagnostics)
	}
	if doc.Diagnostics[0].Code != pylon.CodeUndefinedRef {
		t.Errorf("Diagnostics[0].Code = %q, want %q", doc.Diagnostics[0].Code, pylon.CodeUndefinedRef)
	}
}

func TestStoreChangeRecomputesDiagnostics(t *testing.T) {
	s := NewStore()
	const uri = "file:///tmp/e.pylon"
	s.Open(uri, 1, "[ hello ]") // clean
	doc, _ := s.Get(uri)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("clean source produced diagnostics: %+v", doc.Diagnostics)
	}
	// Change to invalid source — diagnostics should now be non-empty.
	s.Change(uri, 2, "[ foo | unknown ]")
	doc, _ = s.Get(uri)
	if len(doc.Diagnostics) != 1 {
		t.Fatalf("Change did not refresh diagnostics: %+v", doc.Diagnostics)
	}
	if doc.Diagnostics[0].Code != pylon.CodeUnknownRenderer {
		t.Errorf("Code = %q, want %q", doc.Diagnostics[0].Code, pylon.CodeUnknownRenderer)
	}
}
