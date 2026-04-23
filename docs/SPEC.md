# Pylon Specification

Canonical reference for the Pylon source language, renderer
catalogue, error model, and CLI conventions. Companion to
[`README.md`](../README.md): the README is the onboarding front door,
this document is the lookup reference.

A Pylon source file is an optional YAML-subset frontmatter block
followed by a body of nodes, edges, references, and renderer pipelines.

## Browser integration

The `<pylon-chart>` custom element accepts three attributes:

- `format="ascii|svg|png"` — default `ascii`. Selects the rendering
  backend.
- `src="source text"` — inline Pylon source. Alternative to child text
  content.
- `wysiwyg` — flag. When present, the element becomes a split-pane
  editor with a live preview and Copy / Download buttons for the
  current format.

Every color, spacing, font, and shape is themeable via `--pylon-*`
CSS custom properties in [`../src/js/pylon.css`](../src/js/pylon.css).
Adopters re-skin by overriding those custom properties in their own
stylesheet.

## Grammar overview

| Primitive              | Syntax                              | Section                                              |
| ---------------------- | ----------------------------------- | ---------------------------------------------------- |
| Frontmatter            | `--- ... ---`                       | [Frontmatter](#frontmatter)                          |
| Nodes (bordered / not) | `[ LABEL ]`, `( LABEL )`, `[- X -]` | [Nodes](#nodes)                                      |
| Edges                  | `->`, `<-`, `<->`, `-- ( L ) -->`   | [Edges](#edges)                                      |
| Names and refs         | `[ LABEL :: NAME ]`, `&NAME`        | [Named nodes and refs](#named-nodes-and-references)  |
| Data and renderers     | `@NAME`, `\| NAME`                  | [Data and renderers](#data-references-and-renderers) |

## Frontmatter

A source file may begin with a YAML-subset frontmatter block fenced by
`---` on its own line. All keys are optional.

```pylon
---
size: 30x5
theme: simple
data:
  - x: 1
    y: 10
---
[- Hello -]
```

### Keys

| Key     | Value type  | Description                                                              |
| ------- | ----------- | ------------------------------------------------------------------------ |
| `size`  | `INT x INT` | Maximum outer width x height in cells. Content stays tight when smaller. |
| `theme` | identifier  | Palette / glyph set: `simple` (default), `ascii`, `dark`, `light`.       |
| `data`  | list or map | One or more `{x, y}` series for `@ref` / renderer boxes.                 |

`size` caps the outer dimensions; flow chains that would exceed the
declared width wrap to a vertical stack. `theme: ascii` swaps the
Unicode box-drawing glyphs for plain `+ - | < >`.

### YAML subset accepted by `data:`

The `data:` parser is a narrow YAML subset, not full YAML.

- **Block style only.** Flow style (`[...]`, `{...}`) is rejected.
- **Spaces-only indent.** A tab anywhere inside the frontmatter
  rejects the whole block via toast.
- **Scalars.** Numbers, double-quoted strings, or unquoted strings
  (trimmed, taken literally).
- **No reserved-word coercion.** `true`, `false`, `null`, `~`, `yes`,
  `no` stay literal strings.
- **Not supported.** Multiline scalars (`|`, `>`), anchors (`&`, `*`),
  and tags (`!!`).

Unsupported shapes leave `data` unset and surface an
`Unsupported data: frontmatter shape` toast.

### Data shapes

Two shapes are accepted. A flat list is reachable as `@data`; a map
exposes each key as its own reference (`@counter`, `@sales`):

```pylon
---
data:
  counter:
    - x: 1
      y: 10
  sales:
    - x: 1
      y: 100
---
[ @sales | bar ]
```

Unquoted string `x` labels are supported — categorical charts are the
friendly default.

## Nodes

Nodes are the drawn units of a diagram. A node body may contain text
lines, nested nodes, or a flow chain; lines stack vertically.

- `[ LABEL ]` — bordered; the wrapper is drawn.
- `( LABEL )` — borderless; wrapper drops, children flow into the
  parent layout.

### Alignment

The alignment dash must be flush with the bracket; whitespace between
the dash and the bracket defeats it.

| Form      | Result                                            |
| --------- | ------------------------------------------------- |
| `[ x ]`   | Centered (default).                               |
| `[- x -]` | Centered (explicit both sides).                   |
| `[- x ]`  | Right-aligned (left spring pushes content right). |
| `[ x -]`  | Left-aligned.                                     |

Multiple top-level nodes stack vertically:

```pylon
[Hello]
(World)
```

```txt
   ┌───────────┐
   │   Hello   │
   └───────────┘
       World
```

## Edges

Edges are drawn between sibling nodes on a single line (a flow
chain). Supported forms: `->` (right), `<-` (left), `<->`
(bidirectional), and longer variants `-->`, `--->`, … — any
`<?-+>?` token with at least one arrow glyph is accepted; length
only changes visual spacing.

### Labelled edges

A borderless node between two edge halves labels the edge:

```pylon
[ Alice ] -- ( friend ) --> [ Bob ]
```

```txt
┌───────────┐               ┌─────────┐
│   Alice   │──  friend  ──▶│   Bob   │
└───────────┘               └─────────┘
```

Bidirectional `<-- ( role ) -->` and reverse `<-- ( role ) --` both
work.

## Named nodes and references

A node declaration can be tagged so later syntax can reference it.
Names live in the `&` namespace; they are independent from the `@`
data-ref namespace.

### Declaration

`[ LABEL :: NAME ]` declares a node named `NAME`. The trailing `::` +
identifier is stripped from the rendered label and attached to the
AST. Borderless `( ... :: NAME )` works the same way.

### Reference

`&NAME` references a previously declared node. References are
pointers, not copies — the declaration is the only place the full box
is drawn. How a reference renders depends on where the declaration
lives relative to the reference.

**Same row — self-loop arc.** Declaration and reference on the same
line with the declaration adjacent:

```pylon
[- a :: x -] -> &x
```

```txt
┌───────┐
│   a   │
└───────┘
  │   ▲
  └───┘
```

**Different row — gutter arrow.** Declaration on one line, reference
on a later line:

```pylon
[ a :: x ]
[ c ] -> &x
```

```txt
   ┌───────┐
   │   a   │◀────┐
   └───────┘     │
   ┌───────┐     │
   │   c   │─────┘
   └───────┘
```

**Fallback — literal text.** Anywhere else (nested refs, a second ref
to an already gutter-routed target, etc.), the reference falls back
to the plain name as inline text.

Duplicate declarations and unresolved references surface as a toast
inside the `<pylon-chart>` element — see the
[error model](#error-model).

## Data references and renderers

Boxes can render data instead of literal text. Series declared under
the `data:` frontmatter key are resolved via `@NAME`, and a trailing
`| NAME` pipe routes the box body through a renderer.

`@` references data (series); `&` references nodes. The two
namespaces do not overlap.

### `@ref` data references

`@<name>` inside a box body resolves to the named series at render
time. The sigil is recognised only at a box-body boundary: `@` must
follow `[`, `(`, whitespace, or start of string, and be followed by
an identifier then whitespace / `|` / `]` / `)` / end of string.
Anything else stays literal label text — `[ user@example.com ]` and
mid-word `foo@bar` do not resolve.

### `| renderer` pipe

A trailing `| <name>` inside a box marks the renderer. Default is
`text` when omitted. Whitespace around `|` is required, so literal
`|` in labels still works.

| Form                       | Behaviour                                          |
| -------------------------- | -------------------------------------------------- |
| `[ label ]`                | `text` renderer; label rendered as-is.             |
| `[ @counter \| bar ]`      | Pipes the resolved series to `bar`.                |
| `[ @data \| bar \| text ]` | First pipe wins (`bar`); `text` consumed silently. |
| `[ data \| bar \| 123 ]`   | Malformed tail; whole `\| …` left as literal text. |

Chains (`| a | b | c`) extract only when every segment is a valid
identifier up to the closing bracket, first renderer wins, the rest
are consumed silently. A non-identifier tail falls back to literal
label text.

### Composition

`| renderer` composes with `::` naming, alignment dashes, and
borderless `(...)`. Order inside the brackets is
`<dashes>? <content> :: <name>? | <renderer>? <dashes>?`.

| Form                            | Meaning                                                      |
| ------------------------------- | ------------------------------------------------------------ |
| `[- @counter \| bar -]`         | Centered chart.                                              |
| `[ @counter \| bar :: chart1 ]` | Named chart (tag before the pipe is fine too).               |
| `( @counter \| bar )`           | Borderless chart (wrapper drops; bars flow into the parent). |

A bare `@ref` with no renderer surfaces its own inline error
(`@<name>: requires a renderer`) so the omission is loud rather than
silent.

## Available renderers

Pylon 0.2.0 ships four renderers. Unknown names surface an inline
`⚠ unknown renderer: NAME`.

| Renderer       | Input              | Output                                                                   |
| -------------- | ------------------ | ------------------------------------------------------------------------ |
| `text`         | string or `@ref`   | Pass-through; `@ref` is emitted via `JSON.stringify`.                    |
| `hbar` (`bar`) | `@ref` → `[{x,y}]` | Horizontal bars scaled against `max(y)` with `(value)` labels.           |
| `vbar`         | `@ref` → `[{x,y}]` | Vertical bars scaled against `max(y)`; `x` and `(value)` labels as feet. |
| `banner`       | literal string     | 6-row block-letter banner of the uppercased source text.                 |

`bar` is a v0.1 alias for `hbar`; both render identically. `banner`
is Go-only in v1 — the JS renderer surfaces `⚠ unknown renderer:
banner` until parity lands. See
[`ROADMAP.md`](ROADMAP.md).

### `text`

```pylon
---
data:
  - x: 1
    y: 10
  - x: 2
    y: 20
---
[- @data | text -]
```

```txt
┌─────────────────────────────────────┐
│   [{"x":1,"y":10},{"x":2,"y":20}]   │
└─────────────────────────────────────┘
```

### `hbar` / `bar`

```pylon
---
data:
  - x: 1
    y: 10
  - x: 2
    y: 20
  - x: 3
    y: 15
---
[ @data | bar ]
```

```txt
┌───────────────────────────┐
│   1 │ █████      (10) │   │
│   2 │ ██████████ (20) │   │
│   3 │ ████████   (15) │   │
└───────────────────────────┘
```

`hbar` uses a 10-cell bar-width budget by default. A tight `size:`
clips the bar first, then truncates the `(value)` label. `hbar` only
accepts an `@ref`; a raw string surfaces `⚠ bar: use @ref` (or
`hbar:` when invoked as `hbar`).

### `vbar`

```pylon
---
data:
  - x: 1
    y: 10
  - x: 2
    y: 20
  - x: 3
    y: 15
---
[ @data | vbar ]
```

```txt
┌──────────────────┐
│        █         │
│        █         │
│        █   █     │
│        █   █     │
│        █   █     │
│    █   █   █     │
│    █   █   █     │
│    █   █   █     │
│    █   █   █     │
│    █   █   █     │
│    1   2   3     │
│   (10)(20)(15)   │
└──────────────────┘
```

`vbar` mirrors `hbar`'s 10-cell budget but applies it to the chart
HEIGHT. Each series entry becomes a single-column bar; column width
is `max(|label|, |(value)|, 3)`. The two footer rows carry the `x`
label and `(value)` text. All-zero `y` renders zero-height bars — not
an error.

### banner

```pylon
[ Pylon | banner ]
```

```txt
┌──────────────────────────────────────────────────┐
│   ██████╗ ██╗   ██╗██╗      ██████╗ ███╗   ██╗   │
│   ██╔══██╗╚██╗ ██╔╝██║     ██╔═══██╗████╗  ██║   │
│   ██████╔╝ ╚████╔╝ ██║     ██║   ██║██╔██╗ ██║   │
│   ██╔═══╝   ╚██╔╝  ██║     ██║   ██║██║╚██╗██║   │
│   ██║        ██║   ███████╗╚██████╔╝██║ ╚████║   │
│   ╚═╝        ╚═╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═══╝   │
└──────────────────────────────────────────────────┘
```

`banner` renders the box's literal text as 6-row block letters.
Input is uppercased before lookup; the supported set is `A-Z`,
`0-9`, space, and `. , ! ? - ' "`. Unknown runes fall back to the
`?` glyph. `theme: ascii` swaps the ANSI-shadow box-drawing glyphs
for a `#` + space grid.

v1 is literal-string only. A `@ref` inside a `| banner` box is
silently ignored in Go; the JS renderer has no `banner` at all in
v1 and surfaces `⚠ unknown renderer: banner` instead. Parity is
tracked in [`ROADMAP.md`](ROADMAP.md).

## Error model

Pylon surfaces errors through two channels.

- **Toast.** Parse-shape errors that invalidate the whole source
  (malformed `data:`, tab indent, duplicate name, unresolved
  reference). Rendered inside the `<pylon-chart>` element, not via
  `alert()`.
- **Inline `⚠`.** Render-time errors confined to one box (bad
  renderer, missing ref, wrong shape). Rendered as the first line of
  the offending box. Long messages are clipped by the box width, same
  as any other content.

### Render-time error catalogue

| Trigger                            | Inline message               |
| ---------------------------------- | ---------------------------- |
| Raw string piped to non-`text`     | `bar: use @ref`              |
| Unknown renderer name              | `unknown renderer: foo`      |
| `@ref` not declared in frontmatter | `@missing not found`         |
| Bare `@ref` with no renderer       | `@name: requires a renderer` |
| Renderer got wrong shape           | `hbar: expected [{x,y}]`     |
| Empty series for a bar renderer    | `hbar: empty series`         |
| Any negative `y`                   | `hbar: negative y`           |
| Duplicate `x` in series            | `hbar: duplicate x "a"`      |

Bar-family errors (shape / empty / negative-y / duplicate-x) carry
the actually-invoked renderer name: `hbar:` or `vbar:`. `bar` is an
alias so its shape errors surface as `hbar:`; the raw-string
`use @ref` case keeps the literally-written name (`bar:`).

### Toast-path errors

- `Unsupported data: frontmatter shape` — tab indent, flow style, or
  any shape outside the accepted YAML subset.
- Duplicate `::` name declaration.
- Unresolved `&name` reference at the end of parsing.

## CLI notes

The Go CLI under [`../src/go/`](../src/go/) reads a `.pylon` source
(positional argument or stdin) and emits ASCII, SVG, or PNG to a file
or stdout. It ships the same parser and renderers as the Web UI, so
a diagram that works in the browser works at the shell.

Install with:

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon@latest
```

The module path is `github.com/cmj0121/pylon/src/go`, not
`github.com/cmj0121/pylon`. Running
`go install github.com/cmj0121/pylon@latest` without the `/src/go`
suffix fails with an "unrecognized import path" error — this is a Go
sub-directory module, not a top-level one. The `cmd/pylon` tail selects
the CLI main package (a sibling `cmd/pylon-lsp` package holds the
Language Server) and yields a binary named `pylon` as expected.

### Formats and I/O

| Format  | Description                                                             |
| ------- | ----------------------------------------------------------------------- |
| `ascii` | Default. Box-drawing glyphs (or `+ - \|` under `theme: ascii`).         |
| `svg`   | One `<tspan>` per character cell for column-aligned monospace fidelity. |
| `png`   | Raster with embedded JetBrains Mono; glyphs outside coverage fall back. |

Conventions: positional `.pylon` path (stdin when omitted); output is
stdout when `-o` is `-` (default); verbose logging goes to stderr.
Flag details: `pylon --help`.

### Known limitations

- PNG output embeds JetBrains Mono; glyphs outside its coverage
  (CJK, emoji) fall back to tofu or the replacement glyph.
- SVG emits one `<tspan>` per character cell. Intentional — it keeps
  cells addressable and column-aligned across renderers.

The Go CLI binary bundles JetBrains Mono Regular (OFL 1.1, shipped
at
[`../src/go/pkg/pylon/assets/JetBrainsMono-OFL.txt`](../src/go/pkg/pylon/assets/JetBrainsMono-OFL.txt))
to rasterize PNG output. With stripped debug info
(`-ldflags="-s -w" -trimpath`), `dist/pylon` is around 4.8 MiB on
darwin/arm64.

## Building from source

Three subtrees ship independent `Makefile`s — the JS library, the Go
CLI, and the Vim plugin.

```sh
make -C src/js run            # dev server at http://localhost:3333
make -C src/js build          # minified bundle at dist/pylon.min.js
make -C src/js test           # smoke fixtures

make -C src/go build          # native binary at dist/pylon
make -C src/go test           # unit + golden fixtures
make -C src/go parity         # byte-compare Go vs JS ASCII output

make -C contrib/vim install   # Vim syntax plugin (override VIM_PACK for custom dests)
```

## Editor support

A Vim plugin lives under [`../contrib/vim/`](../contrib/vim/). It
ships syntax highlighting and filetype detection for `.pylon` files,
recognised by extension. Minimal install with vim-plug:

```vim
Plug 'cmj0121/pylon', { 'rtp': 'contrib/vim' }
```

`rtp` points vim-plug at the subdirectory that contains `syntax/` and
`ftdetect/`, not the repo root.

### Highlighted groups

| Group               | Matches                                                     |
| ------------------- | ----------------------------------------------------------- |
| `pylonFrontmatter`  | YAML-subset block between the two `---` fences at file head |
| `pylonFMKey`        | identifier keys inside frontmatter (`size:`, `data:`, …)    |
| `pylonFMNumber`     | numeric scalars inside frontmatter                          |
| `pylonFMString`     | double-quoted strings inside frontmatter                    |
| `pylonBracket`      | `[`, `]`, `(`, `)` around nodes                             |
| `pylonEdge`         | `->`, `<-`, `<->`, `-->`, `<-->`, … arrow tokens            |
| `pylonAlignMarker`  | leading / trailing `-` flush with a bracket                 |
| `pylonNameDeclare`  | `:: ident` trailing name declaration                        |
| `pylonNodeRef`      | `&ident` node back-references                               |
| `pylonDataRef`      | `@ident` data references (boundary-checked)                 |
| `pylonRendererPipe` | trailing `\| renderer` on a node body                       |

See [`../contrib/vim/README.md`](../contrib/vim/README.md) for
lazy.nvim, native packages, and bundled `Makefile` targets. The
regexes in `syntax/pylon.vim` duplicate grammar rules from
`src/js/pylon.js` — best-effort visual approximation, not a second
parser; the JS side is canonical when the two diverge.
