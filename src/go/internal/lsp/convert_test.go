package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

func TestSpanToRange(t *testing.T) {
	span := pylon.Span{
		Start: pylon.Position{Offset: 0, Line: 2, Column: 5},
		End:   pylon.Position{Offset: 12, Line: 3, Column: 8},
	}
	got := spanToRange(span)
	want := protocol.Range{
		Start: protocol.Position{Line: 2, Character: 5},
		End:   protocol.Position{Line: 3, Character: 8},
	}
	if got != want {
		t.Errorf("spanToRange = %+v, want %+v", got, want)
	}
}

func TestSeverityFor(t *testing.T) {
	cases := []struct {
		in   pylon.Severity
		want protocol.DiagnosticSeverity
	}{
		{pylon.SeverityError, protocol.DiagnosticSeverityError},
		{pylon.SeverityWarning, protocol.DiagnosticSeverityWarning},
		{pylon.Severity(99), protocol.DiagnosticSeverityInformation}, // fallback
	}
	for _, c := range cases {
		if got := severityFor(c.in); got != c.want {
			t.Errorf("severityFor(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestToProtocolDiagnostic(t *testing.T) {
	d := pylon.Diagnostic{
		Code:     pylon.CodeUndefinedRef,
		Severity: pylon.SeverityError,
		Message:  "Undefined ref: &missing",
		Span: pylon.Span{
			Start: pylon.Position{Offset: 6, Line: 0, Column: 6},
			End:   pylon.Position{Offset: 14, Line: 0, Column: 14},
		},
	}
	got := toProtocolDiagnostic(d)
	if got.Range.Start.Line != 0 || got.Range.Start.Character != 6 {
		t.Errorf("Range.Start = %+v", got.Range.Start)
	}
	if got.Range.End.Line != 0 || got.Range.End.Character != 14 {
		t.Errorf("Range.End = %+v", got.Range.End)
	}
	if got.Severity == nil || *got.Severity != protocol.DiagnosticSeverityError {
		t.Errorf("Severity = %v", got.Severity)
	}
	if got.Code == nil {
		t.Fatal("Code is nil")
	}
	if got.Code.Value != string(pylon.CodeUndefinedRef) {
		t.Errorf("Code.Value = %v, want %q", got.Code.Value, pylon.CodeUndefinedRef)
	}
	if got.Source == nil || *got.Source != diagnosticSource {
		t.Errorf("Source = %v, want %q", got.Source, diagnosticSource)
	}
	if got.Message != "Undefined ref: &missing" {
		t.Errorf("Message = %q", got.Message)
	}
}
