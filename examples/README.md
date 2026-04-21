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
