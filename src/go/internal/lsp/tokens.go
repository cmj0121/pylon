// Semantic tokens for pylon sources — a partial MVP.
//
// This commit (U7) emits tokens for three classes the U1a/U1b AST
// already span-tracks:
//
//   - `&ref`  → variable
//   - `@ref`  → variable + readonly modifier
//   - `[`, `]`, `(`, `)` → operator (single-char per bracket)
//
// DEFERRED to feat/pylon-lsp-ux (require source-scan logic or new
// AST spans the engine does not track today):
//
//   - frontmatter key / number / string tokens
//   - edge arrows `->`, `<-`, `<->`, `-->`, etc.
//   - alignment marker dashes flush with brackets
//   - `| renderer` pipe tags
//   - `:: name` declaration identifiers
//   - frontmatter fence `---` lines
//
// The legend reserves space for those classes via the documented
// growth path below — future commits append entries, never reorder.
//
// Wire format: the server advertises (tokenTypes, tokenModifiers) in
// the initialize legend; each subsequent textDocument/semanticTokens
// response is a flat []uint32 whose groups of 5 encode one token as
// (deltaLine, deltaStartChar, length, tokenType, tokenModifiers).
// LSP requires source-order emission and disallows inter-token
// overlap on the same line — encodeTokens asserts both.
package lsp

import (
	"sort"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// tokenTypes is the server-authoritative semantic-token type legend.
// ADD-ONLY: reorders / removals break editor clients that cached the
// legend from a previous initialize response. Future commits append.
//
// Growth path (entries to add when U7's deferred classes land):
//
//	"property"   for pylonFMKey
//	"string"     for pylonFMString
//	"number"     for pylonFMNumber
//	"decorator"  for pylonRendererPipe
//	"keyword"    for pylonFrontmatter fence lines
var tokenTypes = []string{
	"variable", // 0: &ref (pylonNodeRef) and @ref (pylonDataRef)
	"operator", // 1: brackets [ ] ( ) (pylonBracket)
}

// tokenModifiers is the server-authoritative modifier bitmask legend.
// Modifiers are OR-ed together; bit N corresponds to entry N.
//
// Growth path:
//
//	"declaration" for :: name when NameSpan lands
var tokenModifiers = []string{
	"readonly", // bit 0: set on @ref — frontmatter data is read-only
}

// Token-type indices used below; keep in sync with tokenTypes order.
const (
	tokTypeVariable uint32 = 0
	tokTypeOperator uint32 = 1
)

// Modifier bits; keep in sync with tokenModifiers order.
const modReadonly uint32 = 1 << 0

// semanticToken is an intermediate form. LSP wire format is a flat
// delta-encoded []uint32; we collect raw tokens first, sort by
// source order, then encode.
type semanticToken struct {
	Line      uint32
	StartChar uint32
	Length    uint32
	TokenType uint32
	Modifiers uint32
}

// collectSemanticTokens walks a parsed AST and returns every token
// this MVP knows how to emit. Order of collection does not matter —
// encodeTokens sorts and validates before emitting.
func collectSemanticTokens(root pylon.AST) []semanticToken {
	if root == nil {
		return nil
	}
	var out []semanticToken
	var visit func(n pylon.Node)
	visit = func(n pylon.Node) {
		if n == nil {
			return
		}
		switch x := n.(type) {
		case *pylon.Box:
			emitBracketTokens(&out, x)
			for _, c := range x.Items {
				visit(c)
			}
		case *pylon.Row:
			for _, c := range x.Items {
				visit(c)
			}
		case *pylon.Edge:
			if x.Label != nil {
				visit(x.Label)
			}
		case *pylon.Ref:
			emitRefToken(&out, x)
		case *pylon.DataRef:
			emitDataRefToken(&out, x)
		}
	}
	visit(root)
	return out
}

// emitBracketTokens emits the opener and closer single-char operator
// tokens for a Box. Skipped when Span is zero-valued (the synthetic
// default-example fallback has no real source span to highlight).
// Brackets always occupy one byte of source, so they never cross
// line boundaries.
func emitBracketTokens(out *[]semanticToken, b *pylon.Box) {
	if b.Span == (pylon.Span{}) {
		return
	}
	// Opener: 1-char at Span.Start.
	*out = append(*out, semanticToken{
		Line:      uint32(b.Span.Start.Line),
		StartChar: uint32(b.Span.Start.Column),
		Length:    1,
		TokenType: tokTypeOperator,
	})
	// Closer: 1-char ending at Span.End (exclusive).
	if b.Span.End.Column == 0 {
		// Defensive: no closer to highlight (empty box would have
		// End.Column >= 1 for a well-formed source, but guard anyway).
		return
	}
	*out = append(*out, semanticToken{
		Line:      uint32(b.Span.End.Line),
		StartChar: uint32(b.Span.End.Column - 1),
		Length:    1,
		TokenType: tokTypeOperator,
	})
}

// emitRefToken emits a variable token covering the literal `&ident`.
// Refs are single-line by construction (grammar admits no newline in
// an identifier) so a single token per Ref is sufficient.
func emitRefToken(out *[]semanticToken, r *pylon.Ref) {
	if r.Span == (pylon.Span{}) || r.Span.Start.Line != r.Span.End.Line {
		return
	}
	*out = append(*out, semanticToken{
		Line:      uint32(r.Span.Start.Line),
		StartChar: uint32(r.Span.Start.Column),
		Length:    uint32(r.Span.End.Column - r.Span.Start.Column),
		TokenType: tokTypeVariable,
	})
}

// emitDataRefToken is the @ref counterpart — same shape as ref but
// carries the readonly modifier because frontmatter data is not
// user-writable from a box body.
func emitDataRefToken(out *[]semanticToken, d *pylon.DataRef) {
	if d.Span == (pylon.Span{}) || d.Span.Start.Line != d.Span.End.Line {
		return
	}
	*out = append(*out, semanticToken{
		Line:      uint32(d.Span.Start.Line),
		StartChar: uint32(d.Span.Start.Column),
		Length:    uint32(d.Span.End.Column - d.Span.Start.Column),
		TokenType: tokTypeVariable,
		Modifiers: modReadonly,
	})
}

// sortAndValidateTokens sorts tokens into source order ((Line,
// StartChar) ascending) and drops any that overlap a preceding
// token on the same line. Overlaps indicate a collection-side bug
// — LSP's behaviour on overlap is undefined, so silently dropping
// is safer than letting clients render garbage.
func sortAndValidateTokens(toks []semanticToken) []semanticToken {
	sort.SliceStable(toks, func(i, j int) bool {
		if toks[i].Line != toks[j].Line {
			return toks[i].Line < toks[j].Line
		}
		return toks[i].StartChar < toks[j].StartChar
	})
	out := toks[:0]
	var prev semanticToken
	var havePrev bool
	for _, t := range toks {
		if havePrev && prev.Line == t.Line && prev.StartChar+prev.Length > t.StartChar {
			// Overlap with previous on same line — skip.
			continue
		}
		out = append(out, t)
		prev = t
		havePrev = true
	}
	return out
}

// encodeTokens produces the LSP wire shape: a []uint32 of 5-tuples
// (deltaLine, deltaStartChar, length, tokenType, modifiers). The
// first token's deltas are measured from origin (0, 0).
func encodeTokens(toks []semanticToken) []uint32 {
	out := make([]uint32, 0, len(toks)*5)
	var prevLine, prevChar uint32
	first := true
	for _, t := range toks {
		var dLine, dChar uint32
		if first {
			dLine = t.Line
			dChar = t.StartChar
			first = false
		} else {
			dLine = t.Line - prevLine
			if dLine == 0 {
				dChar = t.StartChar - prevChar
			} else {
				dChar = t.StartChar
			}
		}
		out = append(out, dLine, dChar, t.Length, t.TokenType, t.Modifiers)
		prevLine = t.Line
		prevChar = t.StartChar
	}
	return out
}
