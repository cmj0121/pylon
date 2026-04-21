package lsp

import (
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

// TestServerStdoutGuard asserts the server writes ZERO bytes to stdout
// when started with a closed stdin. LSP uses stdout as the transport;
// any non-protocol byte there (a stray log line, a panic trace, etc.)
// corrupts the wire format for every client.
//
// glsp's RunStdio reads from os.Stdin and writes to os.Stdout, so the
// guard has to redirect the process-global os.Stdin / os.Stdout for
// the duration of Run. This test does NOT call t.Parallel — concurrent
// redirection would clobber other tests.
func TestServerStdoutGuard(t *testing.T) {
	origStdin, origStdout, origStderr := os.Stdin, os.Stdout, os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	// Closed stdin: server reads EOF immediately and exits.
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdin: %v", err)
	}
	if err := stdinW.Close(); err != nil {
		t.Fatalf("close stdin write: %v", err)
	}
	os.Stdin = stdinR

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	os.Stdout = stdoutW

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	os.Stderr = stderrW

	// Drain stderr concurrently so zerolog writes don't block the
	// pipe buffer.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.Discard, stderrR)
	}()

	done := make(chan error, 1)
	go func() {
		done <- Run()
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return within 3s of stdin EOF")
	}

	// Close the write ends so the read ends see EOF.
	_ = stdoutW.Close()
	_ = stderrW.Close()

	stdoutBytes, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	if len(stdoutBytes) > 0 {
		t.Errorf("stdout emitted %d byte(s) before protocol traffic: %q", len(stdoutBytes), stdoutBytes)
	}

	_ = stdinR.Close()
	_ = stdoutR.Close()
	wg.Wait()
}
