// pylon is a CLI that reads Pylon source and emits ASCII/SVG/PNG.
//
// This commit wires the ASCII and SVG paths; the `png` format still
// prints "not yet implemented" to stderr and exits non-zero.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cmj0121/pylon/src/go/pylon"
)

// CLI is the kong-tagged struct describing the command-line surface.
type CLI struct {
	Format  string `kong:"short='f',enum='ascii,svg,png',default='ascii',help='output format'"`
	Output  string `kong:"short='o',default='-',help='output file; - is stdout'"`
	Input   string `kong:"arg,optional,type='existingfile',help='input file; stdin when omitted'"`
	Verbose int    `kong:"short='v',type='counter',help='verbose; repeat for debug (-vv)'"`
}

func main() {
	var cli CLI
	_ = kong.Parse(&cli,
		kong.Name("pylon"),
		kong.Description("Render Pylon source into ASCII/SVG/PNG."),
		kong.UsageOnError(),
	)

	setupLogging(cli.Verbose)

	src, err := readSource(cli.Input)
	if err != nil {
		log.Error().Err(err).Msg("read input")
		os.Exit(1)
	}

	switch cli.Format {
	case "ascii":
		ast := pylon.Parse(src)
		out := pylon.RenderASCII(ast) + "\n"
		if err := writeOutput(cli.Output, out); err != nil {
			log.Error().Err(err).Msg("write output")
			os.Exit(1)
		}
	case "svg":
		ast := pylon.Parse(src)
		out := pylon.RenderSVG(ast) + "\n"
		if err := writeOutput(cli.Output, out); err != nil {
			log.Error().Err(err).Msg("write output")
			os.Exit(1)
		}
	case "png":
		fmt.Fprintf(os.Stderr, "pylon: %s format not yet implemented\n", cli.Format)
		os.Exit(1)
	default:
		// kong's enum tag already guards this, but keep the catchall.
		fmt.Fprintf(os.Stderr, "pylon: unknown format %q\n", cli.Format)
		os.Exit(2)
	}
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
