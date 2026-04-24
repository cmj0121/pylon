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
		fmt.Fprintf(&b,
			`<text x="0" y="%d" xml:space="preserve">%s</text>`,
			baseline, escapeXML(row))
		b.WriteByte('\n')
	}
	b.WriteString(`</svg>`)
	return b.String()
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
