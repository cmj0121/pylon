# Pylon

> Draw your data and flow in plain text, by human or AI.

[![CI](https://github.com/cmj0121/pylon/actions/workflows/ci.yml/badge.svg)](https://github.com/cmj0121/pylon/actions/workflows/ci.yml)

Live demo: <https://cmj0121.github.io/pylon/>

**Pylon** is a simple, human-friendly tool to draw your data and flow in plain text, by human or AI.
It focuses on keeping charts and diagrams in plain text so they are easy to edit, share, and collaborate on.

## Syntax

Pylon uses a simple syntax to define nodes and edges in a graph. Here are some examples:

| Name      | Syntax          | Description                                                           |
| --------- | --------------- | --------------------------------------------------------------------- |
| node      | `[ LABEL ]`     | Bordered node. `( LABEL )` is the borderless variant.                 |
| edge      | `->`            | Edge from the previous node to the next. `<-`, `<->`, `-->` etc. too. |
| alignment | `-`             | Alignment marker flush with the bracket (`[- x -]` centers `x`).      |
| name      | `[ x :: NAME ]` | Tag a node with `NAME` so it can be referenced later.                 |
| ref       | `&NAME`         | Reference a previously named node.                                    |
| data ref  | `@NAME`         | Reference a data series from frontmatter.                             |
| renderer  | `\| NAME`       | Pipe box content to a renderer, e.g. `\| bar`.                        |

### Metadata

Like markdown, you can specify metadata at the top of the source, fenced by `---`. The metadata controls
graph-wide properties such as size and theme.

| Key   | Value Type  | Description                                                         |
| ----- | ----------- | ------------------------------------------------------------------- |
| size  | int x int   | Maximum outer width x height (content stays tight below).           |
| theme | string      | Define the theme of the graph (e.g. ascii, dark, light).            |
| data  | list or map | One or more `{x, y}` series for `@ref` / renderer boxes. See below. |

## Example

A single centered label:

```pylon
[- 12:00 -]
```

```txt
┌───────────┐
│   12:00   │
└───────────┘
```

A three-node flow chart:

```pylon
[ Start ] -> [ Process ] -> [ End ]
```

```txt
┌───────────┐  ┌─────────────┐  ┌─────────┐
│   Start   │─▶│   Process   │─▶│   End   │
└───────────┘  └─────────────┘  └─────────┘
```

The same diagram under `theme: ascii` swaps the Unicode glyphs for plain `+-|<>`:

```txt
+-----------+  +-------------+  +---------+
|   Start   |->|   Process   |->|   End   |
+-----------+  +-------------+  +---------+
```

More examples live under [`examples/`](examples/) — one file per feature
(flow chains, named refs, labelled edges, bar charts, themes).

## Data and Renderers

Boxes can render data instead of literal text. Declare series in a `data:` frontmatter
block, reference them with `@NAME`, and pipe them into a renderer with `| NAME`.

### `data:` frontmatter

Two shapes are accepted. A flat list is reachable as `@data`:

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

A map exposes each key as its own reference (`@counter`, `@sales`):

```pylon
---
data:
  counter:
    - x: 1
      y: 10
    - x: 2
      y: 20
  sales:
    - x: 1
      y: 100
    - x: 2
      y: 50
---
[ @sales | bar ]
```

The `data:` parser is a small YAML subset: block style only, spaces-only indent (tabs
reject the whole frontmatter via toast), and scalars may be numbers, double-quoted
strings, or unquoted strings (trimmed, taken literally). Reserved words like `true`,
`false`, `null`, `~`, `yes`, and `no` are not interpreted — they stay literal strings.
Flow style (`[...]`, `{...}`), multiline scalars, anchors, and tags are not supported.
Unsupported shapes leave `data` unset and surface an `Unsupported data: frontmatter
shape` toast.

Categorical `x` labels work the same way — unquoted string keys are the friendly
default:

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

Renders as:

```txt
┌──────────────────────────────────┐
│     apples │ █████      (10) │   │
│    bananas │ ██████████ (20) │   │
│   cherries │ ████████   (15) │   │
└──────────────────────────────────┘
```

### `@ref` data references

`@<name>` inside a box body resolves to the named series at render time. `@` is only
treated as a sigil when it is preceded by `[`, `(`, whitespace, or start-of-string, and
followed by an identifier then whitespace / `|` / `]` / `)` / end-of-string. Labels like
`[ user@example.com ]` or mid-word `foo@bar` stay literal text.

`@ref` shares no namespace with `&ref`: `&` references a previously declared node, `@`
references a data series from frontmatter.

### `| renderer` pipeline

A trailing `| <name>` inside a box marks the renderer. Default is `text` when omitted,
and whitespace around `|` is required so literal `|` in labels still works.

- `[ label ]` — uses `text`; label rendered as-is.
- `[ @counter | bar ]` — pipes the resolved series to `bar`.
- `[ @data | bar | text ]` — first pipe wins (`bar`); extras are consumed silently.
- `[ data | bar | 123 ]` — malformed chain (non-identifier tail); the whole `| ...` is
  left as literal label text.

The `| <ident>` chain only extracts when it runs to end-of-label.

### Available renderers (v0.2.0)

| Renderer       | Input              | Output                                                                   |
| -------------- | ------------------ | ------------------------------------------------------------------------ |
| `text`         | string or `@ref`   | Pass-through; `@ref` is emitted via `JSON.stringify`.                    |
| `hbar` (`bar`) | `@ref` → `[{x,y}]` | Horizontal bars scaled against `max(y)` with `(value)` labels.           |
| `vbar`         | `@ref` → `[{x,y}]` | Vertical bars scaled against `max(y)`; `x` and `(value)` labels as feet. |

`bar` is kept as a v0.1 alias for `hbar`; both render identically.

`text` example:

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

`bar` example:

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

`bar` uses a 10-cell bar-width budget by default; a tight `size:` clips the bar first
and then truncates the `(value)` label. `bar` only accepts an `@ref`; a raw string
surfaces `⚠ bar: use @ref`.

`vbar` example:

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

`vbar` mirrors `hbar`'s 10-cell budget but applies it to the chart HEIGHT. Each
series entry becomes a single-column bar, column width is `max(|label|, |(value)|, 3)`,
and the two footer rows carry the `x` label and `(value)` text. All-zero `y` renders
zero-height bars (no error).

### Composition

`| renderer` composes with `::` naming, alignment dashes, and borderless `(...)`:

- `[- @counter | bar -]` — centered chart.
- `[ @counter | bar :: chart1 ]` — named chart (tag before the pipe is fine too).
- `( @counter | bar )` — borderless chart (wrapper drops, bars flow into the parent).

### Errors

Parse-shape errors (malformed `data:`, tab indent) go to the `<pylon-chart>` toast,
same mechanism as duplicate-name errors. Render-time errors appear inline inside the
offending box, prefixed with `⚠`, so users can pinpoint them:

| Trigger                            | Inline message           |
| ---------------------------------- | ------------------------ |
| Raw string piped to non-`text`     | `bar: use @ref`          |
| Unknown renderer name              | `unknown renderer: foo`  |
| `@ref` not declared in frontmatter | `@missing not found`     |
| Renderer got wrong shape           | `hbar: expected [{x,y}]` |
| Empty series for a bar renderer    | `hbar: empty series`     |
| Any negative `y`                   | `hbar: negative y`       |
| Duplicate `x` in series            | `hbar: duplicate x "a"`  |

Bar-family errors (shape / empty / negative-y / duplicate-x) carry the actually-invoked
renderer name: `hbar:` or `vbar:`. `bar` is an alias so its shape errors surface as
`hbar:`; the raw-string `use @ref` case keeps the literally-written name.

A bare `@ref` with no renderer surfaces its own inline error (`@<name>: requires a
renderer`) so the omission is loud rather than silent.

Long inline errors are clipped by the box width, same as any other content.

## Web UI

A zero-dependency JavaScript scaffold lives under [`src/js/`](src/js/). It registers a `<pylon-chart>` custom element
that renders a Pylon source string as **ASCII**, **SVG**, or **PNG**, and an optional `wysiwyg` attribute turns the
element into a split-pane editor with a live preview and Copy / Download buttons for the current format.

### Run the demo

```sh
make -C src/js run            # serves src/js/ on http://localhost:3333
```

Or open `src/js/index.html` directly in a browser.

### Build a minified bundle

```sh
make -C src/js build          # emits dist/{pylon.min.js,pylon.css,index.html}
```

### Embed

```html
<link rel="stylesheet" href="pylon.css" />
<script src="pylon.js"></script>

<!-- View-only -->
<pylon-chart format="svg">[- Hello -]</pylon-chart>

<!-- Split editor + preview, dropdown switches format -->
<pylon-chart wysiwyg></pylon-chart>
```

Attributes: `format="ascii|svg|png"` (default `ascii`), `src` (source, alternative to child text), `wysiwyg` (flag).
All colors, spacing, fonts, and shape are themeable via `--pylon-*` custom properties in `pylon.css`.

### Syntax reference

Nodes:

- `[ label ]` bordered node.
- `( label )` borderless node (transparent wrapper; children flow into the parent).
- A node body may contain text lines, nested nodes, or a flow chain; lines stack vertically.

Alignment inside a node (dash must be flush with the bracket):

- `[ x ]` or `[- x -]` — centered (default).
- `[- x ]` — right-aligned (left spring pushes content right).
- `[ x -]` — left-aligned.

Flow chains (siblings on a single line, connected by edges):

- `[ A ] -> [ B ]` right arrow.
- `[ A ] <- [ B ]` left arrow.
- `[ A ] <-> [ B ]` bidirectional.
- `[ A ] --> [ B ]`, `[ A ] ---> [ B ]`, ... longer edge lines (any `<?-+>?` with at least one arrow).
- `[ Alice ] -- ( friend ) --> [ Bob ]` — labelled edge. The label is any borderless node written between the two
  edge halves; bidirectional `<-- ( role ) -->` and reverse `<-- ( role ) --` also work.

Named nodes and references:

- `[ value :: name ]` — declare a node named `name`. The trailing `::` + identifier is stripped from the rendered
  label and attached to the AST. Applies to `(...)` as well.
- `&name` — reference a previously declared node. References are pointers, not copies: the declaration is the
  only place the full box is drawn. How a reference renders depends on where the declaration lives:

  - **Same row, adjacent** (e.g. `[ a :: b ] -> &b`) — drawn as a self-loop arc under the declaration.

    ```txt
    ┌───────┐
    │   a   │
    └───────┘
      │   ▲
      └───┘
    ```

  - **Different row** (e.g. `[ a :: b ]` on one line, `[ c ] -> &b` on the next) — drawn as an arrow through a
    right-side gutter that loops back to the declaration.

    ```txt
    ┌───────┐
    │   a   │◀────┐
    └───────┘     │
    ┌───────┐     │
    │   c   │─────┘
    └───────┘
    ```

  - **Anywhere else** (nested refs, second ref to an already-gutter-routed target, etc.) — the reference falls
    back to the plain name as inline text.

- Duplicate declarations and unresolved references surface as a toast inside the `<pylon-chart>` element
  (no native `alert`).

Data references and renderers:

- `@name` — reference a data series from the `data:` frontmatter. The sigil is recognised only at a box-body
  boundary: `@` must follow `[`, `(`, whitespace, or start-of-string, and be followed by an identifier then
  whitespace / `|` / `]` / `)` / end-of-string. Anything else — `user@example.com`, mid-word `foo@bar` — stays
  literal label text. Namespace is independent from `&name` (node references).
- `| <renderer>` — trailing pipe inside a box routes the body to a renderer. Default is `text`. Whitespace
  around `|` is required, so `[ foo | bar ]` still reads as literal when `bar` is not a known identifier at
  end-of-label. Chains (`| a | b | c`) extract only when every segment is a valid identifier up to the closing
  bracket; first renderer wins, the rest are consumed silently. Malformed chains fall back to literal label
  text.
- v0.2.0 ships three renderers:

  - `text` (default): string pass-through; `@ref` is emitted via `JSON.stringify`.
  - `hbar` (alias `bar`): horizontal bar chart over `[{x, y}, ...]`. Scales `y` against `max(y)` into a
    10-cell bar budget, right-labels each row with `(value)`. Requires `@ref` input — a raw string surfaces
    `⚠ hbar: use @ref`.
  - `vbar`: vertical bar chart over the same `[{x, y}, ...]` shape. Each entry becomes a single-column bar;
    the 10-cell budget is the chart HEIGHT, and two footer rows carry the `x` label and `(value)` text.

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

- Composes with `::` naming, alignment dashes, and borderless `(...)`: `[- @counter | bar -]`,
  `[ @counter | bar :: chart1 ]`, `( @counter | bar )` all work. Order inside the brackets is
  `<dashes>? <content> :: <name>? | <renderer>? <dashes>?`.
- Render-time errors (bad renderer, missing ref, wrong shape) render inline inside the offending box as
  `⚠ ...`. Parse-shape errors on `data:` (tab indent, unrecognised structure) surface as a toast, same as
  duplicate-name errors.

Frontmatter (YAML-subset, at the head of the source, fenced by `---`):

```pylon
---
size: 30x5                    # outer dimensions in cells
theme: simple | ascii | dark | light
data:                         # optional series for @ref / renderer boxes
  - x: 1
    y: 10
---
[- Hello -]
```

`size` is a maximum — content stays tight to its own dimensions when smaller, and flow chains wrap to a vertical
stack when a line would exceed the declared width. `theme` swaps the palette (or switches the ASCII backend to
plain `+-|` glyphs when `ascii`). `data:` holds one or more `{x, y}` series (flat list or keyed map) for
`@ref` boxes — see the **Data and Renderers** section. All three keys are optional.

Multiple top-level nodes stack vertically:

```pylon
[Hello]
(World)
```

## Editor support

A Vim plugin lives under [`contrib/vim/`](contrib/vim/) providing syntax highlighting and
filetype detection for Pylon source files. The recognised filename extension is `.pylon`.

Install with [vim-plug](https://github.com/junegunn/vim-plug):

```vim
Plug 'cmj0121/pylon', { 'rtp': 'contrib/vim' }
```

See [`contrib/vim/README.md`](contrib/vim/README.md) for other install methods (lazy.nvim,
native packages).

Every colored construct in a single snippet — frontmatter, bordered node, alignment,
edge, named node, `&ref`, `@ref`, and `| bar`:

```pylon
---
size: 60x20
theme: simple
data:
  - x: 1
    y: 10
  - x: 2
    y: 20
---
[- Start :: begin -] -> [ Process ] -> [ End -]
[ Loop ] -> &begin
[ @data | bar ]
```

The colors only appear in Vim; GitHub's markdown renderer leaves the `pylon` fence tag
as plain text.

## DDD (Dream-Driven Development)

This project follows the DDD (Dream-Driven Development) methodology, which means the project
is driven by what I envision.

All features are based on my needs and my dreams.
