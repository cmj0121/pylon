package pylon

// Severity tags a Diagnostic with its LSP-facing severity.
// SeverityError corresponds to JS's toast-path (whole-source invalid);
// SeverityWarning to the inline `⚠` render-time surface.
type Severity int

const (
	SeverityError   Severity = 1
	SeverityWarning Severity = 2
)

// Code is a stable identifier for a diagnostic class. Editors, config
// files, and the CI parity gate match against these string values, so
// the Code enum is an ADD-ONLY API — renaming or removing a Code is a
// breaking change. New diagnostic classes get a new Code.
type Code string

const (
	CodeUnsupportedData Code = "data.unsupported"       // frontmatter `data:` rejected (tab indent, flow style, etc.)
	CodeDuplicateName   Code = "ref.duplicate"          // `:: name` collided with an earlier declaration
	CodeUndefinedRef    Code = "ref.undefined"          // `&name` target never declared
	CodeUnknownRenderer Code = "renderer.unknown"       // `| renderer` names an unknown renderer
	CodeUseAtRef        Code = "renderer.use_at_ref"    // non-text renderer handed a raw string
	CodeBareDataRef     Code = "renderer.bare_data_ref" // `@ref` in a box body with no `| renderer`
	CodeDataNotFound    Code = "data.not_found"         // `@ref` doesn't resolve against frontmatter `data:`
	CodeBarShape        Code = "bar.shape"              // bar-family series doesn't match `[{x,y}]`
	CodeBarEmpty        Code = "bar.empty"              // bar-family series has zero entries
	CodeBarNegativeY    Code = "bar.negative_y"         // bar-family entry has y < 0
	CodeBarDuplicateX   Code = "bar.duplicate_x"        // bar-family series has duplicate x keys
)

// Diagnostic is one complaint about the source. Message text matches
// the JS reference implementation byte-for-byte — the CI parity gate
// compares messages directly, so wording changes cross-channel have
// to land in both implementations at once.
//
// Span is narrow: the offending token, name, or frontmatter key.
// BoxSpan is broader context: for render-time (SeverityWarning)
// diagnostics, it's the enclosing box the `⚠` message would replace;
// for toast-path (SeverityError) diagnostics it equals Span.
type Diagnostic struct {
	Code     Code
	Severity Severity
	Message  string
	Span     Span
	BoxSpan  Span
}
