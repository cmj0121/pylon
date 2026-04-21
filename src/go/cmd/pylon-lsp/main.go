// Command pylon-lsp is the Language Server for Pylon source files.
//
// Scaffold only; glsp wiring lands in U4. Running this binary today
// prints a placeholder line to stderr and exits non-zero so it cannot
// be mistaken for a working LSP server.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "pylon-lsp: scaffold stub; server wiring lands in U4")
	os.Exit(1)
}
