package lsp

import (
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// diagnosticSource is carried on every published diagnostic so
// clients can distinguish pylon-lsp diagnostics from other language
// servers sharing the same buffer.
const diagnosticSource = "pylon-lsp"

// spanToRange translates a pylon.Span into an LSP protocol.Range.
// Both sides are 0-based (line + UTF-16 column) — the conversion is
// a pure field rename.
func spanToRange(s pylon.Span) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(s.Start.Line),
			Character: uint32(s.Start.Column),
		},
		End: protocol.Position{
			Line:      uint32(s.End.Line),
			Character: uint32(s.End.Column),
		},
	}
}

// severityFor maps pylon.Severity to the LSP severity enum.
// SeverityError becomes Error; SeverityWarning becomes Warning;
// anything else falls back to Information so clients always have a
// visible severity rather than dropping the diagnostic.
func severityFor(sev pylon.Severity) protocol.DiagnosticSeverity {
	switch sev {
	case pylon.SeverityError:
		return protocol.DiagnosticSeverityError
	case pylon.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	default:
		return protocol.DiagnosticSeverityInformation
	}
}

// toProtocolDiagnostic converts one pylon.Diagnostic into the LSP
// wire shape. Code is emitted as the string form ("ref.undefined",
// etc.) — stable across releases and matchable by editor configs.
func toProtocolDiagnostic(d pylon.Diagnostic) protocol.Diagnostic {
	sev := severityFor(d.Severity)
	source := diagnosticSource
	return protocol.Diagnostic{
		Range:    spanToRange(d.Span),
		Severity: &sev,
		Code:     &protocol.IntegerOrString{Value: string(d.Code)},
		Source:   &source,
		Message:  d.Message,
	}
}
