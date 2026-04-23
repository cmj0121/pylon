# Pylon examples

Small self-contained `.pylon` files showing individual features.

Render any example by piping into the Pylon WYSIWYG demo (`make -C src/js run`)
or by copying the source into a `<pylon-chart>` element.

| File                                       | Shows                                               |
| ------------------------------------------ | --------------------------------------------------- |
| [hello.pylon](hello.pylon)                 | Simplest bordered node                              |
| [flow-chart.pylon](flow-chart.pylon)       | Right-arrow edges between three nodes               |
| [alignment.pylon](alignment.pylon)         | Centered / right / left alignment markers           |
| [nested-nodes.pylon](nested-nodes.pylon)   | Bordered container with two nested nodes            |
| [ref-self-loop.pylon](ref-self-loop.pylon) | Same-row `&ref` arc under the declaration           |
| [ref-cross-row.pylon](ref-cross-row.pylon) | Cross-row `&ref` routed through the right gutter    |
| [edge-labels.pylon](edge-labels.pylon)     | Labelled edge between two nodes                     |
| [chart-hbar.pylon](chart-hbar.pylon)       | Horizontal bar chart from `data:` frontmatter       |
| [chart-vbar.pylon](chart-vbar.pylon)       | Vertical bar chart from `data:` frontmatter         |
| [banner.pylon](banner.pylon)               | Block-letter banner of a literal string             |
| [progress.pylon](progress.pylon)           | Progress bars with `%` labels from a `@ref` series  |
| [heatmap.pylon](heatmap.pylon)             | 2D matrix of ramp glyphs from a `@ref` series       |
| [theme-ascii.pylon](theme-ascii.pylon)     | `theme: ascii` swaps Unicode glyphs for plain ASCII |

## hello.pylon

```pylon
[- Hello -]
```

```txt
┌───────────┐
│   Hello   │
└───────────┘
```

## flow-chart.pylon

```pylon
[ Start ] -> [ Process ] -> [ End ]
```

```txt
┌───────────┐  ┌─────────────┐  ┌─────────┐
│   Start   │─▶│   Process   │─▶│   End   │
└───────────┘  └─────────────┘  └─────────┘
```

## alignment.pylon

```pylon
[- center -]
[- right ]
[ left -]
```

```txt
   ┌────────────┐
   │   center   │
   └────────────┘
   ┌───────────┐
   │     right │
   └───────────┘
    ┌──────────┐
    │ left     │
    └──────────┘
```

## nested-nodes.pylon

```pylon
[ Outer
  [ inner1 ]
  [ inner2 ]
]
```

```txt
┌────────────────────┐
│       Outer        │
│   ┌────────────┐   │
│   │   inner1   │   │
│   └────────────┘   │
│   ┌────────────┐   │
│   │   inner2   │   │
│   └────────────┘   │
└────────────────────┘
```

## ref-self-loop.pylon

```pylon
[a :: X] -> &X
```

```txt
┌───────┐
│   a   │
└───────┘
  │   ▲
  └───┘
```

## ref-cross-row.pylon

```pylon
[a :: X]
[b] -> &X
```

```txt
   ┌───────┐
   │   a   │◀────┐
   └───────┘     │
   ┌───────┐     │
   │   b   │─────┘
   └───────┘
```

## edge-labels.pylon

```pylon
[ Alice ] -- ( friend ) --> [ Bob ]
```

```txt
┌───────────┐               ┌─────────┐
│   Alice   │──  friend  ──▶│   Bob   │
└───────────┘               └─────────┘
```

## chart-hbar.pylon

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
┌──────────────────────────────────┐
│     apples │ █████      (10) │   │
│    bananas │ ██████████ (20) │   │
│   cherries │ ████████   (15) │   │
└──────────────────────────────────┘
```

## chart-vbar.pylon

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

## banner.pylon

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

## progress.pylon

```pylon
---
data:
  release:
    - x: spec
      y: 100
    - x: code
      y: 85
    - x: tests
      y: 60
    - x: docs
      y: 40
---
[ @release | progress ]
```

```txt
┌─────────────────────────────────────┐
│    spec ████████████████████ 100%   │
│    code █████████████████░░░  85%   │
│   tests ████████████░░░░░░░░  60%   │
│    docs ████████░░░░░░░░░░░░  40%   │
└─────────────────────────────────────┘
```

## heatmap.pylon

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
┌───────────────┐
│   Mon ░░▒▓█   │
│   Tue  ▒▓█▒   │
│   Wed ░▓█▓░   │
│   Thu  ░▒░    │
└───────────────┘
```

## theme-ascii.pylon

```pylon
---
theme: ascii
---
[ Start ] -> [ Process ] -> [ End ]
```

```txt
+-----------+  +-------------+  +---------+
|   Start   |->|   Process   |->|   End   |
+-----------+  +-------------+  +---------+
```
