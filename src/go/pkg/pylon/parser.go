package pylon

import (
	"regexp"
	"strings"
)

// DefaultExample matches the JS default (empty-source fallback).
const DefaultExample = "[- Pylon WYSIWYG -]"

var frontmatterRe = regexp.MustCompile(`(?s)\A---[ \t]*\r?\n(.*?)\r?\n---[ \t]*\r?\n?`)

// Parse turns source into an AST (root *Box). Empty input falls back
// to DefaultExample; a leading `---...---` block is parsed as
// frontmatter metadata attached to the returned root.
func Parse(source string) AST {
	if strings.TrimSpace(source) == "" {
		return Parse(DefaultExample)
	}
	top := NewSource(source)
	// Absolute offset of the trimmed source within the original text.
	// TrimSpace strips leading whitespace (including newlines); Positions
	// must reference the untrimmed source so LSP ranges line up with the
	// document as the editor sees it.
	trimLead := len(source) - len(strings.TrimLeft(source, " \t\r\n"))
	srcTrimmed := strings.TrimSpace(source)

	var meta Meta
	body := srcTrimmed
	bodyRelToTrimmed := 0
	if fm := frontmatterRe.FindStringSubmatchIndex(srcTrimmed); fm != nil {
		inner := srcTrimmed[fm[2]:fm[3]]
		// parseFrontmatter needs absolute positions so spans on Meta
		// resolve to the original source. Slice a sub-Source over the
		// inner bytes via the document's trim-lead offset.
		innerAbs := trimLead + fm[2]
		innerSrc := top.Sub(innerAbs, innerAbs+len(inner))
		meta = parseFrontmatter(innerSrc)
		afterFM := srcTrimmed[fm[1]:]
		// Body starts at the first non-whitespace byte after the closing
		// `---` fence, matching strings.TrimSpace semantics.
		lead := len(afterFM) - len(strings.TrimLeft(afterFM, " \t\r\n"))
		bodyRelToTrimmed = fm[1] + lead
		body = strings.TrimSpace(afterFM)
	}
	if body == "" {
		// Frontmatter-only input: render the default example, but keep
		// the parsed Meta from the real source. Spans for the default
		// example reference the default text (the user's source has no
		// body to address).
		defSrc := NewSource(DefaultExample)
		root := buildRoot(defSrc, DefaultExample, meta)
		return root
	}
	bodyAbs := trimLead + bodyRelToTrimmed
	bodySrc := top.Sub(bodyAbs, bodyAbs+len(body))
	return buildRoot(bodySrc, body, meta)
}

// buildRoot shared tail of Parse: run parseItems, decide whether the
// single-item shortcut applies, synthesise a root box otherwise, then
// run name collection / ref resolution.
func buildRoot(bodySrc *Source, body string, meta Meta) *Box {
	items := parseItems(bodySrc)
	var root *Box
	if len(items) == 1 {
		if b, ok := items[0].(*Box); ok {
			root = b
		}
	}
	if root == nil {
		bareText := false
		if len(items) == 1 {
			if _, ok := items[0].(*Text); ok {
				bareText = true
			}
		}
		root = &Box{
			Bordered: bareText,
			Align:    AlignCenter,
			Items:    items,
			Span:     bodySrc.SpanOf(0, len(body)),
		}
	}
	root.Meta = meta

	nameMap := map[string]*Box{}
	rep := &parseReport{}
	for _, e := range meta.Errors {
		rep.errs = append(rep.errs, e.Message)
	}
	collectNames(root, nameMap, rep)
	resolveRefs(root, nameMap, rep, map[*Box]bool{})
	if len(rep.errs) > 0 {
		root.Errors = dedupe(rep.errs)
	}
	if len(rep.unresolvedRefs) > 0 {
		root.UnresolvedRefs = rep.unresolvedRefs
	}
	if len(rep.duplicateNames) > 0 {
		root.DuplicateNames = rep.duplicateNames
	}
	return root
}

// parseReport accumulates parse-time errors surfaced by collectNames
// and resolveRefs. The string form feeds the legacy root.Errors
// pipeline; the span-ful slices feed pkg/pylon.Validate().
type parseReport struct {
	errs           []string
	unresolvedRefs []UnresolvedRef
	duplicateNames []DuplicateName
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ---- item parser --------------------------------------------------

var (
	alignLeftRe     = regexp.MustCompile(`^-\s+`)
	alignRightRe    = regexp.MustCompile(`\s+-$`)
	nameDeclRe      = regexp.MustCompile(`::\s*([A-Za-z_]\w*)\s*$`)
	refRe           = regexp.MustCompile(`^&([A-Za-z_]\w*)`)
	dataRefIdentRe  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)`)
	edgeRe          = regexp.MustCompile(`^<?-+>?`)
	dashOnlyRe      = regexp.MustCompile(`^-+$`)
	rendererTrailRe = regexp.MustCompile(`\s+\|\s+([A-Za-z_][A-Za-z0-9_]*)(?::([A-Za-z_][A-Za-z0-9_-]*))?(?:\s+\|\s+[A-Za-z_][A-Za-z0-9_]*(?::[A-Za-z_][A-Za-z0-9_-]*)?)*\s*$`)
)

// parseItems turns the inner content of a box into a slice of child
// nodes. Newlines separate items vertically; lines containing at
// least one edge token wrap into a *Row.
//
// src is a view over the body text the caller wants parsed. All Span
// fields populated on returned nodes reference absolute positions in
// the original source (via src's base + linePrefix).
func parseItems(src *Source) []Node {
	view := src.View()
	items := []Node{}
	var lineItems []Node
	var textBuf strings.Builder

	flushText := func() {
		t := strings.TrimSpace(textBuf.String())
		textBuf.Reset()
		if t != "" {
			lineItems = append(lineItems, &Text{Content: t})
		}
	}

	flushLine := func() {
		flushText()
		if len(lineItems) == 0 {
			return
		}
		lineItems = stitchLabeledEdges(lineItems)
		hasEdge := false
		for _, it := range lineItems {
			if _, ok := it.(*Edge); ok {
				hasEdge = true
				break
			}
		}
		if hasEdge {
			items = append(items, &Row{Items: lineItems})
		} else {
			items = append(items, lineItems...)
		}
		lineItems = nil
	}

	i := 0
	for i < len(view) {
		c := view[i]
		switch {
		case c == '[' || c == '(':
			flushText()
			end := findMatching(view, i)
			if end < 0 {
				textBuf.WriteByte(c)
				i++
				continue
			}
			sub := src.Sub(i, end+1)
			lineItems = append(lineItems, parseBracketedNode(sub))
			i = end + 1
		case c == '\n':
			flushLine()
			i++
		case c == '&':
			if m := refRe.FindStringSubmatch(view[i:]); m != nil {
				flushText()
				refLen := len(m[0])
				lineItems = append(lineItems, &Ref{
					Name: m[1],
					Span: src.SpanOf(i, i+refLen),
				})
				i += refLen
				continue
			}
			textBuf.WriteByte(c)
			i++
		case c == '@':
			prevCh := byte(0)
			if i > 0 {
				prevCh = view[i-1]
			}
			identMatch := dataRefIdentRe.FindStringSubmatch(view[i+1:])
			if identMatch != nil {
				var nextCh byte
				if i+1+len(identMatch[0]) < len(view) {
					nextCh = view[i+1+len(identMatch[0])]
				}
				if isDataRefBoundary(prevCh, nextCh) {
					flushText()
					drefLen := 1 + len(identMatch[0])
					lineItems = append(lineItems, &DataRef{
						Name: identMatch[1],
						Span: src.SpanOf(i, i+drefLen),
					})
					i += drefLen
					continue
				}
			}
			textBuf.WriteByte(c)
			i++
		case c == '<' || c == '-':
			match := edgeRe.FindString(view[i:])
			hasLeft := strings.HasPrefix(match, "<")
			hasRight := strings.HasSuffix(match, ">")
			if match != "" && (hasLeft || hasRight) {
				flushText()
				dashes := len(match)
				if hasLeft {
					dashes--
				}
				if hasRight {
					dashes--
				}
				direction := DirRight
				if hasLeft && hasRight {
					direction = DirBoth
				} else if hasLeft {
					direction = DirLeft
				}
				ln := dashes
				if ln < 1 {
					ln = 1
				}
				lineItems = append(lineItems, &Edge{Direction: direction, Length: ln})
				i += len(match)
				continue
			}
			textBuf.WriteByte(c)
			i++
		default:
			textBuf.WriteByte(c)
			i++
		}
	}
	flushLine()
	return items
}

// findMatching mirrors the JS helper: counts depth of the same bracket
// kind starting at s[start] and returns the index of the matching
// close, or -1 if unbalanced / unexpected opener.
func findMatching(s string, start int) int {
	open := s[start]
	var close byte
	switch open {
	case '[':
		close = ']'
	case '(':
		close = ')'
	default:
		return -1
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// isDataRefBoundary mirrors the JS boundary rule for `@ident`.
func isDataRefBoundary(prev, next byte) bool {
	if prev != 0 && !(prev == ' ' || prev == '\t' || prev == '\n' ||
		prev == '[' || prev == '(') {
		return false
	}
	if next == 0 {
		return true
	}
	switch next {
	case ' ', '\t', '\n', '|', ']', ')':
		return true
	}
	return false
}

// parseBracketedNode parses a single `[...]` / `(...)` Source view,
// returning a *Box with Span covering the full bracketed range.
// Handles alignment markers, `::` name declaration, `| renderer`
// trailing marker, then recurses into the remaining body via a
// sub-Source so nested nodes carry absolute positions.
func parseBracketedNode(src *Source) *Box {
	view := src.View()
	if len(view) == 0 {
		return &Box{Bordered: true, Align: AlignCenter, Span: src.SpanOf(0, 0)}
	}
	open := view[0]
	bordered := open == '['
	var closeCh byte
	if bordered {
		closeCh = ']'
	} else {
		closeCh = ')'
	}
	if view[len(view)-1] != closeCh {
		return &Box{
			Bordered: true,
			Align:    AlignCenter,
			Items:    []Node{&Text{Content: strings.TrimSpace(view)}},
			Span:     src.SpanOf(0, len(view)),
		}
	}
	inner := view[1 : len(view)-1]
	align, clean, cleanStart := extractAlign(inner)
	body := clean
	var name string
	if m := nameDeclRe.FindStringSubmatchIndex(clean); m != nil {
		name = clean[m[2]:m[3]]
		body = strings.TrimRight(clean[:m[0]], " \t")
	}
	var renderer, rendererArg string
	if m := rendererTrailRe.FindStringSubmatchIndex(body); m != nil {
		renderer = body[m[2]:m[3]]
		// Group 2 captures the optional `:arg` suffix on the FIRST
		// renderer pipe. Subsequent pipes (chained renderers) are
		// matched by the outer non-capturing group and discarded —
		// only the first renderer wins by existing convention.
		if m[4] >= 0 {
			rendererArg = body[m[4]:m[5]]
		}
		body = strings.TrimRight(body[:m[0]], " \t")
	}

	var items []Node
	if body != "" {
		// body is always a prefix of clean (name / renderer are
		// stripped from the tail only), and extractAlign already told
		// us where clean sits inside inner. 1 accounts for the
		// opening bracket byte at view[0].
		bodyAbs := 1 + cleanStart
		bodySrc := src.Sub(bodyAbs, bodyAbs+len(body))
		items = parseItems(bodySrc)
	}

	box := &Box{
		Bordered: bordered,
		Align:    align,
		Items:    items,
		Span:     src.SpanOf(0, len(view)),
	}
	if name != "" {
		box.Name = name
	}
	if renderer != "" {
		box.Renderer = renderer
		box.RendererArg = rendererArg
	}
	return box
}

// extractAlign strips leading `-\s+` / trailing `\s+-` alignment
// markers from inner and returns (align, cleaned, cleanedStart). The
// cleanedStart offset is the byte index of cleaned[0] within inner,
// so callers can map positions back without an O(n·m) substring
// search. cleaned may be empty when the body between the alignment
// markers is whitespace-only.
func extractAlign(inner string) (string, string, int) {
	leftLoc := alignLeftRe.FindStringIndex(inner)   // nil or [0, n]
	rightLoc := alignRightRe.FindStringIndex(inner) // nil or [m, len(inner)]
	align := AlignCenter
	switch {
	case leftLoc != nil && rightLoc == nil:
		align = AlignRight
	case leftLoc == nil && rightLoc != nil:
		align = AlignLeft
	}
	lo := 0
	hi := len(inner)
	if leftLoc != nil {
		lo = leftLoc[1]
	}
	if rightLoc != nil {
		hi = rightLoc[0]
	}
	middle := inner[lo:hi]
	// Mirror the previous strings.TrimSpace(cleaned) behaviour and
	// report the absolute start of the surviving content.
	lead := len(middle) - len(strings.TrimLeft(middle, " \t\n\r"))
	trail := len(middle) - len(strings.TrimRight(middle, " \t\n\r"))
	return align, middle[lead : len(middle)-trail], lo + lead
}

// stitchLabeledEdges collapses the 3-item `dashes/edge + borderless-box
// + dashes/edge` pattern into a single Edge with a Label, when at
// least one of the outer items is an arrow-bearing edge.
func stitchLabeledEdges(items []Node) []Node {
	out := []Node{}
	i := 0
	isDashes := func(n Node) bool {
		t, ok := n.(*Text)
		return ok && dashOnlyRe.MatchString(t.Content)
	}
	isEdge := func(n Node) bool {
		_, ok := n.(*Edge)
		return ok
	}
	isBorderlessBox := func(n Node) bool {
		b, ok := n.(*Box)
		return ok && !b.Bordered
	}
	for i < len(items) {
		if i+2 < len(items) {
			a, b, c := items[i], items[i+1], items[i+2]
			if (isDashes(a) || isEdge(a)) &&
				isBorderlessBox(b) &&
				(isDashes(c) || isEdge(c)) &&
				(isEdge(a) || isEdge(c)) {
				ea, _ := a.(*Edge)
				ec, _ := c.(*Edge)
				hasL := ea != nil && (ea.Direction == DirLeft || ea.Direction == DirBoth)
				hasR := ec != nil && (ec.Direction == DirRight || ec.Direction == DirBoth)
				var direction string
				switch {
				case hasL && hasR:
					direction = DirBoth
				case hasL:
					direction = DirLeft
				default:
					direction = DirRight
				}
				leftLen := 0
				if ea != nil {
					leftLen = ea.Length
				} else if t, ok := a.(*Text); ok {
					leftLen = len(t.Content)
				}
				rightLen := 0
				if ec != nil {
					rightLen = ec.Length
				} else if t, ok := c.(*Text); ok {
					rightLen = len(t.Content)
				}
				label, _ := b.(*Box)
				out = append(out, &Edge{
					Direction: direction,
					Label:     label,
					LeftLen:   leftLen,
					RightLen:  rightLen,
				})
				i += 3
				continue
			}
		}
		out = append(out, items[i])
		i++
	}
	return out
}

// ---- ref resolution -----------------------------------------------

// collectNames walks the AST collecting every Box.Name into nameMap.
// Duplicate names are recorded as errors; first occurrence wins. The
// duplicate's full span is also captured so pkg/pylon.Validate can
// surface it to the LSP.
func collectNames(n Node, nameMap map[string]*Box, rep *parseReport) {
	switch x := n.(type) {
	case *Box:
		if x.Name != "" {
			if _, exists := nameMap[x.Name]; exists {
				rep.errs = append(rep.errs, "Duplicate node name: "+x.Name)
				rep.duplicateNames = append(rep.duplicateNames, DuplicateName{Name: x.Name, Span: x.Span})
			} else {
				nameMap[x.Name] = x
			}
		}
		for _, c := range x.Items {
			collectNames(c, nameMap, rep)
		}
	case *Row:
		for _, c := range x.Items {
			collectNames(c, nameMap, rep)
		}
	case *Edge:
		if x.Label != nil {
			collectNames(x.Label, nameMap, rep)
		}
	}
}

// resolveRefs tags Ref nodes as SameRowArc / CrossRow where applicable,
// or replaces them with inline Text (unresolved / already-claimed).
func resolveRefs(n Node, nameMap map[string]*Box, rep *parseReport, claimed map[*Box]bool) {
	switch x := n.(type) {
	case *Box:
		x.Items = resolveRefsInItems(x.Items, nameMap, rep, claimed)
		for _, c := range x.Items {
			resolveRefs(c, nameMap, rep, claimed)
		}
	case *Row:
		tagSameRowArcs(x, nameMap)
		markCrossRowRefs(x, nameMap, claimed)
		x.Items = resolveRefsInItems(x.Items, nameMap, rep, claimed)
		for _, c := range x.Items {
			resolveRefs(c, nameMap, rep, claimed)
		}
	case *Edge:
		if x.Label != nil {
			resolveRefs(x.Label, nameMap, rep, claimed)
		}
	}
}

func resolveRefsInItems(items []Node, nameMap map[string]*Box, rep *parseReport, claimed map[*Box]bool) []Node {
	out := make([]Node, 0, len(items))
	for _, c := range items {
		if r, ok := c.(*Ref); ok {
			if r.SameRowArc || r.CrossRow {
				out = append(out, c)
				continue
			}
			out = append(out, resolveRefText(r, nameMap, rep))
			continue
		}
		if e, ok := c.(*Edge); ok && e.Label != nil {
			// Labels may hold refs; let the walker handle them via
			// recursion in the caller.
			_ = e
		}
		out = append(out, c)
	}
	return out
}

func resolveRefText(r *Ref, nameMap map[string]*Box, rep *parseReport) Node {
	if _, ok := nameMap[r.Name]; !ok {
		rep.errs = append(rep.errs, "Undefined ref: &"+r.Name)
		rep.unresolvedRefs = append(rep.unresolvedRefs, UnresolvedRef{Name: r.Name, Span: r.Span})
		return &Text{Content: "&" + r.Name}
	}
	return &Text{Content: r.Name}
}

// tagSameRowArcs marks refs whose target box lives in the same row.
// Also consumes the edge immediately preceding a same-row ref so it
// does not render inline.
func tagSameRowArcs(row *Row, nameMap map[string]*Box) {
	items := row.Items
	boxByName := map[string]*Box{}
	for _, c := range items {
		if b, ok := c.(*Box); ok && b.Name != "" {
			boxByName[b.Name] = b
		}
	}
	boxLike := func(n Node) *Box {
		if b, ok := n.(*Box); ok {
			return b
		}
		if r, ok := n.(*Ref); ok && r.Target != nil {
			return r.Target
		}
		return nil
	}
	for i, it := range items {
		ref, ok := it.(*Ref)
		if !ok {
			continue
		}
		target, ok := boxByName[ref.Name]
		if !ok {
			continue
		}
		var sourceBox *Box
		if i >= 2 {
			if _, isEdge := items[i-1].(*Edge); isEdge {
				sourceBox = boxLike(items[i-2])
			}
		}
		if sourceBox == nil && i >= 1 {
			sourceBox = boxLike(items[i-1])
		}
		ref.SameRowArc = true
		ref.Target = target
		if sourceBox != nil {
			ref.SourceBox = sourceBox
		} else {
			ref.SourceBox = target
		}
		if i > 0 {
			if e, ok := items[i-1].(*Edge); ok {
				e.consumedBySameRowArc = true
			}
		}
	}
}

// markCrossRowRefs tags refs whose declaration lives in a different
// top-level row. Each target gets at most one gutter arrow.
func markCrossRowRefs(row *Row, nameMap map[string]*Box, claimed map[*Box]bool) {
	for i, it := range row.Items {
		ref, ok := it.(*Ref)
		if !ok {
			continue
		}
		if ref.SameRowArc {
			continue
		}
		target, ok := nameMap[ref.Name]
		if !ok {
			continue
		}
		if claimed[target] {
			continue
		}
		claimed[target] = true
		ref.CrossRow = true
		ref.Target = target
		if i > 0 {
			if e, ok := row.Items[i-1].(*Edge); ok {
				e.consumedByCrossRow = true
			}
		}
	}
}
