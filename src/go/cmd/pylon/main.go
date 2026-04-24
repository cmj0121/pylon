// pylon is a CLI that reads Pylon source and emits ASCII/SVG/PNG.
//
// All three formats are wired: ASCII and SVG are text; PNG is a
// binary bitmap rendered via an embedded JetBrains Mono font.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cmj0121/pylon/src/go/pkg/pylon"
)

// Version is the build-time version string. The Makefile injects the
// output of `git describe --tags --always --dirty` via
// `-ldflags="-X main.Version=..."`; go-install users without the
// Makefile (and the release pipeline is not yet in place) get the
// literal default "dev".
var Version = "dev"

// CLI is the kong-tagged struct describing the command-line surface.
type CLI struct {
	Format  string           `kong:"short='f',enum='ascii,svg,png',default='ascii',help='output format'"`
	Output  string           `kong:"short='o',default='-',help='output file; - is stdout'"`
	Input   string           `kong:"arg,optional,type='existingfile',help='input file; stdin when omitted'"`
	Verbose int              `kong:"short='v',type='counter',help='verbose; repeat for debug (-vv)'"`
	Strict  bool             `kong:"help='exit non-zero when any diagnostic is emitted (for CI gating)'"`
	Color   string           `kong:"enum='auto,always,never',default='auto',help='when to emit ANSI color escapes'"`
	Version kong.VersionFlag `kong:"help='show version and exit'"`
}

func main() {
	var cli CLI
	_ = kong.Parse(&cli,
		kong.Name("pylon"),
		kong.Description("Render Pylon source into ASCII/SVG/PNG."),
		kong.UsageOnError(),
		kong.Vars{"version": "pylon version " + Version},
	)

	setupLogging(cli.Verbose)

	src, err := readSource(cli.Input)
	if err != nil {
		log.Error().Err(err).Msg("read input")
		os.Exit(1)
	}

	ast := pylon.Parse(src)
	ast.Meta.ColorEnabled = resolveColor(cli.Color)
	diags := pylon.Validate(ast)
	for _, d := range diags {
		emitDiagnostic(os.Stderr, displayPath(cli.Input), d)
	}

	switch cli.Format {
	case "ascii":
		out := pylon.RenderASCII(ast) + "\n"
		if err := writeOutput(cli.Output, out); err != nil {
			log.Error().Err(err).Msg("write output")
			os.Exit(1)
		}
	case "svg":
		out := pylon.RenderSVG(ast) + "\n"
		if err := writeOutput(cli.Output, out); err != nil {
			log.Error().Err(err).Msg("write output")
			os.Exit(1)
		}
	case "png":
		out, err := pylon.RenderPNG(ast)
		if err != nil {
			log.Error().Err(err).Msg("render png")
			os.Exit(1)
		}
		if err := writeOutputBytes(cli.Output, out); err != nil {
			log.Error().Err(err).Msg("write output")
			os.Exit(1)
		}
	default:
		// kong's enum tag already guards this, but keep the catchall.
		fmt.Fprintf(os.Stderr, "pylon: unknown format %q\n", cli.Format)
		os.Exit(2)
	}

	// Strict-mode gating: rendering already completed so users see
	// their output; we just translate "any diagnostic" into a non-zero
	// exit for CI pipelines.
	if cli.Strict && len(diags) > 0 {
		os.Exit(2)
	}
}

// displayPath returns the label CLI diagnostics print for input
// location. Matches the stdin convention of other Unix tools: `-`
// when the source came from stdin, otherwise the provided path.
func displayPath(input string) string {
	if input == "" {
		return "-"
	}
	return input
}

// emitDiagnostic writes one diagnostic to w in a human-readable
// one-line format:
//
//	[CODE] PATH:LINE:COL: MESSAGE
//
// Line and column are 1-based for CLI output (editor / compiler
// convention). LSP ranges stay 0-based on the protocol wire; the
// conversion happens here, not in Validate.
func emitDiagnostic(w io.Writer, path string, d pylon.Diagnostic) {
	fmt.Fprintf(w, "[%s] %s:%d:%d: %s\n",
		d.Code, path, d.Span.Start.Line+1, d.Span.Start.Column+1, d.Message)
}

func setupLogging(verbose int) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch {
	case verbose >= 2:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case verbose == 1:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
}

func readSource(path string) (string, error) {
	if path == "" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func writeOutput(path, payload string) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.WriteString(payload)
		return err
	}
	return os.WriteFile(path, []byte(payload), 0o644)
}

// writeOutputBytes is the binary-safe variant used for PNG output.
// It writes raw bytes without any text-mode transformation, so PNG
// headers / compressed streams reach stdout unmodified.
func writeOutputBytes(path string, payload []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(payload)
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

// resolveColor converts the `--color=auto|always|never` flag into the
// boolean that ends up on Meta.ColorEnabled. Auto-mode is
// intentionally conservative — it only returns true when:
//   - NO_COLOR is unset (https://no-color.org),
//   - COLORTERM is set to `truecolor` or `24bit` (terminals that
//     declare 24-bit support — otherwise our SGR bytes show up as
//     literal garbage in terminals that don't parse them),
//   - stdout is a character device (pipes / files get no colour).
//
// `always` and `never` force the flag regardless of environment so
// CI and scripted callers have a deterministic switch.
func resolveColor(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		ct := os.Getenv("COLORTERM")
		if ct != "truecolor" && ct != "24bit" {
			return false
		}
		stat, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return stat.Mode()&os.ModeCharDevice != 0
	}
}
