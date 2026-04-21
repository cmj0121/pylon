// Command pylon-dump-diagnostics dumps Validate() output as canonical
// JSON for the cross-implementation diagnostic-set parity gate.
//
// Reads a single .pylon path (or `-` for stdin), runs Parse +
// Validate, partitions the diagnostics by Severity into `errors`
// (toast-path) and `inlineWarnings` (render-time `⚠`), sorts each
// array lexicographically, and emits pretty-printed JSON on stdout.
// Exit code stays 0 even when the source contains diagnostics — this
// is a dump tool, not a lint.
//
// Paired with scripts/pylon-dump-diagnostics.mjs for the JS side;
// scripts/pylon-diagnostics-parity.sh diffs the two outputs.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

type dump struct {
	Errors         []string `json:"errors"`
	InlineWarnings []string `json:"inlineWarnings"`
}

func main() {
	src, err := readSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pylon-dump-diagnostics: %v\n", err)
		os.Exit(1)
	}

	ast := pylon.Parse(string(src))
	diags := pylon.Validate(ast)

	out := dump{Errors: []string{}, InlineWarnings: []string{}}
	for _, d := range diags {
		switch d.Severity {
		case pylon.SeverityError:
			out.Errors = append(out.Errors, d.Message)
		case pylon.SeverityWarning:
			out.InlineWarnings = append(out.InlineWarnings, d.Message)
		}
	}
	sort.Strings(out.Errors)
	sort.Strings(out.InlineWarnings)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	// Node's JSON.stringify does not escape `&`, `<`, `>` — the JS
	// dumper produces raw characters. Match that convention so the
	// parity gate can byte-compare outputs without normalisation.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "pylon-dump-diagnostics: %v\n", err)
		os.Exit(1)
	}
}

func readSource() ([]byte, error) {
	if len(os.Args) < 2 || os.Args[1] == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(os.Args[1])
}
