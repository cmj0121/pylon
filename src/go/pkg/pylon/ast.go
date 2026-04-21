// Package pylon implements the Pylon diagram grammar and renderers.
//
// The JS reference implementation lives at src/js/pylon.js; this
// package is a Go-idiomatic rewrite that is expected to produce
// byte-identical ASCII output for the subset currently covered.
package pylon

// Node is the sealed-interface root of the AST. Every concrete node
// type carries a marker isNode method so the set stays closed.
type Node interface {
	isNode()
}

// Alignment values used by Box.Align.
const (
	AlignLeft   = "left"
	AlignCenter = "center"
	AlignRight  = "right"
)

// Edge direction values used by Edge.Direction.
const (
	DirRight = "right"
	DirLeft  = "left"
	DirBoth  = "both"
)

// Box is a bracketed or parenthesised container. Bordered=true means
// the source was `[...]`; false means `(...)`. Name is populated when
// the body carried a `:: ident` declaration. Renderer is populated
// when the body carried a trailing `| renderer` marker (first wins).
type Box struct {
	Bordered bool
	Align    string
	Name     string
	Renderer string
	Items    []Node
	// Meta is populated on the root box only; for nested boxes it is
	// a zero-value Meta.
	Meta Meta
	// Errors carries parse-shape errors (toast-path) surfaced by the
	// root node. Never populated for nested boxes.
	Errors []string
	// Span covers the full bracketed (or parenthesised) range, from the
	// opening `[` / `(` through the closing `]` / `)`. The synthetic
	// root box that wraps a bare text source has Span covering the
	// whole input. Zero Span means span-free (the synthetic root for
	// the empty-source fallback before re-parse).
	Span Span
}

func (*Box) isNode() {}

// Row is the horizontal-chain wrapper: a line that contains at least
// one edge. Rows only appear as children of a Box.
type Row struct {
	Items []Node

	// Internal helpers populated by ref-resolution. Not part of the
	// public AST contract and ignored by downstream renderers.
	linearPartsCache interface{}
}

func (*Row) isNode() {}

// Edge is a flow token between sibling nodes on the same line. When
// Label is non-nil the edge came from a labelled-edge pattern
// `[A] -- ( label ) --> [B]`; LeftLen and RightLen hold the dash
// counts on each side for potential re-rendering.
type Edge struct {
	Direction string
	Length    int
	Label     *Box
	LeftLen   int
	RightLen  int

	// Internal flags used during same-row / cross-row ref resolution.
	consumedBySameRowArc bool
	consumedByCrossRow   bool
}

func (*Edge) isNode() {}

// Ref is a `&ident` named-node reference. After resolution it either
// becomes inline Text (via the unresolved / duplicate-claim path) or
// it remains a Ref flagged as SameRowArc / CrossRow — those render
// as arcs rather than inline text.
type Ref struct {
	Name string

	// Populated by resolveRefs.
	SameRowArc bool
	CrossRow   bool
	Target     *Box
	SourceBox  *Box

	// Span covers the literal `&ident` token (sigil + identifier).
	Span Span
}

func (*Ref) isNode() {}

// DataRef is an `@ident` data reference inside a box body.
type DataRef struct {
	Name string
	// Span covers the literal `@ident` token (sigil + identifier).
	Span Span
}

func (*DataRef) isNode() {}

// Text is a literal text run.
type Text struct {
	Content string
}

func (*Text) isNode() {}

// Size is the frontmatter `size: WxH` declaration.
type Size struct {
	W int
	H int
}

// Meta carries frontmatter data attached to the root box.
//
// Data mirrors the JS shape: either a []map[string]any (flat list)
// or a map[string]any keyed by series name (whose values are lists),
// or nil when no `data:` block appeared. Errors surfaces parse-shape
// errors to the toast path.
type Meta struct {
	Size   *Size
	Theme  string
	Data   interface{}
	Errors []string
}

// AST is a legacy alias for *Box — the JS parser exposes `parse()` as
// returning the root box directly, and we keep the same convention.
// Kept as a separate named type so future separation of the top-level
// from the root box is cheap.
type AST = *Box
