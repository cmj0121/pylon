package pylon

import (
	"strings"
	"unicode/utf8"
)

// Box-drawing glyph palette. The JS side supports a few themes; this
// commit ships `simple` (default Unicode) and `ascii` fallbacks.
type boxChars struct {
	tl, tr, bl, br string
	h, v           string
	arrowL, arrowR string
}

var unicodeBox = boxChars{
	tl: "┌", tr: "┐", bl: "└", br: "┘",
	h: "─", v: "│",
	arrowL: "◀", arrowR: "▶",
}

var asciiBox = boxChars{
	tl: "+", tr: "+", bl: "+", br: "+",
	h: "-", v: "|",
	arrowL: "<", arrowR: ">",
}

func boxCharsFor(ast *Box) boxChars {
	if ast.Meta.Theme == "ascii" {
		return asciiBox
	}
	return unicodeBox
}

// RenderASCII turns an AST into a newline-joined ASCII block (no
// trailing newline), matching the JS `exporters.ascii` format.
func RenderASCII(ast *Box) string {
	return strings.Join(RenderRows(ast), "\n")
}

// RenderRows turns an AST into a flat slice of display rows.
func RenderRows(ast *Box) []string {
	bc := boxCharsFor(ast)
	var targetW, targetH *int
	if ast.Meta.Size != nil {
		w := ast.Meta.Size.W
		h := ast.Meta.Size.H
		targetW = &w
		targetH = &h
	}
	data := ast.Meta.Data
	rows := renderBoxRows(ast, bc, targetW, targetH, data)
	return overlayCrossRowGutter(rows, ast, bc, data)
}

const (
	naturalPad    = 3
	tightChartPad = 0
)

// isTightChartBox reports whether a chart-primitive box should render
// with zero inner padding instead of the default 3-cell pad. Targets
// the bar / heatmap / sparkline / candlestick / hist / step / gantt /
// progress family — primitives whose bodies are dense data-viz grids
// where the 3-cell margin wastes screen real estate in SVG/PNG
// output. `banner` and `text` stay at the natural pad because they're
// literal text that benefits from breathing room.
func isTightChartBox(b *Box) bool {
	switch b.Renderer {
	case "bar", "hbar", "vbar", "progress",
		"heatmap", "sparkline", "candlestick",
		"hist", "step", "gantt":
		return true
	}
	return false
}

// renderBoxRows mirrors the JS function of the same name. The data
// argument carries the root's frontmatter series map so chart boxes
// anywhere in the tree can resolve @ref; normal (non-chart) boxes
// ignore it.
func renderBoxRows(ast *Box, bc boxChars, targetW, targetH *int, data interface{}) []string {
	items := ast.Items
	align := ast.Align
	if align == "" {
		align = AlignCenter
	}
	bordered := ast.Bordered

	// Chart primitives tighten inner padding to a single cell so SVG/
	// PNG output doesn't carry 3 cells of empty margin on each side of
	// every bar / heatmap / sparkline / etc. 1 cell is the floor: hbar
	// / vbar emit an internal `│` separator that would collide with
	// the outer border at zero pad. `banner` and `text` keep the
	// natural 3-cell pad because they're text formatting, not data
	// viz.
	pad := naturalPad
	if isTightChartBox(ast) {
		pad = tightChartPad
	}
	hasPad := bordered || len(items) > 1
	padBudget := 0
	if hasPad {
		padBudget = 2 * pad
	}
	borderBudget := 0
	if bordered {
		borderBudget = 2
	}

	var sizedContentW *int
	if targetW != nil {
		cw := *targetW - borderBudget
		if hasPad {
			cw -= 2 * pad
		}
		if cw < 1 {
			cw = 1
		}
		sizedContentW = &cw
	}

	// Chart dispatch: any box that carries a renderer OR a bare @ref
	// goes through applyChartRenderer so the shared rendererInlineError
	// helper can emit the ⚠ row for malformed inputs (including the
	// bare-@ref case where Renderer is empty). nil signals the
	// "| text over raw text" no-op, where the normal Items path runs.
	var itemRows []string
	if ast.Renderer != "" || firstDataRef(ast) != nil {
		itemRows = applyChartRenderer(ast, data, bc)
	}
	if itemRows == nil {
		itemRows = []string{}
		for _, it := range items {
			itemRows = append(itemRows, renderItemRows(it, bc, sizedContentW, data)...)
		}
	}
	naturalContentW := 1
	for _, r := range itemRows {
		if w := displayWidth(r); w > naturalContentW {
			naturalContentW = w
		}
	}
	naturalOuterW := naturalContentW + padBudget + borderBudget
	naturalOuterH := len(itemRows) + borderBudget

	outerW := naturalOuterW
	if targetW != nil && *targetW < outerW {
		outerW = *targetW
	}
	outerH := naturalOuterH
	if targetH != nil && *targetH < outerH {
		outerH = *targetH
	}

	contentW := outerW - borderBudget
	if contentW < 0 {
		contentW = 0
	}
	contentH := outerH - borderBudget
	if contentH < 0 {
		contentH = 0
	}

	clipped := make([]string, len(itemRows))
	for i, r := range itemRows {
		clipped[i] = clipRow(r, contentW)
	}
	paddedH := vertPad(clipped, contentH, contentW)
	padded := make([]string, len(paddedH))
	for i, r := range paddedH {
		padded[i] = padRow(r, contentW, align, 1)
	}

	if !bordered {
		return padded
	}
	hLine := strings.Repeat(bc.h, contentW)
	out := make([]string, 0, len(padded)+2)
	out = append(out, bc.tl+hLine+bc.tr)
	for _, r := range padded {
		out = append(out, bc.v+r+bc.v)
	}
	out = append(out, bc.bl+hLine+bc.br)
	return out
}

// renderItemRows renders one AST item to its natural rows. data is
// threaded down so nested chart boxes can resolve @ref.
func renderItemRows(n Node, bc boxChars, maxW *int, data interface{}) []string {
	switch x := n.(type) {
	case *Text:
		return []string{x.Content}
	case *Box:
		return renderBoxRows(x, bc, nil, nil, data)
	case *Row:
		return renderRowRows(x, bc, maxW, data)
	}
	return nil
}

// edgeHeads mirrors the JS helper.
func edgeHeads(direction string) (head, tail bool) {
	head = direction == DirRight || direction == DirBoth
	tail = direction == DirLeft || direction == DirBoth
	return
}

// edgeString returns the inline-text representation of an edge.
// data is threaded for the edge-label case so a chart renderer
// inside `( @ref | bar )` would resolve — grammar permits it even
// though no current fixture uses it.
func edgeString(e *Edge, bc boxChars, data interface{}) string {
	head, tail := edgeHeads(e.Direction)
	leftArrow := ""
	if tail {
		leftArrow = bc.arrowL
	}
	rightArrow := ""
	if head {
		rightArrow = bc.arrowR
	}
	if e.Label != nil {
		labelRows := renderBoxRows(e.Label, bc, nil, nil, data)
		labelText := ""
		if len(labelRows) > 0 {
			labelText = labelRows[0]
		}
		dashes := strings.Repeat(bc.h, 2)
		pad := "  "
		return leftArrow + dashes + pad + labelText + pad + dashes + rightArrow
	}
	return leftArrow + bc.h + rightArrow
}

// ---- row rendering -----------------------------------------------

type rowPart struct {
	kind   string // "edge" | "block"
	item   Node
	edge   *Edge
	text   string
	rows   []string
	width  int
	height int
}

func renderRowRows(row *Row, bc boxChars, maxW *int, data interface{}) []string {
	linear := []Node{}
	for _, it := range row.Items {
		if r, ok := it.(*Ref); ok {
			if r.SameRowArc || r.CrossRow {
				continue
			}
		}
		if e, ok := it.(*Edge); ok {
			if e.consumedBySameRowArc || e.consumedByCrossRow {
				continue
			}
		}
		linear = append(linear, it)
	}

	parts := make([]rowPart, 0, len(linear))
	for _, it := range linear {
		if e, ok := it.(*Edge); ok {
			txt := edgeString(e, bc, data)
			parts = append(parts, rowPart{
				kind: "edge", edge: e, item: it, text: txt,
				width: displayWidth(txt),
			})
			continue
		}
		rows := renderItemRows(it, bc, nil, data)
		w := 0
		for _, r := range rows {
			if rw := displayWidth(r); rw > w {
				w = rw
			}
		}
		parts = append(parts, rowPart{
			kind: "block", item: it, rows: rows,
			width: w, height: len(rows),
		})
	}

	totalWidth := 0
	for _, p := range parts {
		totalWidth += p.width
	}
	if maxW != nil && totalWidth > *maxW {
		return renderVerticalChain(parts, bc, data)
	}

	h := 1
	for _, p := range parts {
		if p.kind == "block" && p.height > h {
			h = p.height
		}
	}
	midline := (h - 1) / 2

	columns := make([][]string, len(parts))
	for pi, p := range parts {
		blank := strings.Repeat(" ", p.width)
		if p.kind == "edge" {
			col := make([]string, h)
			for r := 0; r < h; r++ {
				if r == midline {
					col[r] = p.text
				} else {
					col[r] = blank
				}
			}
			columns[pi] = col
			continue
		}
		padded := make([]string, len(p.rows))
		for i, r := range p.rows {
			padded[i] = padRow(r, p.width, AlignLeft, 1)
		}
		extra := h - len(padded)
		top := extra / 2
		bot := extra - top
		col := make([]string, 0, h)
		for i := 0; i < top; i++ {
			col = append(col, blank)
		}
		col = append(col, padded...)
		for i := 0; i < bot; i++ {
			col = append(col, blank)
		}
		columns[pi] = col
	}

	out := make([]string, h)
	for r := 0; r < h; r++ {
		var sb strings.Builder
		for _, col := range columns {
			sb.WriteString(col[r])
		}
		out[r] = sb.String()
	}

	// Same-row arcs.
	arcs := []*Ref{}
	for _, it := range row.Items {
		if r, ok := it.(*Ref); ok && r.SameRowArc {
			arcs = append(arcs, r)
		}
	}
	if len(arcs) > 0 && totalWidth > 0 {
		out = paintSameRowArcs(out, parts, arcs, totalWidth, bc)
	}
	return out
}

// paintSameRowArcs paints the U-arcs beneath / above a row for
// same-row refs. Mirrors the JS compact + non-compact paths.
func paintSameRowArcs(out []string, parts []rowPart, arcs []*Ref, totalWidth int, bc boxChars) []string {
	partStart := map[Node]int{}
	{
		c := 0
		for _, p := range parts {
			partStart[p.item] = c
			c += p.width
		}
	}
	blockPart := func(target *Box) *rowPart {
		for i := range parts {
			if parts[i].kind == "block" && parts[i].item == Node(target) {
				return &parts[i]
			}
		}
		return nil
	}

	type resolved struct {
		tgtStart, tgtW int
		srcStart, srcW int
		hasSrc         bool
		lo, hi         int
	}
	belowResolved := []resolved{}
	selfResolved := []resolved{}
	hasCrossBox := false
	for _, ref := range arcs {
		tgtPart := blockPart(ref.Target)
		if tgtPart == nil {
			continue
		}
		tgtStart := partStart[tgtPart.item]
		tgtW := tgtPart.width
		if ref.SourceBox == ref.Target {
			selfResolved = append(selfResolved, resolved{
				tgtStart: tgtStart, tgtW: tgtW, hasSrc: false,
			})
			continue
		}
		srcPart := blockPart(ref.SourceBox)
		if srcPart == nil {
			continue
		}
		srcStart := partStart[srcPart.item]
		srcW := srcPart.width
		hasCrossBox = true
		lo := tgtStart
		hi := srcStart
		if srcStart < tgtStart {
			lo, hi = srcStart, tgtStart
		}
		belowResolved = append(belowResolved, resolved{
			tgtStart: tgtStart, tgtW: tgtW,
			srcStart: srcStart, srcW: srcW,
			hasSrc: true, lo: lo, hi: hi,
		})
	}
	aboveResolved := []resolved{}
	if hasCrossBox {
		aboveResolved = selfResolved
	} else {
		belowResolved = append(belowResolved, selfResolved...)
	}

	compact := len(belowResolved) == 2
	if compact {
		for _, r := range belowResolved {
			if !r.hasSrc {
				compact = false
				break
			}
		}
	}
	if compact {
		if belowResolved[0].lo != belowResolved[1].lo ||
			belowResolved[0].hi != belowResolved[1].hi {
			compact = false
		}
	}

	type specT struct {
		armL, armR, tgtArm int
		arrowRow           int
		cornerRow          int
	}
	belowSpecs := []specT{}
	for i, r := range belowResolved {
		var armL, armR, tgtArm int
		if !r.hasSrc {
			armL = r.tgtStart + 2
			armR = r.tgtStart + r.tgtW - 3
			tgtArm = armR
		} else if compact {
			isTgtLeft := r.tgtStart < r.srcStart
			leftStart, leftW := r.srcStart, r.srcW
			rightStart, rightW := r.tgtStart, r.tgtW
			if isTgtLeft {
				leftStart, leftW = r.tgtStart, r.tgtW
				rightStart, rightW = r.srcStart, r.srcW
			}
			if i == 0 {
				armL = leftStart + leftW - 3
				armR = rightStart + 2
			} else {
				armL = leftStart + 2
				armR = rightStart + rightW - 3
			}
			if isTgtLeft {
				tgtArm = armL
			} else {
				tgtArm = armR
			}
		} else {
			tgtCol := r.tgtStart + r.tgtW - 3
			srcCol := r.srcStart + r.srcW - 3
			armL = tgtCol
			armR = srcCol
			if srcCol < tgtCol {
				armL, armR = srcCol, tgtCol
			}
			tgtArm = tgtCol
		}
		if armR > armL {
			belowSpecs = append(belowSpecs, specT{armL: armL, armR: armR, tgtArm: tgtArm})
		}
	}

	blankCells := func() []rune {
		s := make([]rune, totalWidth)
		for i := range s {
			s[i] = ' '
		}
		return s
	}
	hRune := []rune(bc.h)[0]
	vRune := []rune(bc.v)[0]

	paintArrow := func(cells []rune, spec specT, head rune) {
		if spec.tgtArm == spec.armL {
			cells[spec.armL] = head
		} else {
			cells[spec.armL] = vRune
		}
		if spec.tgtArm == spec.armR {
			cells[spec.armR] = head
		} else {
			cells[spec.armR] = vRune
		}
	}
	paintCorner := func(cells []rune, spec specT, left, right rune) {
		cells[spec.armL] = left
		for c := spec.armL + 1; c < spec.armR; c++ {
			cells[c] = hRune
		}
		cells[spec.armR] = right
	}

	bodyOut := make([][]rune, len(out))
	for i, s := range out {
		bodyOut[i] = []rune(s)
	}

	gridFromBody := func(rowsIn [][]rune) [][]rune { return rowsIn }

	// helper: stretch arms upward through blanks, converting arrow
	// heads to ▲ at the topmost blank cell.
	extendArmsUp := func(grid [][]rune, specs []specT) {
		for _, spec := range specs {
			for _, c := range []int{spec.armL, spec.armR} {
				isArrow := c == spec.tgtArm
				top := spec.arrowRow
				for r := spec.arrowRow - 1; r >= 0; r-- {
					var ch rune
					if c < len(grid[r]) {
						ch = grid[r][c]
					} else {
						ch = ' '
					}
					if ch != ' ' {
						break
					}
					top = r
				}
				for r := top; r < spec.arrowRow; r++ {
					ensureWidth(&grid[r], c+1)
					grid[r][c] = vRune
				}
				if isArrow && top < spec.arrowRow {
					ensureWidth(&grid[spec.arrowRow], c+1)
					grid[spec.arrowRow][c] = vRune
					ensureWidth(&grid[top], c+1)
					grid[top][c] = '▲'
				}
			}
		}
	}

	upHead := []rune("▲")[0]

	if compact && len(belowSpecs) == 2 {
		// sort by width ascending so inner is first
		if (belowSpecs[0].armR - belowSpecs[0].armL) > (belowSpecs[1].armR - belowSpecs[1].armL) {
			belowSpecs[0], belowSpecs[1] = belowSpecs[1], belowSpecs[0]
		}
		arrowRowIdx := len(bodyOut)
		arrow := blankCells()
		for i := range belowSpecs {
			paintArrow(arrow, belowSpecs[i], upHead)
			belowSpecs[i].arrowRow = arrowRowIdx
		}
		bodyOut = append(bodyOut, arrow)
		for i := range belowSpecs {
			corner := blankCells()
			paintCorner(corner, belowSpecs[i], []rune(bc.bl)[0], []rune(bc.br)[0])
			belowSpecs[i].cornerRow = len(bodyOut)
			bodyOut = append(bodyOut, corner)
		}
		grid := gridFromBody(bodyOut)
		for _, spec := range belowSpecs {
			for _, c := range []int{spec.armL, spec.armR} {
				for r := arrowRowIdx + 1; r < spec.cornerRow; r++ {
					ensureWidth(&grid[r], c+1)
					if grid[r][c] == ' ' {
						grid[r][c] = vRune
					}
				}
			}
		}
		extendArmsUp(grid, belowSpecs)
		bodyOut = grid
	} else if len(belowSpecs) > 0 {
		for i := range belowSpecs {
			arrow := blankCells()
			paintArrow(arrow, belowSpecs[i], upHead)
			belowSpecs[i].arrowRow = len(bodyOut)
			bodyOut = append(bodyOut, arrow)
			corner := blankCells()
			paintCorner(corner, belowSpecs[i], []rune(bc.bl)[0], []rune(bc.br)[0])
			bodyOut = append(bodyOut, corner)
		}
		grid := gridFromBody(bodyOut)
		extendArmsUp(grid, belowSpecs)
		bodyOut = grid
	}

	aboveRows := [][]rune{}
	downHead := []rune("▼")[0]
	for _, r := range aboveResolved {
		armL := r.tgtStart + 2
		armR := r.tgtStart + r.tgtW - 3
		if armR <= armL {
			continue
		}
		spec := specT{armL: armL, armR: armR, tgtArm: armR}
		corner := blankCells()
		paintCorner(corner, spec, []rune(bc.tl)[0], []rune(bc.tr)[0])
		arm := blankCells()
		paintArrow(arm, spec, downHead)
		aboveRows = append(aboveRows, corner, arm)
	}

	result := make([]string, 0, len(aboveRows)+len(bodyOut))
	for _, r := range aboveRows {
		result = append(result, string(r))
	}
	for _, r := range bodyOut {
		result = append(result, string(r))
	}
	return result
}

func ensureWidth(r *[]rune, n int) {
	for len(*r) < n {
		*r = append(*r, ' ')
	}
}

// renderVerticalChain mirrors the JS fallback used when a row exceeds
// its maxW budget. Not exercised by the current fixture set but kept
// for parity.
func renderVerticalChain(parts []rowPart, bc boxChars, data interface{}) []string {
	blocks := []rowPart{}
	for _, p := range parts {
		if p.kind == "block" {
			blocks = append(blocks, p)
		}
	}
	if len(blocks) == 0 {
		return nil
	}
	maxW := 1
	for _, p := range blocks {
		if p.width > maxW {
			maxW = p.width
		}
	}
	result := []string{}
	for _, p := range parts {
		if p.kind == "block" {
			for _, r := range p.rows {
				result = append(result, padRow(r, maxW, AlignCenter, 1))
			}
			continue
		}
		down, up := edgeHeads(p.edge.Direction)
		top := "│"
		if up {
			top = "▲"
		}
		result = append(result, padRow(top, maxW, AlignCenter, 1))
		if p.edge.Label != nil {
			labelRows := renderBoxRows(p.edge.Label, bc, nil, nil, data)
			label := ""
			if len(labelRows) > 0 {
				label = labelRows[0]
			}
			result = append(result, padRow(label, maxW, AlignCenter, 1))
		}
		bot := "│"
		if down {
			bot = "▼"
		}
		result = append(result, padRow(bot, maxW, AlignCenter, 1))
	}
	return result
}

// ---- cross-row gutter --------------------------------------------

// overlayCrossRowGutter draws the right-side arcs for any top-level
// cross-row refs. Shallow — only top-level items participate. data
// threads through to the inner renderItemRows calls for consistency
// with the main render path.
func overlayCrossRowGutter(rows []string, ast *Box, bc boxChars, data interface{}) []string {
	items := ast.Items
	hasCross := false
	for _, it := range items {
		if r, ok := it.(*Row); ok {
			for _, c := range r.Items {
				if ref, ok := c.(*Ref); ok && ref.CrossRow {
					hasCross = true
					break
				}
			}
		}
		if hasCross {
			break
		}
	}
	if !hasCross {
		return rows
	}

	bordered := ast.Bordered
	hasPad := bordered || len(items) > 1
	padBudget := 0
	if hasPad {
		padBudget = 2 * naturalPad
	}
	borderBudget := 0
	if bordered {
		borderBudget = 2
	}
	var targetW *int
	if ast.Meta.Size != nil {
		w := ast.Meta.Size.W
		targetW = &w
	}
	var sizedContentW *int
	if targetW != nil {
		cw := *targetW - borderBudget - padBudget
		if cw < 1 {
			cw = 1
		}
		sizedContentW = &cw
	}

	perItemRows := map[Node][]string{}
	itemStart := map[Node]int{}
	cursor := 0
	for _, item := range items {
		r := renderItemRows(item, bc, sizedContentW, data)
		perItemRows[item] = r
		itemStart[item] = cursor
		cursor += len(r)
	}

	borderTop := 0
	if bordered {
		borderTop = 1
	}
	paddedRows := len(rows) - 2*borderTop
	extraTop := (paddedRows - cursor) / 2
	if extraTop < 0 {
		extraTop = 0
	}
	rowOffset := borderTop + extraTop

	rowWidth := 0
	for _, r := range rows {
		if w := displayWidth(r); w > rowWidth {
			rowWidth = w
		}
	}
	borderLeft := 0
	if bordered {
		borderLeft = 1
	}
	contentInnerW := rowWidth - 2*borderLeft
	centerPad := func(itemW int) int {
		p := (contentInnerW - itemW) / 2
		if p < 0 {
			return 0
		}
		return p
	}
	absCol := func(itemW, innerCol int) int {
		return borderLeft + centerPad(itemW) + innerCol
	}

	type boxPos struct {
		start    int
		height   int
		startCol int
		width    int
	}
	boxAbs := map[*Box]boxPos{}
	rowParts := map[*Row][]rowPart{}
	rowLinearWs := map[*Row]int{}

	for _, item := range items {
		base := itemStart[item] + rowOffset
		if b, ok := item.(*Box); ok && b.Name != "" {
			r := perItemRows[item]
			itemW := 0
			for _, x := range r {
				if w := displayWidth(x); w > itemW {
					itemW = w
				}
			}
			boxAbs[b] = boxPos{
				start:    base,
				height:   len(r),
				startCol: absCol(itemW, 0),
				width:    itemW,
			}
		} else if row, ok := item.(*Row); ok {
			linear := []Node{}
			for _, it := range row.Items {
				if rf, ok := it.(*Ref); ok {
					if rf.SameRowArc || rf.CrossRow {
						continue
					}
				}
				if e, ok := it.(*Edge); ok {
					if e.consumedBySameRowArc || e.consumedByCrossRow {
						continue
					}
				}
				linear = append(linear, it)
			}
			parts := []rowPart{}
			for _, it := range linear {
				if e, ok := it.(*Edge); ok {
					parts = append(parts, rowPart{
						kind: "edge", item: it, edge: e,
						width: displayWidth(edgeString(e, bc, data)),
					})
					continue
				}
				subR := renderItemRows(it, bc, nil, data)
				w := 0
				for _, x := range subR {
					if xw := displayWidth(x); xw > w {
						w = xw
					}
				}
				parts = append(parts, rowPart{
					kind: "block", item: it, rows: subR,
					width: w, height: len(subR),
				})
			}
			rowLinearW := 0
			for _, p := range parts {
				rowLinearW += p.width
			}
			innerCol := 0
			for _, p := range parts {
				if p.kind == "block" {
					if b, ok := p.item.(*Box); ok && b.Name != "" {
						boxAbs[b] = boxPos{
							start:    base,
							height:   p.height,
							startCol: absCol(rowLinearW, innerCol),
							width:    p.width,
						}
					}
				}
				innerCol += p.width
			}
			rowParts[row] = parts
			rowLinearWs[row] = rowLinearW
		}
	}

	type exit struct {
		srcRow, srcCol int
		tgtRow, tgtCol int
	}
	exits := []exit{}
	for _, item := range items {
		row, ok := item.(*Row)
		if !ok {
			continue
		}
		base := itemStart[item] + rowOffset
		parts := rowParts[row]
		rowLinearW := rowLinearWs[row]
		h := 1
		for _, p := range parts {
			if p.kind == "block" && p.height > h {
				h = p.height
			}
		}
		midlineOffset := (h - 1) / 2
		for _, it := range row.Items {
			ref, ok := it.(*Ref)
			if !ok || !ref.CrossRow {
				continue
			}
			tgt, ok := boxAbs[ref.Target]
			if !ok {
				continue
			}
			exits = append(exits, exit{
				srcRow: base + midlineOffset,
				srcCol: absCol(rowLinearW, rowLinearW),
				tgtRow: tgt.start + (tgt.height-1)/2,
				tgtCol: tgt.startCol + tgt.width,
			})
		}
	}
	if len(exits) == 0 {
		return rows
	}

	const gutterExtra = 3
	gutterCol := rowWidth + gutterExtra - 1
	newWidth := gutterCol + 1

	grid := make([][]rune, len(rows))
	for i, r := range rows {
		arr := []rune(r)
		for len(arr) < newWidth {
			arr = append(arr, ' ')
		}
		grid[i] = arr
	}

	hRune := []rune(bc.h)[0]
	vRune := []rune(bc.v)[0]
	arrowLRune := []rune(bc.arrowL)[0]
	brRune := []rune(bc.br)[0]
	trRune := []rune(bc.tr)[0]

	for _, ex := range exits {
		for c := ex.srcCol; c < gutterCol; c++ {
			ensureWidth(&grid[ex.srcRow], c+1)
			grid[ex.srcRow][c] = hRune
		}
		for c := ex.tgtCol + 1; c < gutterCol; c++ {
			ensureWidth(&grid[ex.tgtRow], c+1)
			grid[ex.tgtRow][c] = hRune
		}
		ensureWidth(&grid[ex.tgtRow], ex.tgtCol+1)
		grid[ex.tgtRow][ex.tgtCol] = arrowLRune
		if ex.srcRow > ex.tgtRow {
			ensureWidth(&grid[ex.srcRow], gutterCol+1)
			grid[ex.srcRow][gutterCol] = brRune
			ensureWidth(&grid[ex.tgtRow], gutterCol+1)
			grid[ex.tgtRow][gutterCol] = trRune
			for r := ex.tgtRow + 1; r < ex.srcRow; r++ {
				ensureWidth(&grid[r], gutterCol+1)
				if grid[r][gutterCol] == ' ' {
					grid[r][gutterCol] = vRune
				}
			}
		} else {
			ensureWidth(&grid[ex.srcRow], gutterCol+1)
			grid[ex.srcRow][gutterCol] = trRune
			ensureWidth(&grid[ex.tgtRow], gutterCol+1)
			grid[ex.tgtRow][gutterCol] = brRune
			for r := ex.srcRow + 1; r < ex.tgtRow; r++ {
				ensureWidth(&grid[r], gutterCol+1)
				if grid[r][gutterCol] == ' ' {
					grid[r][gutterCol] = vRune
				}
			}
		}
	}
	out := make([]string, len(grid))
	for i, r := range grid {
		out[i] = string(r)
	}
	return out
}

// ---- display width / padding helpers -----------------------------

func padRow(row string, targetW int, align string, minPad int) string {
	rowW := displayWidth(row)
	slack := targetW - rowW
	if slack <= 0 {
		return row
	}
	if slack <= 2*minPad || align == AlignCenter {
		l := slack / 2
		return strings.Repeat(" ", l) + row + strings.Repeat(" ", slack-l)
	}
	if align == AlignRight {
		return strings.Repeat(" ", slack-minPad) + row + strings.Repeat(" ", minPad)
	}
	return strings.Repeat(" ", minPad) + row + strings.Repeat(" ", slack-minPad)
}

func clipRow(row string, targetW int) string {
	if displayWidth(row) <= targetW {
		return row
	}
	w := 0
	var b strings.Builder
	for _, r := range row {
		cw := runeWidth(r)
		if w+cw > targetW {
			break
		}
		b.WriteRune(r)
		w += cw
	}
	return b.String()
}

func vertPad(rows []string, targetH, contentW int) []string {
	extra := targetH - len(rows)
	if extra <= 0 {
		return rows
	}
	top := extra / 2
	bot := extra - top
	blank := strings.Repeat(" ", contentW)
	out := make([]string, 0, targetH)
	for i := 0; i < top; i++ {
		out = append(out, blank)
	}
	out = append(out, rows...)
	for i := 0; i < bot; i++ {
		out = append(out, blank)
	}
	return out
}

// displayWidth counts terminal/monospace cells for a string. CJK /
// wide ranges count as 2, ZWJ / combining runes count as 0.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// runeWidth is the per-rune width lookup. Matches the JS EAW table.
func runeWidth(r rune) int {
	cp := uint32(r)
	if cp < 0x20 || (cp >= 0x7f && cp < 0xa0) {
		return 0
	}
	if inRanges(cp, eawZero) {
		return 0
	}
	if inRanges(cp, eawWide) {
		return 2
	}
	return 1
}

type cpRange struct {
	lo, hi uint32
}

func inRanges(cp uint32, ranges []cpRange) bool {
	for _, rg := range ranges {
		if cp < rg.lo {
			return false
		}
		if cp <= rg.hi {
			return true
		}
	}
	return false
}

var eawWide = []cpRange{
	{0x1100, 0x115f},
	{0x2329, 0x232a},
	{0x2e80, 0x303e},
	{0x3041, 0x33ff},
	{0x3400, 0x4dbf},
	{0x4e00, 0x9fff},
	{0xa000, 0xa4cf},
	{0xac00, 0xd7a3},
	{0xf900, 0xfaff},
	{0xfe30, 0xfe4f},
	{0xff00, 0xff60},
	{0xffe0, 0xffe6},
	{0x1f300, 0x1f64f},
	{0x1f680, 0x1f6ff},
	{0x1f900, 0x1f9ff},
	{0x20000, 0x2fffd},
	{0x30000, 0x3fffd},
}

var eawZero = []cpRange{
	{0x0300, 0x036f},
	{0x0483, 0x0489},
	{0x1ab0, 0x1aff},
	{0x1dc0, 0x1dff},
	{0x200b, 0x200f},
	{0x2028, 0x202e},
	{0x20d0, 0x20ff},
	{0xfe00, 0xfe0f},
	{0xfe20, 0xfe2f},
	{0xfeff, 0xfeff},
}

// Keep the utf8 import exercised for future needs (e.g. clip with
// grapheme awareness). The placeholder use below prevents Go from
// flagging the import as unused without reading it elsewhere.
var _ = utf8.RuneError
