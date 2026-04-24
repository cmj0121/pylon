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
| [sparkline.pylon](sparkline.pylon)         | Inline trend row of ramp glyphs from a `@ref`       |
| [candlestick.pylon](candlestick.pylon)     | OHLC candles with bull/bear/doji bodies and wicks   |
| [hist.pylon](hist.pylon)                   | Gap-less 8-row histogram of pre-binned counts       |
| [step.pylon](step.pylon)                   | 5-row cumulative step line with corner connectors   |
| [gantt.pylon](gantt.pylon)                 | Task bars over a shared horizontal budget           |
| [theme-ascii.pylon](theme-ascii.pylon)     | `theme: ascii` swaps Unicode glyphs for plain ASCII |
| [showcase.pylon](showcase.pylon)           | Every chart + diagram primitive in one source file  |

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
┌──────────────────────────┐
│  apples │ █████      (10)│
│ bananas │ ██████████ (20)│
│cherries │ ████████   (15)│
└──────────────────────────┘
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
┌────────────┐
│     █      │
│     █      │
│     █   █  │
│     █   █  │
│     █   █  │
│ █   █   █  │
│ █   █   █  │
│ █   █   █  │
│ █   █   █  │
│ █   █   █  │
│ 1   2   3  │
│(10)(20)(15)│
└────────────┘
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
┌───────────────────────────────┐
│ spec ████████████████████ 100%│
│ code █████████████████░░░  85%│
│tests ████████████░░░░░░░░  60%│
│ docs ████████░░░░░░░░░░░░  40%│
└───────────────────────────────┘
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
┌─────────┐
│Mon ░░▒▓█│
│Tue  ▒▓█▒│
│Wed ░▓█▓░│
│Thu  ░▒░ │
└─────────┘
```

## sparkline.pylon

```pylon
---
data:
  latency:
    - x: 1
      y: 42
    - x: 2
      y: 58
    - x: 3
      y: 31
    - x: 4
      y: 65
    - x: 5
      y: 49
    - x: 6
      y: 73
    - x: 7
      y: 38
    - x: 8
      y: 52
    - x: 9
      y: 61
    - x: 10
      y: 45
---
[ @latency | sparkline ]
```

```txt
┌──────────┐
│▃▆▁▇▄█▂▅▆▃│
└──────────┘
```

## candlestick.pylon

```pylon
---
data:
  week:
    - x: Mon
      o: 2
      h: 5
      l: 0
      c: 4
    - x: Tue
      o: 5
      h: 6
      l: 3
      c: 3
    - x: Wed
      o: 3
      h: 4
      l: 2
      c: 3
    - x: Thu
      o: 4
      h: 7
      l: 3
      c: 6
    - x: Fri
      o: 6
      h: 6
      l: 2
      c: 3
    - x: Sat
      o: 2
      h: 3
      l: 1
      c: 2
    - x: Sun
      o: 3
      h: 7
      l: 0
      c: 5
---
[ @week | candlestick ]
```

```txt
┌─────────────────────┐
│         │        │  │
│   │     ▒  █     │  │
││  █     ▒  █     ▒  │
│▒  █  │  ▒  █     ▒  │
│▒  █  ─  │  █  │  ▒  │
│▒     │     │  ─  │  │
││              │  │  │
││                 │  │
│MonTueWedThuFriSatSun│
└─────────────────────┘
```

## hist.pylon

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
┌────┐
│ ██ │
│ ██ │
│ ██ │
│ ██ │
│ ███│
│ ███│
│████│
│████│
│MISP│
└────┘
```

## step.pylon

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
┌───────────┐
│         ┌─│
│       ┌─┘ │
│     ┌─┘   │
│ ┌───┘     │
│─┘         │
└───────────┘
```

## gantt.pylon

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
┌─────────────────────────┐
│spec █████               │
│code    █████████        │
│test           ██████    │
│ship               ██████│
└─────────────────────────┘
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
