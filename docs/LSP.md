# Pylon Language Server

Reference for `pylon-lsp`, the Go-based Language Server that ships
alongside the CLI. Editors running an LSP client get inline
diagnostics, a document-symbol outline, and partial semantic
highlighting from the same parser and diagnostic pipeline the CLI
uses. Companion to [`SPEC.md`](SPEC.md) (grammar + error model) and
[`../README.md`](../README.md) (front door).

## Install

Pre-built `pylon-lsp` binaries for linux (amd64, arm64) and macOS
(Intel, Apple Silicon) ship alongside the `pylon` CLI on each
release — download from
<https://github.com/cmj0121/pylon/releases> and drop the binary
onto your `$PATH`. On other platforms, build from source:

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon-lsp@latest
```

Requires Go 1.25+. The binary lands in `$GOPATH/bin` (typically
`~/go/bin`); make sure that directory is on your `$PATH`. Confirm
the install with:

```sh
pylon-lsp < /dev/null
pylon-lsp --version
```

A correctly-installed server exits with status 0 and writes zero
bytes to stdout or stderr when stdin is closed — stdout is reserved
for the LSP protocol. `--version` (or `-V`) prints
`pylon-lsp version <X>` and exits 0; the Makefile populates `<X>`
via `git describe --tags --always --dirty`, and `go install` users
without the Makefile get the literal `dev` fallback.

## Editor setup

Neovim setup — both `nvim-lspconfig` and plain
`vim.lsp.start` paths — is documented in
[`../contrib/vim/README.md`](../contrib/vim/README.md) under the
"Language Server" section. The bundled Lua module at
[`../contrib/vim/lua/pylon/lsp.lua`](../contrib/vim/lua/pylon/lsp.lua)
is load-safe: it emits a warning and returns cleanly when
nvim-lspconfig is absent, so the same init snippet works before and
after it's installed.

VS Code and Zed packaging land in the follow-up branch
`feat/pylon-lsp-ux`.

## Features

Transport: stdio. Sync: full-text (`TextDocumentSyncKind.Full`).

### Diagnostics

Push-style via `textDocument/publishDiagnostics`. The server
re-validates on every `didOpen` and `didChange`, and publishes an
empty array on `didClose` to clear client-side markers. Each
diagnostic carries:

- `code` — the stable [`pylon.Code`](../src/go/pkg/pylon/diagnostic.go)
  identifier (e.g., `"ref.undefined"`).
- `severity` — `Error` for toast-path (whole-source invalid) or
  `Warning` for inline `⚠` render-time classes.
- `source` — always `"pylon-lsp"` so editors can filter against
  other servers attached to the same buffer.
- `range` — the offending token, name, or frontmatter section, with
  0-based line and UTF-16 column offsets.
- `message` — byte-identical to the JS reference implementation;
  the CI `parity-diagnostics` gate locks this in.

See [Error model mapping](#error-model-mapping) for the full catalogue.

### Document symbols

`textDocument/documentSymbol` returns a hierarchical outline:

- Every `[ label :: name ]` declaration surfaces as a
  `SymbolKind.Variable` named `name` (bare; no `&` prefix). Range
  covers the full bracketed node.
- Each frontmatter `data:` series surfaces as a
  `SymbolKind.Constant` — `@data` for a flat list, one `@<key>`
  per entry for a map-keyed block.

v1 limitation: data-series symbols share the whole `data:`
section's range because the parser does not track per-key spans
today. Per-key narrowing will land alongside the deferred semantic
tokens in `feat/pylon-lsp-ux`.

### Semantic tokens

`textDocument/semanticTokens/full` is **partial** in this release.
The server emits:

- `[`, `]`, `(`, `)` — `operator`.
- `&ref` — `variable`.
- `@ref` — `variable` with the `readonly` modifier.

Deferred to `feat/pylon-lsp-ux` (the existing Vim `syntax/pylon.vim`
regex plugin remains the fallback for these classes — nothing to
disable):

- frontmatter key / number / string tokens;
- edge arrows `->`, `<-`, `<->`, `-->`, `<-->`, etc.;
- alignment marker dashes flush with a bracket;
- `:: name` declaration identifiers;
- `| renderer` pipe tags;
- the `---` frontmatter fence lines.

The legend published in `initialize` is add-only — a locked test
(`TestLegendAddOnlyInvariant`) asserts that editors which cached the
legend from a previous response stay correct across upgrades.

## Error-model mapping

Every diagnostic the server emits maps to one of 11 stable codes.
Wording is byte-identical to [`src/js/pylon.js`](../src/js/pylon.js)
so the `parity-diagnostics` CI gate can diff outputs directly.

Toast-path (whole source invalid, severity Error):

| `Code`             | Message                               | Trigger                                     |
| ------------------ | ------------------------------------- | ------------------------------------------- |
| `data.unsupported` | `Unsupported data: frontmatter shape` | Tab indent, flow style, or other non-subset |
| `ref.duplicate`    | `Duplicate node name: NAME`           | Two `:: NAME` declarations collide          |
| `ref.undefined`    | `Undefined ref: &NAME`                | `&NAME` with no matching declaration        |

Inline render-time (one box, severity Warning). Every wire message
in this group carries a leading warning-triangle glyph (U+26A0) and
a space; the `Message` column below lists the portion after that
prefix, so wire text for the first row is literally
`"⚠ unknown renderer: NAME"`.

| `Code`                   | Message                       | Trigger                                     |
| ------------------------ | ----------------------------- | ------------------------------------------- |
| `renderer.unknown`       | `unknown renderer: NAME`      | `\| NAME` not in `text`/`hbar`/`vbar`/`bar` |
| `renderer.use_at_ref`    | `NAME: use @ref`              | Non-`text` renderer handed raw text         |
| `renderer.bare_data_ref` | `@NAME: requires \| renderer` | `@NAME` in box body with no `\| renderer`   |
| `data.not_found`         | `@NAME not found`             | `@NAME` unresolved against `data:`          |
| `bar.shape`              | `NAME: expected [{x,y}]`      | Bar-family series isn't `[{x,y}]`           |
| `bar.empty`              | `NAME: empty series`          | Bar-family series has zero entries          |
| `bar.negative_y`         | `NAME: negative y`            | Bar-family entry has `y < 0`                |
| `bar.duplicate_x`        | `NAME: duplicate x "X"`       | Two bar-family entries share `x`            |

The `bar` alias dispatches through the horizontal-bar renderer but
threads the literal renderer name into diagnostics — `[ hello | bar ]`
surfaces `renderer.use_at_ref` with `bar:` prefix, and a negative
`y` under `| bar` surfaces `bar.negative_y` likewise. Only `hbar`
and `vbar` produce `hbar:` / `vbar:` prefixes. Clients are free to
strip the leading warning glyph in their display layer.

## CLI diagnostic mode

The `pylon` CLI consumes the same [`pylon.Validate()`](../src/go/pkg/pylon/validate.go)
pass as the server, printing diagnostics to stderr after parsing:

```txt
[CODE] PATH:LINE:COL: MESSAGE
```

`PATH` is the positional input path, or `-` for stdin. `LINE` and
`COL` are 1-based (editor / compiler convention; the LSP wire stays
0-based). Rendering proceeds regardless of severity so users see
their output alongside any errors. The default exit code is `0`;
pass `--strict` to exit `2` when any diagnostic is emitted, which
lets CI pipelines gate a build on Pylon lint errors without parsing
stderr.

```sh
$ echo '[ foo | unknown ]' | pylon >/dev/null
[renderer.unknown] -:1:1: ⚠ unknown renderer: unknown

$ echo '[ foo | unknown ]' | pylon --strict >/dev/null; echo "exit=$?"
[renderer.unknown] -:1:1: ⚠ unknown renderer: unknown
exit=2
```

## Install-path change

Two top-level install targets replace the old flat layout:

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon@latest       # CLI
go install github.com/cmj0121/pylon/src/go/cmd/pylon-lsp@latest   # Language Server
```

The previous command `go install github.com/cmj0121/pylon/src/go@latest`
stops working with this release. Besides splitting the tree to
accommodate the server, the new layout fixes a pre-existing misname:
Go derives the binary name from the last path segment, so the old
command produced a binary called `go`, not `pylon`. Users who had
the old path pinned in scripts should update to the new `/cmd/pylon`
target.

## Troubleshooting

- **`pylon-lsp: command not found`.** The binary went to
  `$GOPATH/bin` (or `~/go/bin` when `GOPATH` is unset); make sure
  that directory is on your `$PATH`.
- **Diagnostics not appearing.** Confirm the client attached: the
  editor's LSP log should show an `initialize` → `initialized`
  handshake. Then verify the server itself is healthy with
  `pylon-lsp < /dev/null` — exit 0 and zero bytes on stdout mean
  the transport is intact.
- **Document-symbol ranges for `@counter` and `@sales` are
  identical.** Known v1 limitation — data-series symbols share the
  parent `data:` section span until the parser grows per-key spans
  in `feat/pylon-lsp-ux`.
- **Editor shows `⚠` as part of the diagnostic text.** That's the
  wire format. Clients are free to strip the prefix in their
  display layer; pylon-lsp emits it so the `parity-diagnostics` CI
  gate can byte-compare against the JS reference.

## Roadmap

`feat/pylon-lsp-ux` follows this branch and picks up the features
left out of v1: VS Code and Zed packaging, the deferred semantic
token classes (edges, frontmatter keys, `:: name`, `| renderer`,
align markers, fence lines), per-series-key symbol spans, and the
navigation surface — hover, go-to-definition, completion, and
rename across `&ref` / `@ref` / `:: name`. The UX branch also owns
a `--strict` CLI flag that would turn diagnostics into a non-zero
exit status.
