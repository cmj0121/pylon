# Pylon

[![CI](https://github.com/cmj0121/pylon/actions/workflows/ci.yml/badge.svg)](https://github.com/cmj0121/pylon/actions/workflows/ci.yml)

> Draw your data and flow in plain text, by human or AI.

Pylon is a small language for writing charts and diagrams as plain
text. The same source renders identically in the browser, on the
command line, and inside your editor.

## Example

```pylon
[ Start ] -> [ Process ] -> [ End ]
```

```txt
┌───────────┐  ┌─────────────┐  ┌─────────┐
│   Start   │─▶│   Process   │─▶│   End   │
└───────────┘  └─────────────┘  └─────────┘
```

Under `theme: ascii`, the same source renders with plain `+-|<>`
glyphs for terminals that prefer pure ASCII:

```txt
+-----------+  +-------------+  +---------+
|   Start   |->|   Process   |->|   End   |
+-----------+  +-------------+  +---------+
```

Frontmatter opts into data series. A `| bar` pipeline turns a box
into a chart:

```pylon
---
data:
  - x: apples
    y: 10
  - x: bananas
    y: 20
  - x: cherries
    y: 15
---
[ @data | bar ]
```

```txt
┌────────────────────────────┐
│   apples │ █████      (10) │
│  bananas │ ██████████ (20) │
│ cherries │ ████████   (15) │
└────────────────────────────┘
```

## Quick start

### Browser

```html
<link rel="stylesheet" href="pylon.css" />
<script src="pylon.min.js"></script>

<pylon-chart format="svg">[ Start ] -> [ End ]</pylon-chart>
<pylon-chart wysiwyg></pylon-chart>
```

Live demo: <https://cmj0121.github.io/pylon/>.

### Command line

Pre-built `pylon` (and `pylon-lsp`) binaries for linux (amd64,
arm64) and macOS (Intel, Apple Silicon) are published on each
release — download from
<https://github.com/cmj0121/pylon/releases>, extract the `tar.gz`,
and drop the binary somewhere on your `$PATH`.

On other platforms (Windows, linux/arm, BSD, …) build from source
with Go 1.25+:

```sh
go install github.com/cmj0121/pylon/src/go/cmd/pylon@latest
```

```sh
# ASCII from stdin, rendered to stdout (default):
echo '[ Start ] -> [ End ]' | pylon

# SVG to a file:
pylon -f svg -o diagram.svg examples/flow-chart.pylon

# PNG to a file (embeds JetBrains Mono for glyph metrics):
pylon -f png -o diagram.png examples/flow-chart.pylon
```

Flag details: `pylon --help`.

### Vim

```vim
Plug 'cmj0121/pylon', { 'rtp': 'contrib/vim' }
```

See [`contrib/vim/`](contrib/vim/) for lazy.nvim, native packages,
and the bundled install target.

## Syntax at a glance

| Name      | Syntax          | Description                                                              |
| --------- | --------------- | ------------------------------------------------------------------------ |
| node      | `[ LABEL ]`     | Bordered node. `( LABEL )` is the borderless variant.                    |
| edge      | `->`            | Edge from the previous node to the next. `<-`, `<->`, `-->` etc. too.    |
| alignment | `-`             | Alignment marker flush with the bracket (`[- x -]` centers `x`).         |
| name      | `[ x :: NAME ]` | Tag a node with `NAME` so it can be referenced later.                    |
| ref       | `&NAME`         | Reference a previously named node.                                       |
| data ref  | `@NAME`         | Reference a data series from frontmatter.                                |
| renderer  | `\| NAME`       | Pipe to a renderer (`bar`, `vbar`, `sparkline`, … — see SPEC catalogue). |

See [`docs/SPEC.md`](docs/SPEC.md) for the full grammar, renderer
catalogue, and error model.

## Learn more

- [`examples/`](examples/) — nineteen standalone sample files,
  one per feature, plus [`showcase.pylon`](examples/showcase.pylon)
  demonstrating every primitive in one source.
- [`docs/SPEC.md`](docs/SPEC.md) — specification reference.
- [`docs/LSP.md`](docs/LSP.md) — Language Server reference (install,
  editor setup, diagnostics catalogue).
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — deferred gaps on the way to
  v1.0.
- Live demo: <https://cmj0121.github.io/pylon/>.

## DDD (Dream-Driven Development)

This project follows the DDD (Dream-Driven Development) methodology, which means the project
is driven by what I envision.

All features are based on my needs and my dreams.

## License

Pylon is released under the [MIT License](LICENSE).

The bundled [JetBrains Mono Regular](src/go/pkg/pylon/assets/JetBrainsMono-Regular.ttf) TTF
(embedded in the Go CLI's PNG renderer) ships under the **SIL Open Font License 1.1**;
the full text is in
[`src/go/pkg/pylon/assets/JetBrainsMono-OFL.txt`](src/go/pkg/pylon/assets/JetBrainsMono-OFL.txt).
