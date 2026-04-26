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

| Key      | Value type    | Description                                                                           |
| -------- | ------------- | ------------------------------------------------------------------------------------- |
| `size`   | `INT x INT`   | Maximum outer width x height in cells. Content stays tight when smaller.              |
| `theme`  | identifier    | Palette / glyph set: `simple` (default), `ascii`, `dark`, `light`.                    |
| `data`   | list or map   | One or more `{x, y}` series for `@ref` / renderer boxes.                              |
| `color`  | `true\|false` | Source-file preference for ANSI color in ASCII output. See [ANSI color](#ansi-color). |
| `banner` | identifier    | Default font for `\| banner` boxes: `monospace`, `digital`, or `mini`.                |

`size` caps the outer dimensions; flow chains that would exceed the
declared width wrap to a vertical stack. `theme: ascii` swaps the
Unicode box-drawing glyphs for plain `+ - | < >`.

### YAML subset accepted by `data:`

The `data:` parser is a narrow YAML subset, not full YAML.

- **Block style only.** Flow style (`[...]`, `{...}`) is rejected,
  except `y:` may take a flow-style numeric array (`y: [1, 2, 3]`)
  to fit 2D shapes like `heatmap`. Flow maps and non-numeric flow
  arrays stay rejected.
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

Pylon 0.2.0 ships eleven renderers. Unknown names surface an inline
`⚠ unknown renderer: NAME`.

| Renderer       | Input                       | Output                                                          |
| -------------- | --------------------------- | --------------------------------------------------------------- |
| `text`         | string or `@ref`            | Pass-through; `@ref` emitted via `JSON.stringify`.              |
| `hbar` (`bar`) | `@ref` → `[{x,y}]`          | Horizontal bars scaled against `max(y)` with `(value)` labels.  |
| `vbar`         | `@ref` → `[{x,y}]`          | Vertical bars scaled against `max(y)`; `x`/`(value)` as feet.   |
| `banner`       | literal string              | 6-row block-letter banner of the uppercased source text.        |
| `progress`     | scalar or `@ref`→`[{x,y}]`  | Progress bar(s) with a right-aligned `%` label; 20-cell budget. |
| `heatmap`      | `@ref` → `[{x, y:[n,...]}]` | 2D matrix of ramp glyphs (`░▒▓█` / `.+*#`).                     |
| `sparkline`    | `@ref` → `[{x,y}]`          | Single-row ramp; 8 levels default / 4 levels ascii.             |
| `candlestick`  | `@ref` → `[{x,o,h,l,c}]`    | Fixed 8-row OHLC candles with bull/bear/doji bodies and wicks.  |
| `hist`         | `@ref` → `[{x,y}]`          | Gap-less 8-row histogram bars; bar-family validators.           |
| `step`         | `@ref` → `[{x,y}]`          | 5-row cumulative step line with box-drawing corner connectors.  |
| `gantt`        | `@ref` → `[{x,start,end}]`  | Horizontal task bars on a shared budget; size-aware.            |

`bar` is a v0.1 alias for `hbar`; both render identically.

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
┌─────────────────────┐
│ 1 │ █████      (10) │
│ 2 │ ██████████ (20) │
│ 3 │ ████████   (15) │
└─────────────────────┘
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
┌──────────────┐
│      █       │
│      █       │
│      █   █   │
│      █   █   │
│      █   █   │
│  █   █   █   │
│  █   █   █   │
│  █   █   █   │
│  █   █   █   │
│  █   █   █   │
│  1   2   3   │
│ (10)(20)(15) │
└──────────────┘
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
Input is uppercased before lookup. Default and `ascii`-theme
banners cover every printable ASCII codepoint from 0x20 (space)
through 0x7E (tilde) — the alphanumerics use the pyfiglet
`ansi_shadow` style and the symbols use the same uniform block
shapes as `banner: monospace` (visual mismatch is a known
trade-off for full coverage). Unknown runes outside the printable
ASCII range fall back to the `?` glyph. `theme: ascii` swaps the
ANSI-shadow box-drawing glyphs for a `#` + space grid.

**East Asian Width caveat.** Every banner font that mixes narrow
ASCII spaces (`U+0020`) with ambiguous-width chars (`█`, `╗`, `═`,
`║`, `╚`, `╝`) misaligns under EAW-wide rendering. EAW-narrow
contexts (most Western terminals, the default for monospace web
fonts) treat both classes as one cell each, and the output aligns.
EAW-wide contexts (CJK terminals, locale-aware monospace fonts)
widen the ambiguous chars to two cells while leaving spaces at one
cell, so banner rows with different space-to-block ratios end up at
different visual widths — the frame `┌──...──┐` no longer lines up
with the inner `│ ████ │` rows. This affects the default font and
all three pure-`█` fonts (`monospace`, `digital`, `mini`) when
rendered without `theme: ascii`. Use `theme: ascii` for CJK /
EAW-wide output: the default font's ASCII variant (`bannerFontASCII`)
uses only narrow chars (`#` + space) and the pure-`█` fonts'
runtime `█→#` substitution produces an identically narrow output,
so every banner family renders consistently in any East Asian width
context under `theme: ascii`.

v1 is literal-string only. A `@ref` inside a `| banner` box is
silently ignored — a future release will resolve `@ref` to its
value and render that instead.

#### Selecting a font

Three pure-`█` banner fonts ship in addition to the default ANSI-shadow
font: `monospace`, `digital`, and `mini`. Pick a font two ways:

- **Per-box** — `[ TEXT | banner:FONT ]` overrides the font for that
  banner only. Use this when one image needs to mix multiple fonts
  (e.g. a `digital` headline above a `mini` subtitle).
- **Frontmatter** — `banner: FONT` picks the image-wide default. Every
  `| banner` box without a per-box `:FONT` arg uses this font.

Resolution order: per-box arg → frontmatter `banner:` → theme dispatch
(ANSI-shadow under default, `#` + space under `theme: ascii`). Unknown
font names silently fall through, matching how `theme:`, `size:`, and
`color:` swallow malformed input.

All three pure-`█` fonts cover every printable ASCII codepoint from
`0x20` through `0x7E`. Under `theme: ascii` the renderer substitutes
`█` → `#` on the rendered rows so the ASCII theme stays valid for any
font choice.

##### `monospace`

Uniform-width `█`-based block-letter font where every glyph occupies
the same cell width (8 cells). Use it when you want grid-aligned
banner output — columns line up cleanly across letters, which the
ANSI-shadow default does not guarantee.

```pylon
---
banner: monospace
---
[ Pylon | banner ]
```

```txt
┌──────────────────────────────────────────────┐
│   ██████  ██   ██ ██       █████  ██   ██    │
│   ██  ██   ██ ██  ██      ██   ██ ███  ██    │
│   ██████    ████  ██      ██   ██ ████ ██    │
│   ██         ██   ██      ██   ██ ██ ████    │
│   ██         ██   ██      ██   ██ ██  ███    │
│   ██         ██   ██████   █████  ██   ██    │
└──────────────────────────────────────────────┘
```

##### `digital`

7-segment LCD-style font, 6 cells wide × 6 rows tall. Single-thick
strokes; digits are literal 7-segment shapes, letters are stylized
approximations. Use it for status displays, scoreboards, retro UI.

```pylon
[ Pylon 0.2 | banner:digital ]
```

```txt
┌────────────────────────────────────────────────────────────┐
│   █████ █   █ █     █████ █   █       █████       █████    │
│   █   █ █   █ █     █   █ ██  █       █   █           █    │
│   █████ █████ █     █   █ █ █ █       █  ██       █████    │
│   █         █ █     █   █ █  ██       █ █ █       █        │
│   █         █ █     █   █ █   █       ██  █       █        │
│   █         █ █████ █████ █   █       █████   █   █████    │
└────────────────────────────────────────────────────────────┘
```

##### `mini`

Compact 1-thick-stroke font, 5 cells wide × 6 rows tall. Use it for
inline-label banners or whenever screen space is tight.

```pylon
[ Pylon 0.2 | banner:mini ]
```

```txt
┌───────────────────────────────────────────────────┐
│   ███  █  █ █     ██  █  █       ██        ██     │
│   █  █ █  █ █    █  █ ██ █      █  █      █  █    │
│   ███   ██  █    █  █ █ ██      █ ██        █     │
│   █      █  █    █  █ █  █      ██ █       █      │
│   █      █  █    █  █ █  █      █  █      █       │
│   █      █  ████  ██  █  █       ██   █   ████    │
└───────────────────────────────────────────────────┘
```

### progress

```pylon
[ 75 | progress ]
```

```txt
┌───────────────────────────┐
│ ███████████████░░░░░  75% │
└───────────────────────────┘
```

```pylon
---
data:
  work:
    - x: build
      y: 100
    - x: test
      y: 70
    - x: deploy
      y: 10
---
[ @work | progress ]
```

```txt
┌──────────────────────────────────┐
│  build ████████████████████ 100% │
│   test ██████████████░░░░░░  70% │
│ deploy ██░░░░░░░░░░░░░░░░░░  10% │
└──────────────────────────────────┘
```

`progress` accepts two input shapes. A literal number in `[0,100]`
renders a single row; a `@ref` pointing to `[{x,y}]` renders one row
per entry with the `x` labels padded into a left-hand column. Each
bar uses a 20-cell budget and is followed by a right-aligned `%3d%%`
label. `y` values outside `[0,100]` are clamped silently (no
diagnostic). Glyphs are `█` filled and `░` empty by default; `theme:
ascii` swaps them for `#` and `.`.

Scalar input that isn't a parseable number surfaces `⚠ progress:
expected number`. Series input reuses the bar-shape validators, so a
malformed series reports `progress: expected [{x,y}]`, `progress:
empty series`, or `progress: duplicate x "…"` (negative `y` is
caught as `progress: negative y` — out-of-range in the other
direction is clamped rather than flagged).

### heatmap

```pylon
---
data:
  activity:
    - x: Mon
      y: [1, 2, 4, 6, 8]
    - x: Tue
      y: [0, 3, 5, 7, 4]
    - x: Wed
      y: [2, 5, 8, 5, 2]
    - x: Thu
      y: [0, 1, 3, 2, 0]
---
[ @activity | heatmap ]
```

```txt
┌───────────┐
│ Mon ░░▒▓█ │
│ Tue  ▒▓█▒ │
│ Wed ░▓█▓░ │
│ Thu  ░▒░  │
└───────────┘
```

`heatmap` accepts an `@ref` pointing to `[{x, y:[n,...]}]` — a
labeled 2D matrix where every row's `y` array has the same length.
Each cell is one glyph on a 5-level ramp; there is no separator
between cells. The ramp is `░▒▓█` by default and `.+*#` under
`theme: ascii`. A right-aligned label column appears when any row's
`x` is non-empty; all-empty labels suppress the column entirely.
Values normalize against the whole matrix: `idx = round(v /
max(all values) * 4)`, so an all-zero input renders as all blanks
rather than an error.

Shape problems (missing `y`, ragged rows, non-numeric entries)
surface `⚠ heatmap: expected [{x, y:[n,...]}]`. An empty series
reports `heatmap: empty series`, and any negative value reports
`heatmap: negative y` — the bar-family validators apply with the
`heatmap:` prefix.

### sparkline

```pylon
---
data:
  trend:
    - x: 1
      y: 12
    - x: 2
      y: 15
    - x: 3
      y: 9
    - x: 4
      y: 18
    - x: 5
      y: 14
---
[ @trend | sparkline ]
```

```txt
┌───────┐
│ ▃▆▁█▅ │
└───────┘
```

`sparkline` accepts an `@ref` pointing to `[{x,y}]` and renders a
single row of ramp glyphs — one per entry. `x` is ignored; only `y`
drives the glyph. The ramp is `▁▂▃▄▅▆▇█` (8 levels, U+2581..U+2588)
by default and `_-*#` (4 levels — the honest ceiling for a monotone
ASCII height ramp) under `theme: ascii`.

Normalization is **relative**: `idx = round((y - min) / (max - min) *
(ramp.length - 1))`. The first occurrence of `min` always maps to
the lowest glyph and `max` to the highest, so the trend shape pops
even when every value is large. A flat series (`min == max`) renders
as all-lowest-glyph — deterministic rather than division-by-zero.

Unlike the bar family, sparkline **tolerates negative `y` and
duplicate `x`**: trend viz over signed values (temperature deltas,
price moves) is the canonical use. Shape problems still surface
`⚠ sparkline: expected [{x,y}]`, and an empty series reports
`sparkline: empty series` — no new diagnostic codes.

### candlestick

```pylon
---
data:
  week:
    - x: Mon
      o: 1
      h: 5
      l: 0
      c: 4
    - x: Tue
      o: 6
      h: 7
      l: 2
      c: 3
    - x: Wed
      o: 4
      h: 5
      l: 3
      c: 4
    - x: Thu
      o: 2
      h: 6
      l: 1
      c: 5
    - x: Fri
      o: 7
      h: 7
      l: 4
      c: 5
---
[ @week | candlestick ]
```

```txt
┌─────────────────┐
│    │        █   │
│    █     │  █   │
│ │  █  │  ▒  █   │
│ ▒  █  ─  ▒  │   │
│ ▒  █  │  ▒      │
│ ▒  │     ▒      │
│ ▒        │      │
│ │               │
│ MonTueWedThuFri │
└─────────────────┘
```

`candlestick` accepts an `@ref` pointing to `[{x, o, h, l, c}]` —
one entry per column, with `x` as the label and `o`, `h`, `l`, `c`
as the open / high / low / close prices. Bodies draw on a fixed
8-row grid; wicks extend from `h` down to the top of the body and
from the bottom of the body down to `l`. Bullish (`c > o`) uses
`▒`, bearish (`c < o`) uses `█`, and doji (`c == o`) collapses the
body to a single `─` row. Wicks are `│`. Under `theme: ascii` the
glyphs swap to `+` bull, `#` bear, `-` doji, `|` wick.

Column width is `max(|xlabel|, 1)`; the candle glyph occupies the
left-most cell of each column and a footer row carries the labels.
When every entry has `x: ""` the renderer drops the footer and
packs columns at width 1 — a **compact mode** suited to dense,
trend-focused layouts. The same week of data rendered compact:

```pylon
---
data:
  week:
    - x: ""
      o: 1
      h: 5
      l: 0
      c: 4
    - x: ""
      o: 6
      h: 7
      l: 2
      c: 3
    - x: ""
      o: 4
      h: 5
      l: 3
      c: 4
    - x: ""
      o: 2
      h: 6
      l: 1
      c: 5
    - x: ""
      o: 7
      h: 7
      l: 4
      c: 5
---
[ @week | candlestick ]
```

```txt
┌───────┐
│  │  █ │
│  █ │█ │
│ │█│▒█ │
│ ▒█─▒│ │
│ ▒█│▒  │
│ ▒│ ▒  │
│ ▒  │  │
│ │     │
└───────┘
```

Normalization is **global**: rows map across the series
`min(l over all)` → `max(h over all)`, so every candle shares one
y-scale. A fully flat series collapses every candle to a single
doji row — deterministic rather than division-by-zero.

Like sparkline, candlestick **tolerates negative values and
duplicate `x`** — signed P&L deltas and repeated dates are normal.
Shape problems (missing field, non-numeric or NaN price) surface
`⚠ candlestick: expected [{x, o, h, l, c}]`; an empty series
reports `candlestick: empty series`; and an entry where
`h < max(o, c)` or `l > min(o, c)` reports
`candlestick: invalid ohlc`.

### hist

```pylon
---
data:
  letters:
    - x: M
      y: 1
    - x: I
      y: 4
    - x: S
      y: 4
    - x: P
      y: 2
---
[ @letters | hist ]
```

```txt
┌──────┐
│  ██  │
│  ██  │
│  ██  │
│  ██  │
│  ███ │
│  ███ │
│ ████ │
│ ████ │
│ MISP │
└──────┘
```

`hist` accepts an `@ref` pointing to `[{x,y}]` where `y` is a
non-negative pre-binned count. Bars draw on a fixed 8-row grid,
one column per entry, **width 1 with no gaps** — this is what
distinguishes `hist` from `vbar`, which pads columns and feet
with whitespace. An x-label footer appears when any entry has a
non-empty `x`; when every `x` is empty the footer drops and the
chart packs into a compact, label-free mode.

Normalization is `cells = round(y / max(y) * 8)`, so the tallest
bar always fills the full 8 rows. A zero-max series degenerates
to a flat empty chart — deterministic rather than
division-by-zero. Glyphs are `█` by default and `#` under
`theme: ascii` (bar-family glyph set).

Validation reuses the bar-family validators: shape problems
surface `⚠ hist: expected [{x,y}]`, an empty series reports
`hist: empty series`, any negative `y` reports
`hist: negative y`, and duplicate `x` reports
`hist: duplicate x "…"`. No new diagnostic codes.

### step

```pylon
---
data:
  signups:
    - x: w1
      y: 5
    - x: w2
      y: 12
    - x: w3
      y: 14
    - x: w4
      y: 25
    - x: w5
      y: 40
    - x: w6
      y: 55
---
[ @signups | step ]
```

```txt
┌─────────────┐
│          ┌─ │
│        ┌─┘  │
│      ┌─┘    │
│  ┌───┘      │
│ ─┘          │
└─────────────┘
```

`step` accepts an `@ref` pointing to `[{x,y}]` and renders a
fixed 5-row step chart — one horizontal segment per entry,
connected by box-drawing corners where values change. Each
entry occupies **two columns**: the flat level of that entry
draws as `─`, and transitions between levels use `┌ ┐ └ ┘` to
turn, with `│` filling multi-row vertical runs. Under
`theme: ascii` the glyphs fold to `- |` with `+` at every
corner.

Normalization is **global** across the series: `row(y) = (H-1)

- round((y - gMin) / (gMax - gMin) \* (H-1))`with`H = 5`, so
`gMin`pins the bottom row and`gMax`pins the top. A flat
series (`gMin == gMax`) collapses to a single horizontal line
  along the bottom row — deterministic rather than
  division-by-zero.

Like sparkline, step **tolerates negative `y` and duplicate
`x`** — cumulative counters that reset, signed deltas, and
repeated x-bins are all normal. Shape problems surface
`⚠ step: expected [{x,y}]` and an empty series reports
`step: empty series` — no new diagnostic codes.

### gantt

```pylon
---
data:
  sprint:
    - x: spec
      start: 0
      end: 5
    - x: code
      start: 3
      end: 12
    - x: test
      start: 10
      end: 16
    - x: ship
      start: 14
      end: 20
---
[ @sprint | gantt ]
```

```txt
┌───────────────────────────┐
│ spec █████                │
│ code    █████████         │
│ test           ██████     │
│ ship               ██████ │
└───────────────────────────┘
```

`gantt` accepts an `@ref` pointing to `[{x, start, end}]` —
one task per row, with `x` as the task label and `start` /
`end` as finite numeric offsets on a shared horizontal budget.
Labels render right-aligned in a left-hand column, and each
task's bar spans `start..end-1` across the budget. Glyphs are
`█` by default and `#` under `theme: ascii`.

The budget is **20 cells by default**. When the frontmatter
declares `size: W x H`, the budget retunes to
`max(5, W - labelWidth - 4)`, so a tighter `size:` squeezes the
bars rather than truncating labels. The floor of 5 cells keeps
short-label charts readable even at aggressive widths.

Shape problems (missing field, non-numeric `start` / `end`)
surface `⚠ gantt: expected [{x, start, end}]`, reusing the
`bar.shape` class. An empty series reports `gantt: empty series`
(reusing `bar.empty`). A task with `start < 0` or `end < start`
reports `gantt: invalid range` — a new code, `gantt.range`,
because a negative or inverted interval is a data error that
no other renderer models.

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

| Trigger                                                  | Inline message               |
| -------------------------------------------------------- | ---------------------------- |
| Raw string piped to non-`text`                           | `bar: use @ref`              |
| Unknown renderer name                                    | `unknown renderer: foo`      |
| `@ref` not declared in frontmatter                       | `@missing not found`         |
| Bare `@ref` with no renderer                             | `@name: requires a renderer` |
| Renderer got wrong shape                                 | `hbar: expected [{x,y}]`     |
| Empty series for a bar renderer                          | `hbar: empty series`         |
| Any negative `y`                                         | `hbar: negative y`           |
| Duplicate `x` in series                                  | `hbar: duplicate x "a"`      |
| Non-numeric scalar in `\| progress`                      | `progress: expected number`  |
| OHLC inconsistency (`h < max(o,c)` or `l > min(o,c)`)    | `candlestick: invalid ohlc`  |
| Gantt range inconsistency (`start < 0` or `end < start`) | `gantt: invalid range`       |

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

## ANSI color

The Go CLI optionally emits 24-bit SGR escape sequences when rendering
ASCII output to a terminal. Color is opt-in semantic emphasis — it
highlights the primitives where up / down / warn carries meaning, and
leaves the rest alone. SVG and PNG have their own palettes (CSS custom
properties on the web, Go theme colors for raster); ANSI is
terminal-only and unrelated to either.

Colored primitives in v1: `candlestick` bodies, `progress` fills, and
inline `⚠ NAME: msg` warning rows. Every other primitive (`bar`,
`vbar`, `hist`, `sparkline`, `heatmap`, `step`, `gantt`, `banner`,
`text`) stays uncolored — they are comparison or trend viz, not
good-vs-bad signals, so semantic color would mislead more than it
helps. The JS / browser renderer has no ANSI path.

### `--color` flag

`--color=auto|always|never` controls whether escape sequences are
emitted. Default is `auto`.

Decision order (first match wins):

| Condition                                             | Color on? |
| ----------------------------------------------------- | --------- |
| `--color=never`                                       | no        |
| `--color=always`                                      | yes       |
| `--color=auto` + frontmatter `color: true`            | yes       |
| `--color=auto` + frontmatter `color: false`           | no        |
| `--color=auto` + `NO_COLOR` env set                   | no        |
| `--color=auto` + `COLORTERM` is `truecolor` / `24bit` | yes       |
| `--color=auto` + stdout is an interactive TTY         | yes       |
| otherwise under `auto`                                | no        |

The `color:` frontmatter key lets a source file declare its own
preference — useful when a specific chart is meant to be colored
(or pointedly plain) regardless of the reader's shell setup. The
CLI flag still wins: `--color=never` overrides `color: true` and
vice versa, so CI and scripted callers retain a deterministic
switch.

`NO_COLOR=1` wins whenever mode is `auto` and the frontmatter is
silent, per <https://no-color.org/>. `COLORTERM=truecolor` (or
`24bit`) is a positive signal that short-circuits the TTY check —
it covers `nohup` / tmux-outer-pipe cases where isatty would be
false but the user clearly wants color. Otherwise the TTY check is
enough: most modern terminals (iTerm, Alacritty, Kitty,
Terminal.app, Windows Terminal, GNOME Terminal) parse 24-bit SGR
cleanly whether or not they advertise `COLORTERM`. On Windows,
`auto` falls back to off because stdlib TTY detection is flaky
there; pass `--color=always` explicitly to force color on Windows
terminals that do support it.

`theme: ascii` suppresses color unconditionally, even under
`--color=always`. ASCII theme is the plain-text opt-out; choosing it
says "no Unicode, no color," and the CLI honours that.

### Semantic palette

<!-- markdownlint-disable MD060 -->

| Kind        | Glyphs colored                            | SGR                      |
| ----------- | ----------------------------------------- | ------------------------ |
| bull / up   | candlestick bull body, progress ≥80% fill | `\x1b[38;2;64;160;64m`   |
| bear / down | candlestick bear body, progress <40% fill | `\x1b[38;2;200;64;64m`   |
| neutral     | candlestick doji, progress 40–79% fill    | `\x1b[38;2;200;170;32m`  |
| warn        | whole inline `⚠ NAME: msg` row           | `\x1b[1;38;2;200;64;64m` |

<!-- markdownlint-enable MD060 -->

Each colored span is terminated by a `\x1b[0m` reset. The progress
thresholds are applied per-row: a multi-row `@ref | progress` can mix
bull / neutral / bear fills in the same box.

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
