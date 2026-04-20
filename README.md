# Pylon

> Draw your data and flow in plain text, by human or AI.

**Pylon** is the simple, human-friend and easy-to-use tool to draw your data and flow in plain text, by human or AI.
It focus on save your data and chart in plain text, managed by human or AI, and can be easily shared and collaborated.

## Syntax

Pylon uses a simple syntax to define nodes and edges in a graph. Here are some examples:

| Name      | Syntax   | Description                                             |
| --------- | -------- | ------------------------------------------------------- |
| node      | [ NAME ] | Define a node with the given NAME.                      |
| edge      | ->       | Define an edge from the previous node to the next node. |
| alignment | -        | Define an alignment between nodes.                      |
| data      | @NAME    | Load data from the given NAME (e.g. API or JSON).       |
| render    | \| NAME  | Render the data with the given NAME (e.g. chart type).  |

### Metadata

As the markdown, you can specified the metadata in the header of the graph, with the three dashes `---` to separate
the metadata and the graph content. The metadata can be used to specify the size, theme, and other properties of the
graph, more metadata will be added in the future.

| Key   | Value Type | Description                                              |
| ----- | ---------- | -------------------------------------------------------- |
| size  | int x int  | Define the size of the graph                             |
| theme | string     | Define the theme of the graph (e.g. ascii, dark, light). |

### Render

The pylon can render the graph in different style, like plain text or chart. The render is defined by the `|` symbol,
and the name of the render (e.g. chart type) is specified after the `|` symbol. It may just render the error message
if the render type is not supported, and more render types will be added in the future.

| Name      | Description                            |
| --------- | -------------------------------------- |
| (default) | Render the graph in plain with border. |
| text      | Render the graph in plain.             |
| bar       | Render the graph as a bar chart.       |
| line      | Render the graph as a line chart.      |

## Example

### Single Text

The simple text to show current time:

```pylon
---
size: 80x80
theme: simple
---
[ - 12:00 - ]
```

this graph will generate the time as the single node, and the alignment will make it centered, with
the metadata to set the size and theme of the graph.

```txt
    ┌───────────────────┐
    │                   │
    |       12:00       │
    │                   │
    └───────────────────┘
```

### Node with Text and Chart

The pylon also can load data from API or JSON to draw the chart, for example, the simple bar chart:

```pylon
---
size: 80x80
theme: simple

data:
  counter:
    - x: 1
      y: 10
    - x: 2
      y: 20
    - x: 3
      y: 15
---

[ -
  [ Counter | text ]
  [ - @counter | bar - ]
- ]
```

This example show a node that contains the text and the bar chart.

The first node is the plain text node without border, and the second node is the line chart node that load data
from the `counter` data source. The alignment will make the text and chart centered in the node, and the metadata
will set the size and theme of the graph.

```txt
    ┌───────────────────┐
    │                   │
    |   Counter         │
    |   ┌─────────────┐ │
    | 1 | ███    (10) | |
    | 2 | ██████ (20) | |
    | 3 | ████   (15) | |
    |   └─────────────┘ │
    └───────────────────┘
```

## Diagram

The pylon can also draw the diagram with the edge and alignment, for example, the simple flow chart:

```pylon
---
size: 80x80
theme: simple
---
[ Start ] -> [ Process ] -> [ End ]
```

This example show a simple flow chart with three nodes and two edges.

```txt
    ┌───────────┐     ┌─────────┐     ┌─────┐
    │  Start    │ --> │ Process │ --> │ End │
    └───────────┘     └─────────┘     └─────┘
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

## DDD (Dream-Driven Development)

This project follows the DDD (Dream-Driven Development) methodology, which means the project
is driven by what I envision.

All features are based on my needs and my dreams.
