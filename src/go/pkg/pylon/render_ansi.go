// ANSI 24-bit color wrapping for ASCII output.
//
// Pylon's ASCII renderer emits plain monospace text by default. When
// Meta.ColorEnabled is true, semantic glyphs (candlestick bull/bear/
// doji, progress bars, warning rows) can be wrapped with 24-bit SGR
// sequences via wrapANSI. Renderers call wrapANSI unconditionally;
// colorOn=false returns the glyph unchanged so the call sites stay
// uniform.
//
// Three non-negotiables anchor the design:
//
//  1. No package-level mutable state. The color toggle lives on
//     Meta.ColorEnabled; every call site threads it through from the
//     AST root.
//  2. Auto-mode requires COLORTERM=truecolor (or =24bit). The CLI
//     resolves this; the library itself only honours ColorEnabled.
//  3. Negative integration testing (TestNoANSILeak) strips every
//     SGR sequence from colored output and asserts byte-identity
//     with plain output, so any missed rune-counter in the package
//     fails on first fixture.
package pylon

import "strings"

// semKind classifies which semantic palette a glyph belongs to. A
// zero value (semNone) means "uncolored" and passes through wrapANSI
// unchanged regardless of colorOn.
type semKind int

const (
	// semNone is the zero value. wrapANSI returns the glyph unchanged.
	semNone semKind = iota
	// semBull is the bullish candlestick / positive trend palette.
	semBull
	// semBear is the bearish candlestick / negative trend palette.
	semBear
	// semDoji is the neutral / indecision palette (yellow-ochre).
	semDoji
	// semWarn is the warning palette — bold red with a full reset
	// sequence (so the bold attribute clears along with the colour).
	semWarn
)

// SGR palette strings. Kept as package-level constants (read-only) so
// the renderers can allocate wrappers cheaply. Prefixes include the
// leading ESC `\x1b[` and trailing `m`; suffixes are the matching
// reset.
const (
	sgrBullPrefix = "\x1b[38;2;64;160;64m"
	sgrBullSuffix = "\x1b[39m"

	sgrBearPrefix = "\x1b[38;2;200;64;64m"
	sgrBearSuffix = "\x1b[39m"

	sgrDojiPrefix = "\x1b[38;2;200;170;32m"
	sgrDojiSuffix = "\x1b[39m"

	// semWarn uses a full reset (`\x1b[0m`) rather than `\x1b[39m`
	// because the prefix sets bold (SGR 1) in addition to the colour.
	sgrWarnPrefix = "\x1b[1;38;2;200;64;64m"
	sgrWarnSuffix = "\x1b[0m"
)

// wrapANSI returns glyph wrapped in the SGR sequence for kind when
// colorOn is true; otherwise returns glyph unchanged. Safe to call on
// any string; semNone returns glyph unchanged regardless of colorOn
// so callers can pass semNone for neutral glyphs without branching.
func wrapANSI(glyph string, kind semKind, colorOn bool) string {
	if !colorOn || kind == semNone {
		return glyph
	}
	switch kind {
	case semBull:
		return sgrBullPrefix + glyph + sgrBullSuffix
	case semBear:
		return sgrBearPrefix + glyph + sgrBearSuffix
	case semDoji:
		return sgrDojiPrefix + glyph + sgrDojiSuffix
	case semWarn:
		return sgrWarnPrefix + glyph + sgrWarnSuffix
	default:
		return glyph
	}
}

// stripANSI removes every `\x1b[...m` SGR sequence from s. It is the
// inverse of wrapANSI for testing: TestNoANSILeak strips the colored
// render and asserts byte-identity with the plain render. The state
// machine below mirrors skipSGR's grammar — ESC, `[`, any number of
// `0-9;`, then a final byte in a-zA-Z. Malformed sequences fall
// through and are copied literally so iteration always advances.
func stripANSI(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(runes); {
		if runes[i] == 0x1b {
			j := skipSGR(runes, i)
			if j > i {
				i = j
				continue
			}
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

// skipSGR returns the rune index after an SGR sequence starting at i,
// or i if there's no SGR at i. Safe on ragged strings: if the sequence
// is malformed (ESC with no following `[`, or `[` with no terminator
// before end-of-string, or a non-SGR final byte we accept anyway) it
// returns i unchanged so the caller can fall through to copying the
// rune verbatim and advancing by one.
//
// The general CSI grammar accepts any final byte in 0x40..0x7e; for
// SGR specifically the final byte is `m`, but the broader rule keeps
// future codes (cursor moves, private modes) from tripping up the
// width math even if a renderer emits them.
func skipSGR(runes []rune, i int) int {
	if i >= len(runes) || runes[i] != 0x1b {
		return i
	}
	// Need at least `\x1b[` and a terminator.
	if i+1 >= len(runes) || runes[i+1] != '[' {
		return i
	}
	j := i + 2
	for j < len(runes) {
		r := runes[j]
		if (r >= '0' && r <= '9') || r == ';' {
			j++
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return j + 1
		}
		// Unexpected byte mid-sequence — treat as malformed.
		return i
	}
	// Hit end-of-string without terminator — malformed.
	return i
}
