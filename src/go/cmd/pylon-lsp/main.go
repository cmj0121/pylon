// Command pylon-lsp is the Language Server for Pylon source files.
//
// Speaks LSP over stdio. Logging is routed to stderr exclusively —
// stdout is the LSP transport, and any byte there corrupts the wire.
package main

import (
	"fmt"
	"os"

	"github.com/cmj0121/pylon/src/go/internal/lsp"
)

// Version is the build-time version string. Populated by the Makefile
// via `-ldflags="-X main.Version=..."`; see the matching declaration
// in cmd/pylon/main.go for the source-of-truth rationale.
var Version = "dev"

func main() {
	// --version is the only CLI surface the LSP exposes; handle it
	// manually rather than pulling in kong. Anything else goes
	// straight to the stdio LSP loop so the transport contract
	// (stdout = LSP traffic only) stays intact.
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-V" {
			fmt.Println("pylon-lsp version " + Version)
			return
		}
	}

	if err := lsp.Run(); err != nil {
		// stderr, not stdout — stdout is reserved for LSP traffic.
		_, _ = os.Stderr.WriteString("pylon-lsp: " + err.Error() + "\n")
		os.Exit(1)
	}
}
