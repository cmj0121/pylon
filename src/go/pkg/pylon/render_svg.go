package pylon

import (
	"fmt"
	"strings"
)

// SVG cell geometry. 5px wide / 13px tall packs Cascadia-class glyphs
// at their natural advance with zero inter-cell whitespace — the grid
// matches the PNG renderer's 10pt / 72 DPI tight cell. Fonts with
// different metrics have minor drift, but the values give a reasonable
// default viewBox. Reduce svgFontPx below 10 and text glyphs start
// overlapping adjacent cells on browsers that snap to pixel rows.
const (
	svgCellW  = 5
	svgCellH  = 13
	svgFontPx = 10
)

// Theme color palette. Matches the `--pylon-ink` custom property from
// src/js/pylon.css for each theme: `simple` / `light` / `ascii` use the
// light-mode ink; `dark` uses the dark-mode ink. Unknown themes fall
// back to the default (`simple`).
func svgFillForTheme(theme string) string {
	switch theme {
	case "dark":
		return "#e6dfc8"
	case "light", "ascii", "simple", "":
		return "#0f1c2d"
	default:
		return "#0f1c2d"
	}
}

// RenderSVG turns an AST into a complete, self-contained SVG document.
//
// The implementation reuses RenderASCII so SVG and ASCII layouts are
// byte-identical at the grid level — SVG is simply a painted version
// of the ASCII character grid, one <text> element per row.
//
// Limitation: this renderer emits one <text> per whole row. That keeps
// the output small and matches typical Latin-only diagrams. It does
// NOT align CJK / emoji cells on a per-cell basis the way the JS
// reference does (where each rune becomes a <tspan> with an explicit
// width). For diagrams that mix wide and narrow characters, or that
// rely on precise cell alignment across variable fonts, prefer the JS
// renderer. Per-cell <tspan> alignment is a potential follow-up.
func RenderSVG(ast *Box) string {
	rows := RenderRows(ast)
	if len(rows) == 0 {
		rows = []string{""}
	}

	maxW := 0
	for _, r := range rows {
		if w := displayWidth(r); w > maxW {
			maxW = w
		}
	}
	if maxW == 0 {
		maxW = 1
	}

	widthPx := maxW * svgCellW
	heightPx := len(rows) * svgCellH

	fill := svgFillForTheme(ast.Meta.Theme)

	var b strings.Builder
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`,
		widthPx, heightPx, widthPx, heightPx)
	b.WriteByte('\n')
	fmt.Fprintf(&b,
		`<style>text { font-family: "Cascadia Code", "JetBrains Mono", "Iosevka", ui-monospace, Menlo, Consolas, monospace; font-size: %dpx; fill: %s; white-space: pre; }</style>`,
		svgFontPx, fill)
	b.WriteByte('\n')

	// Baseline positions each row one cell height apart; we treat the
	// cell as containing the baseline at the bottom of the cell, which
	// places descenders near the grid line below.
	for i, row := range rows {
		baseline := (i + 1) * svgCellH
		// Nudge the baseline up (cellH - fontPx) so the line sits
		// visually centered inside its cell. For the default 13x10
		// cell/font that is a 3px shift.
		baseline -= svgCellH - svgFontPx
		emitSVGRow(&b, row, i*svgCellH, baseline, fill)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// emitSVGRow walks a single ASCII row and emits a mix of <rect> and
// <text> elements. Unicode block glyphs (`█▓▒░`) coalesce into
// contiguous <rect> runs rendered with opacity matching the ramp
// step, so chart bars show as clean solid / dithered blocks instead
// of font characters whose antialiasing differs by renderer. Every
// other cell is emitted as a <text> segment at its natural position.
//
// Each text segment carries textLength=cells*svgCellW and
// lengthAdjust="spacingAndGlyphs" so the browser compresses (or
// stretches) the glyph run to fit exactly the cell grid. Without
// this the text's natural advance drifts away from svgCellW — a
// 10px Cascadia renders at ~6px per char, while cells are 5px — so
// over a wide row text characters walk right and overlap the
// adjacent <rect>s. `lengthAdjust="spacingAndGlyphs"` tells the
// browser it can scale both the inter-char spacing and the glyphs
// themselves; without it the attribute is a no-op on some renderers.
//
// yTop is the top y-pixel of the row; baseline is the text baseline
// (already nudged for cellH/fontPx alignment); fill is the resolved
// theme ink color used both for text and for rect fill.
func emitSVGRow(b *strings.Builder, row string, yTop, baseline int, fill string) {
	runes := []rune(row)
	i := 0
	flushText := func(start, end int) {
		if start >= end {
			return
		}
		seg := string(runes[start:end])
		if strings.TrimSpace(seg) == "" {
			return
		}
		fmt.Fprintf(b,
			`<text x="%d" y="%d" textLength="%d" lengthAdjust="spacingAndGlyphs" xml:space="preserve">%s</text>`,
			start*svgCellW, baseline, (end-start)*svgCellW, escapeXML(seg))
		b.WriteByte('\n')
	}
	textStart := 0
	for i < len(runes) {
		op, ok := svgBlockOpacity(runes[i])
		if !ok {
			i++
			continue
		}
		j := i
		for j < len(runes) {
			op2, ok2 := svgBlockOpacity(runes[j])
			if !ok2 || op2 != op {
				break
			}
			j++
		}
		flushText(textStart, i)
		width := (j - i) * svgCellW
		if op < 1 {
			fmt.Fprintf(b,
				`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" opacity="%g"/>`,
				i*svgCellW, yTop, width, svgCellH, fill, op)
		} else {
			fmt.Fprintf(b,
				`<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`,
				i*svgCellW, yTop, width, svgCellH, fill)
		}
		b.WriteByte('\n')
		i = j
		textStart = j
	}
	flushText(textStart, len(runes))
}

// svgBlockOpacity maps a Unicode block-shade glyph to the opacity its
// <rect> should render at. Returns ok=false for any other rune — that
// rune stays as text in the SVG output. Opacities mirror the Unicode
// reference ramp: `░`=25%, `▒`=50%, `▓`=75%, `█`=100%.
func svgBlockOpacity(r rune) (float64, bool) {
	switch r {
	case '█':
		return 1.0, true
	case '▓':
		return 0.75, true
	case '▒':
		return 0.5, true
	case '░':
		return 0.25, true
	}
	return 0, false
}

// escapeXML escapes the five XML special characters for safe inclusion
// in element content. Matches the minimal set used by std text/xml.
func escapeXML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}
