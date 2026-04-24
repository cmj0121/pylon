package pylon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoANSILeak is the negative integration test demanded by
// tenth-man: for every fixture in testdata/, render twice — once
// plain, once with ColorEnabled=true — strip ANSI from the colored
// result, and assert byte-identity with the plain result. Any
// rune-counter in the package that forgets to skip SGR runs shows up
// here as a width / padding / clipping divergence on the first
// fixture that exercises the bug. Because no renderer currently calls
// wrapANSI (U2's deliverable), the stripped and plain outputs are
// trivially equal today; the test stays to catch regressions the
// moment semantic rendering lights up.
func TestNoANSILeak(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pylon") {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatalf("read fixture %s: %v", name, err)
			}
			astPlain := Parse(string(src))
			astPlain.Meta.ColorEnabled = false
			plain := RenderASCII(astPlain)

			astColor := Parse(string(src))
			astColor.Meta.ColorEnabled = true
			colored := RenderASCII(astColor)

			stripped := stripANSI(colored)
			if stripped != plain {
				t.Errorf("ANSI-stripped color output != plain output for %s\n--- plain ---\n%s\n--- stripped ---\n%s",
					name, plain, stripped)
			}
		})
	}
}

func TestDisplayWidthSkipsSGR(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"wrapped ascii", wrapANSI("hello", semBull, true), 5},
		{"wrapped warn (bold+reset)", wrapANSI("X", semWarn, true), 1},
		{"multiple wraps", wrapANSI("ab", semBull, true) + wrapANSI("cd", semBear, true), 4},
		{"cjk width", "漢字", 4},
		{"wrapped cjk", wrapANSI("漢", semDoji, true), 2},
		{"empty", "", 0},
		{"lone esc is counted", "\x1bx", 0 + 1}, // lone ESC is zero-width control, x is 1
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayWidth(tc.s); got != tc.want {
				t.Errorf("displayWidth(%q) = %d, want %d", tc.s, got, tc.want)
			}
		})
	}
}

func TestClipRowPreservesSGR(t *testing.T) {
	// Clip a colored string at a width less than its full content —
	// the SGR prefix and suffix must survive so the terminal sees the
	// matching reset.
	row := wrapANSI("abcdef", semBull, true)
	clipped := clipRow(row, 3)
	// Display width of the clipped row must be exactly 3.
	if w := displayWidth(clipped); w != 3 {
		t.Errorf("clipped displayWidth = %d, want 3", w)
	}
	// The SGR prefix must be present.
	if !strings.Contains(clipped, sgrBullPrefix) {
		t.Errorf("clipped row %q missing SGR prefix", clipped)
	}
	// Stripping ANSI from the clipped row must leave exactly 3 plain
	// content characters.
	stripped := stripANSI(clipped)
	if stripped != "abc" {
		t.Errorf("clipRow stripped = %q, want %q", stripped, "abc")
	}
}

func TestClipRowNoClipPassthrough(t *testing.T) {
	// When the row fits inside targetW, clipRow returns it unchanged.
	row := wrapANSI("ab", semBear, true)
	got := clipRow(row, 10)
	if got != row {
		t.Errorf("clipRow passthrough changed string: %q -> %q", row, got)
	}
}

func TestWrapANSIOffReturnsGlyph(t *testing.T) {
	// colorOn=false must be a no-op regardless of kind.
	kinds := []semKind{semNone, semBull, semBear, semDoji, semWarn}
	for _, k := range kinds {
		got := wrapANSI("X", k, false)
		if got != "X" {
			t.Errorf("wrapANSI(%q, %v, false) = %q, want %q", "X", k, got, "X")
		}
	}
	// semNone is always a no-op even when colorOn=true.
	if got := wrapANSI("X", semNone, true); got != "X" {
		t.Errorf("wrapANSI(X, semNone, true) = %q, want %q", got, "X")
	}
}

func TestWrapANSIOnWrapsGlyph(t *testing.T) {
	cases := []struct {
		kind   semKind
		prefix string
		suffix string
	}{
		{semBull, sgrBullPrefix, sgrBullSuffix},
		{semBear, sgrBearPrefix, sgrBearSuffix},
		{semDoji, sgrDojiPrefix, sgrDojiSuffix},
		{semWarn, sgrWarnPrefix, sgrWarnSuffix},
	}
	for _, tc := range cases {
		got := wrapANSI("G", tc.kind, true)
		want := tc.prefix + "G" + tc.suffix
		if got != want {
			t.Errorf("wrapANSI kind=%v got=%q want=%q", tc.kind, got, want)
		}
	}
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"bull", wrapANSI("X", semBull, true), "X"},
		{"bear", wrapANSI("X", semBear, true), "X"},
		{"warn full reset", wrapANSI("X", semWarn, true), "X"},
		{"mixed", "a" + wrapANSI("b", semBull, true) + "c", "abc"},
		{"empty", "", ""},
		{"lone esc passthrough", "\x1bx", "\x1bx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripANSI(tc.in); got != tc.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSkipSGRMalformed(t *testing.T) {
	// Malformed sequences must return i unchanged so iteration
	// advances by one rune at the call site.
	cases := []struct {
		name  string
		in    []rune
		i     int
		wantJ int
	}{
		{"no esc at i", []rune("abc"), 0, 0},
		{"esc no bracket", []rune("\x1bx"), 0, 0},
		{"esc bracket no terminator", []rune("\x1b[123"), 0, 0},
		{"esc bracket with m", []rune("\x1b[0m"), 0, 4},
		{"esc bracket with K", []rune("\x1b[2K"), 0, 4},
		{"out of range", []rune("abc"), 10, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skipSGR(tc.in, tc.i); got != tc.wantJ {
				t.Errorf("skipSGR(%q, %d) = %d, want %d", string(tc.in), tc.i, got, tc.wantJ)
			}
		})
	}
}
