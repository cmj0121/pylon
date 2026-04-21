package lsp

import "testing"

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
