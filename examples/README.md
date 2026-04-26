# Pylon examples

Small self-contained `.pylon` files showing individual features.

Render any example by piping into the Pylon WYSIWYG demo (`make -C src/js run`)
or by copying the source into a `<pylon-chart>` element.

<!-- markdownlint-disable MD013 -->

| File                                               | Shows                                                                   |
| -------------------------------------------------- | ----------------------------------------------------------------------- |
| [hello.pylon](hello.pylon)                         | Simplest bordered node                                                  |
| [flow-chart.pylon](flow-chart.pylon)               | Right-arrow edges between three nodes                                   |
| [alignment.pylon](alignment.pylon)                 | Centered / right / left alignment markers                               |
| [nested-nodes.pylon](nested-nodes.pylon)           | Bordered container with two nested nodes                                |
| [ref-self-loop.pylon](ref-self-loop.pylon)         | Same-row `&ref` arc under the declaration                               |
| [ref-cross-row.pylon](ref-cross-row.pylon)         | Cross-row `&ref` routed through the right gutter                        |
| [edge-labels.pylon](edge-labels.pylon)             | Labelled edge between two nodes                                         |
| [chart-hbar.pylon](chart-hbar.pylon)               | Horizontal bar chart from `data:` frontmatter                           |
| [chart-vbar.pylon](chart-vbar.pylon)               | Vertical bar chart from `data:` frontmatter                             |
| [banner.pylon](banner.pylon)                       | Block-letter banner of a literal string                                 |
| [banner-monospace.pylon](banner-monospace.pylon)   | Uniform-width `█` banner via `banner: monospace`                        |
| [banner-multi-font.pylon](banner-multi-font.pylon) | Three banners in one image: `digital`, frontmatter default, `monospace` |
| [progress.pylon](progress.pylon)                   | Progress bars with `%` labels from a `@ref` series                      |
| [heatmap.pylon](heatmap.pylon)                     | 2D matrix of ramp glyphs from a `@ref` series                           |
| [sparkline.pylon](sparkline.pylon)                 | Inline trend row of ramp glyphs from a `@ref`                           |
| [candlestick.pylon](candlestick.pylon)             | OHLC candles with bull/bear/doji bodies and wicks                       |
| [hist.pylon](hist.pylon)                           | Gap-less 8-row histogram of pre-binned counts                           |
| [step.pylon](step.pylon)                           | 5-row cumulative step line with corner connectors                       |
| [gantt.pylon](gantt.pylon)                         | Task bars over a shared horizontal budget                               |
| [theme-ascii.pylon](theme-ascii.pylon)             | `theme: ascii` swaps Unicode glyphs for plain ASCII                     |
| [line.pylon](line.pylon)                           | Multi-row continuous line plot with point markers                       |
| [area.pylon](area.pylon)                           | Filled-below-curve trend using the heatmap shade ramp                   |
| [scatter.pylon](scatter.pylon)                     | 2D scatter plot of discrete points                                      |
| [sbar.pylon](sbar.pylon)                           | Horizontal stacked bars from a multi-value series                       |
| [bullet.pylon](bullet.pylon)                       | Target+actual+bands dashboard chart                                     |
| [box.pylon](box.pylon)                             | 5-number-summary boxplot (whiskers + IQR + median)                      |
| [calendar.pylon](calendar.pylon)                   | GitHub-style year-of-days activity heatmap                              |
| [showcase.pylon](showcase.pylon)                   | Every chart + diagram primitive; opts into ANSI color via `color: true` |

<!-- markdownlint-enable MD013 -->

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
┌────────────────────────────┐
│   apples │ █████      (10) │
│  bananas │ ██████████ (20) │
│ cherries │ ████████   (15) │
└────────────────────────────┘
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

## banner-monospace.pylon

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
┌─────────────────────────────────┐
│  spec ████████████████████ 100% │
│  code █████████████████░░░  85% │
│ tests ████████████░░░░░░░░  60% │
│  docs ████████░░░░░░░░░░░░  40% │
└─────────────────────────────────┘
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
┌───────────┐
│ Mon ░░▒▓█ │
│ Tue  ▒▓█▒ │
│ Wed ░▓█▓░ │
│ Thu  ░▒░  │
└───────────┘
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
┌────────────┐
│ ▃▆▁▇▄█▂▅▆▃ │
└────────────┘
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
┌───────────────────────┐
│          │        │   │
│    │     ▒  █     │   │
│ │  █     ▒  █     ▒   │
│ ▒  █  │  ▒  █     ▒   │
│ ▒  █  ─  │  █  │  ▒   │
│ ▒     │     │  ─  │   │
│ │              │  │   │
│ │                 │   │
│ MonTueWedThuFriSatSun │
└───────────────────────┘
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
┌─────────────┐
│          ┌─ │
│        ┌─┘  │
│      ┌─┘    │
│  ┌───┘      │
│ ─┘          │
└─────────────┘
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
┌───────────────────────────┐
│ spec █████                │
│ code    █████████         │
│ test           ██████     │
│ ship               ██████ │
└───────────────────────────┘
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

## banner-multi-font.pylon

```pylon
---
banner: mini
---
[ HI | banner:digital ]
[ HI | banner ]
[ HI | banner:monospace ]
```

```txt
     ┌──────────────────┐
     │   █   █   █      │
     │   █   █   █      │
     │   █████   █      │
     │   █   █   █      │
     │   █   █   █      │
     │   █   █   █      │
     └──────────────────┘
      ┌────────────────┐
      │   █  █ ███     │
      │   █  █  █      │
      │   ████  █      │
      │   █  █  █      │
      │   █  █  █      │
      │   █  █ ███     │
      └────────────────┘
   ┌──────────────────────┐
   │   ██  ██  ██████     │
   │   ██  ██    ██       │
   │   ██████    ██       │
   │   ██  ██    ██       │
   │   ██  ██    ██       │
   │   ██  ██  ██████     │
   └──────────────────────┘
```

## line.pylon

```pylon
---
data:
  series:
    - x: 1
      y: 1
    - x: 2
      y: 4
    - x: 3
      y: 2
    - x: 4
      y: 5
    - x: 5
      y: 3
---
[ @series | line ]
```

```txt
┌───────────────┐
│         ●     │
│     ●    │    │
│      │ │      │
│    │      ●   │
│       ●       │
│               │
│   ●           │
└───────────────┘
```

## area.pylon

```pylon
---
data:
  series:
    - x: 1
      y: 1
    - x: 2
      y: 4
    - x: 3
      y: 2
    - x: 4
      y: 5
    - x: 5
      y: 3
---
[ @series | area ]
```

```txt
┌───────────┐
│      █    │
│      ▓    │
│    █ ▒    │
│    ▓ ░█   │
│    ▒ ░▓   │
│    ░█░▒   │
│    ░▓░░   │
└───────────┘
```

## scatter.pylon

```pylon
---
data:
  series:
    - x: 1
      y: 5
    - x: 3
      y: 2
    - x: 5
      y: 7
    - x: 7
      y: 1
    - x: 9
      y: 4
---
[ @series | scatter ]
```

```txt
┌────────────────────────────────────┐
│                  ●                 │
│                                    │
│   ●                                │
│                                ●   │
│                                    │
│          ●                         │
│                         ●          │
└────────────────────────────────────┘
```

## sbar.pylon

```pylon
---
data:
  series:
    - x: Q1
      y: [10, 5, 3, 2]
    - x: Q2
      y: [8, 7, 4, 1]
    - x: Q3
      y: [12, 6, 5, 2]
---
[ @series | sbar ]
```

```txt
┌──────────────────────────┐
│   Q1 │ ░░░░▒▒▓█   (20)   │
│   Q2 │ ░░░▒▒▒▓▓   (20)   │
│   Q3 │ ░░░░░▒▒▓▓█ (25)   │
└──────────────────────────┘
```

## bullet.pylon

```pylon
---
data:
  series:
    - x: Sales
      y: 70
      target: 100
    - x: Profit
      y: 45
      target: 60
    - x: Stock
      y: 25
      target: 80
---
[ @series | bullet ]
```

```txt
┌────────────────────────────────────────┐
│    Sales │ ██████████████▓▓▓▓▓◆ (70)   │
│   Profit │ █████████▒▒▒◆▓▓▓▓▓▓▓ (45)   │
│    Stock │ █████░▒▒▒▒▒▒▒▓▓▓◆▓▓▓ (25)   │
└────────────────────────────────────────┘
```

## box.pylon

```pylon
---
data:
  series:
    - x: Mon
      min: 5
      q1: 12
      med: 18
      q3: 24
      max: 30
    - x: Tue
      min: 8
      q1: 14
      med: 20
      q3: 28
      max: 35
    - x: Wed
      min: 3
      q1: 10
      med: 16
      q3: 22
      max: 28
---
[ @series | box ]
```

```txt
┌────────────────────────────────┐
│   Mon │  ────┤▓▓▓█▓▓├────      │
│   Tue │    ────┤▓▓█▓▓▓▓├────   │
│   Wed │ ────┤▓▓▓█▓▓├────       │
└────────────────────────────────┘
```

## calendar.pylon

```pylon
---
data:
  series:
    - date: "2026-01-05"
      y: 3
    - date: "2026-01-12"
      y: 5
    - date: "2026-01-15"
      y: 1
    - date: "2026-01-20"
      y: 4
    - date: "2026-02-01"
      y: 7
---
[ @series | calendar ]
```

```txt
┌───────────────┐
│   Sun     █   │
│   Mon ▒▓      │
│   Tue   ▒     │
│   Wed         │
│   Thu  ░      │
│   Fri         │
│   Sat         │
└───────────────┘
```
