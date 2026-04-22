# pylon (Go CLI)

A local command-line renderer for [Pylon](../../README.md) sources. Reads a `.pylon` file
(positional argument or stdin) and emits **ASCII**, **SVG**, or **PNG** to a file or stdout.
The parser and renderers mirror the JS library under [`../js/`](../js/); a parity gate keeps
ASCII output byte-identical across the two implementations.

## Install

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon@latest
```

The module path ends in `/src/go` — the Go module lives in this subdirectory,
and each binary is a separate `cmd/` package. The earlier path
`github.com/cmj0121/pylon/src/go@latest` produced a binary named `go` (Go
derives the binary name from the last path component); the `cmd/pylon` path
fixes that and yields the expected `pylon` binary.

A Language Server binary ships at `cmd/pylon-lsp`:

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon-lsp@latest
```

The scaffold lands in this branch; wired server behaviour arrives with U4.

## Usage

```sh
# ASCII from stdin:
echo '[ Start ] -> [ End ]' | pylon

# SVG to a file:
pylon -f svg -o diagram.svg diagram.pylon

# PNG to a file:
pylon -f png -o diagram.png diagram.pylon
```

Flags: `-f ascii|svg|png` (default `ascii`), `-o PATH` (default `-` / stdout), `-v` / `-vv`
for info / debug logging on stderr. A positional input path takes precedence over stdin.

## Development

```sh
make -C src/go build        # build ./dist/pylon
make -C src/go test         # run `go test ./...`
make -C src/go install      # `go install` into $GOBIN or $GOPATH/bin
make -C src/go parity       # byte-compare ASCII Go vs JS on shared fixtures
```

The parity gate requires both binaries built: `make -C src/go build` and
`make -C src/js build` (which produces `dist/pylon.min.js`). It iterates every
`.pylon` fixture under `pkg/pylon/testdata/` and diffs the ASCII output of
`dist/pylon -f ascii FIXTURE` against the Node shim in
[`../../scripts/pylon-render-js.mjs`](../../scripts/pylon-render-js.mjs).

### Adding a golden fixture

1. Drop `NAME.pylon` (the source) into [`pkg/pylon/testdata/`](pkg/pylon/testdata/).
2. Add the matching `NAME.ascii` (and optionally `NAME.svg`) produced by the **current** Go
   build, so that tests and the parity gate both have a golden to compare against.
3. `make -C src/go test` should pass locally; `make -C src/go parity` confirms the JS
   library agrees on the ASCII output.

Keep fixtures small and single-purpose — one feature per file, mirroring the convention
under [`../../examples/`](../../examples/).

### Regenerating goldens

The `.ascii` files under `pkg/pylon/testdata/` are authored to match the JS reference
renderer, so the canonical regen command routes through the Node shim rather than the Go
binary:

```sh
node ../../scripts/pylon-render-js.mjs pkg/pylon/testdata/NAME.pylon > pkg/pylon/testdata/NAME.ascii
```

The repo-root [`.pre-commit-config.yaml`](../../.pre-commit-config.yaml) excludes
`pkg/pylon/testdata/*.ascii` from the end-of-file-fixer and trailing-whitespace hooks, so
the golden keeps the exact bytes the renderer emits: no trailing newline, and trailing
spaces on padded chart rows are preserved. Don't hand-edit these files — if a golden looks
wrong, regenerate it and let `make -C src/go parity` confirm Go still agrees with JS.

## Known limitations

- **CJK text shaping in PNG.** The embedded font is JetBrains Mono Regular; glyphs outside
  its coverage (CJK, emoji) fall back to tofu or the replacement glyph.
- **Per-cell SVG tspan.** SVG output emits one `<tspan>` per character cell for monospace
  fidelity; this is verbose compared to a single-run text node and is intentional — it
  keeps cells addressable and column-aligned across renderers.

## License

The CLI source inherits the repository's top-level license. The embedded TTF —
[`pkg/pylon/assets/JetBrainsMono-Regular.ttf`](pkg/pylon/assets/JetBrainsMono-Regular.ttf) — ships
under the **SIL Open Font License 1.1**; the full text is in
[`pkg/pylon/assets/JetBrainsMono-OFL.txt`](pkg/pylon/assets/JetBrainsMono-OFL.txt).
