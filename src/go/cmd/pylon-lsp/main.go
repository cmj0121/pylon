// Command pylon-lsp is the Language Server for Pylon source files.
//
// Speaks LSP over stdio. Logging is routed to stderr exclusively —
// stdout is the LSP transport, and any byte there corrupts the wire.
package main

import (
	"os"

	"github.com/cmj0121/pylon/src/go/internal/lsp"
)

func main() {
	if err := lsp.Run(); err != nil {
		// stderr, not stdout — stdout is reserved for LSP traffic.
		_, _ = os.Stderr.WriteString("pylon-lsp: " + err.Error() + "\n")
		os.Exit(1)
	}
}
