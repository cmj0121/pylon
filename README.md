# Pylon

> Draw your data and flow in plain text, by human or AI.

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

### Metadata

Like markdown, you can specify metadata at the top of the source, fenced by `---`. The metadata controls
graph-wide properties such as size and theme.

| Key   | Value Type | Description                                               |
| ----- | ---------- | --------------------------------------------------------- |
| size  | int x int  | Maximum outer width x height (content stays tight below). |
| theme | string     | Define the theme of the graph (e.g. ascii, dark, light).  |

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
make -C src/js build          # emits dist/pylon.min.js via `npx esbuild`
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

Frontmatter (YAML-subset, at the head of the source, fenced by `---`):

```pylon
---
size: 30x5                    # outer dimensions in cells
theme: simple | ascii | dark | light
---
[- Hello -]
```

`size` is a maximum — content stays tight to its own dimensions when smaller, and flow chains wrap to a vertical
stack when a line would exceed the declared width. `theme` swaps the palette (or switches the ASCII backend to
plain `+-|` glyphs when `ascii`). Both are optional.

Multiple top-level nodes stack vertically:

```pylon
[Hello]
(World)
```

## DDD (Dream-Driven Development)

This project follows the DDD (Dream-Driven Development) methodology, which means the project
is driven by what I envision.

All features are based on my needs and my dreams.
