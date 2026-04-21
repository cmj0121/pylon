package pylon

import (
	"sort"
	"unicode/utf16"
	"unicode/utf8"
)

// Position identifies a single point in the source. Line and Column
// follow the LSP convention: zero-based, with Column counted in UTF-16
// code units (not bytes, not Unicode code points) so LSP clients can
// translate Column directly to an editor cursor offset.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Span identifies a [Start, End) byte range in the source, with Start
// and End pre-resolved to LSP-ready Positions.
type Span struct {
	Start Position
	End   Position
}

// Source is an immutable view into a pylon source document. Parser
// helpers take *Source (rather than string) so every sub-range can
// resolve absolute offsets without threading a per-call base.
//
// Text always references the ORIGINAL source bytes (CR included, no
// normalisation). Base is the absolute byte offset of this view into
// Text; a top-level Source has Base==0. Sub(start, end) returns a
// narrower view sharing Text and linePrefix — linePrefix is built
// once against the original Text and reused by every derived view.
type Source struct {
	Text       string
	Base       int
	linePrefix []int
}

// NewSource builds a top-level Source over s. linePrefix is computed
// eagerly: the cost is O(len(s)) once, and every downstream PositionAt
// is O(log(lines)) rather than O(offset).
func NewSource(s string) *Source {
	return &Source{Text: s, Base: 0, linePrefix: buildLinePrefix(s)}
}

// Sub returns a view into the same Text covering local byte range
// [start, end). Start and end are offsets within the CURRENT view
// (not the original Text). linePrefix is shared — positions returned
// by the sub-view still reference absolute lines in the original Text.
func (s *Source) Sub(start, end int) *Source {
	baseAbs := s.Base + start
	endAbs := s.Base + end
	if baseAbs < 0 {
		baseAbs = 0
	}
	if endAbs > len(s.Text) {
		endAbs = len(s.Text)
	}
	if baseAbs > endAbs {
		baseAbs = endAbs
	}
	return &Source{Text: s.Text[:endAbs], Base: baseAbs, linePrefix: s.linePrefix}
}

// View returns the byte slice this Source covers — Text[Base:end], where
// end is the rightmost reachable offset. Callers should prefer this
// over s.Text[s.Base:] when iterating, since Sub() also narrows End.
func (s *Source) View() string {
	return s.Text[s.Base:]
}

// PositionAt converts a local offset (within this view, 0 at Base)
// to an absolute Position. local is clamped to [0, len(view)].
func (s *Source) PositionAt(local int) Position {
	abs := s.Base + local
	if abs < 0 {
		abs = 0
	}
	if abs > len(s.Text) {
		abs = len(s.Text)
	}
	line := lineFor(s.linePrefix, abs)
	col := utf16ColumnAt(s.Text, s.linePrefix[line], abs)
	return Position{Offset: abs, Line: line, Column: col}
}

// SpanOf builds a Span covering local byte range [start, end). Both
// endpoints are resolved via PositionAt, so column counting is UTF-16
// correct for labels containing non-ASCII or surrogate-pair glyphs.
func (s *Source) SpanOf(start, end int) Span {
	return Span{Start: s.PositionAt(start), End: s.PositionAt(end)}
}

// buildLinePrefix returns the byte offset of the first character of
// each line. linePrefix[0] is always 0; subsequent entries point to
// the byte after each '\n'. CR bytes stay part of the preceding line,
// matching the LSP wire format which uses \n as the line separator.
func buildLinePrefix(s string) []int {
	out := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, i+1)
		}
	}
	return out
}

// lineFor returns the 0-based line index containing byte offset abs.
// Uses binary search over linePrefix (sorted ascending by construction).
func lineFor(linePrefix []int, abs int) int {
	// sort.Search finds the first index where linePrefix[i] > abs.
	// The line containing abs is (that index) - 1.
	idx := sort.Search(len(linePrefix), func(i int) bool {
		return linePrefix[i] > abs
	})
	if idx == 0 {
		return 0
	}
	return idx - 1
}

// utf16ColumnAt returns the UTF-16 column of abs within the line that
// starts at lineStart. Column is the count of UTF-16 code units
// between [lineStart, abs) over the text in s. Bytes are decoded as
// UTF-8; each rune contributes utf16.RuneLen code units (1 for BMP,
// 2 for supplementary-plane runes). Invalid UTF-8 is counted as one
// code unit per byte to stay deterministic.
func utf16ColumnAt(s string, lineStart, abs int) int {
	if abs <= lineStart {
		return 0
	}
	if abs > len(s) {
		abs = len(s)
	}
	col := 0
	i := lineStart
	for i < abs {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			col++
			i++
			continue
		}
		col += utf16.RuneLen(r)
		i += size
	}
	return col
}
