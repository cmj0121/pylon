// Pylon -- tiny WYSIWYG scaffold
//
// Registers a <pylon-chart> custom element. Renders its source
// (textContent or the `src` attribute) in one of three backends:
//
//   <pylon-chart>[- Hello -]</pylon-chart>                  // ASCII (default)
//   <pylon-chart format="svg">[- Hello -]</pylon-chart>     // inline SVG
//   <pylon-chart format="png">[- Hello -]</pylon-chart>     // canvas -> <img>
//
// (The HTML spec requires custom element names to contain a hyphen,
// hence `pylon-chart` rather than `pylon`.)
//
// Add the `wysiwyg` attribute to turn the element into a split-pane
// editor: plaintext textarea on the left, rendered output on the right
// with a format dropdown (ASCII / SVG / PNG). The dropdown overrides
// the `format` attribute until the attribute is changed again.
//
// The parser is a stub for now: the source text becomes the single
// label on a drawn box. Wire in the real parser once the grammar is
// settled.

(() => {
  // ---- stub parser ------------------------------------------------------
  // Replace with the real Pylon parser later. For now, any input becomes
  // a single-box AST with an optional frontmatter block:
  //   ---
  //   size:  WxH          # outer dimensions in cells
  //   theme: IDENT        # reserved -- applied by later commits
  //   ---
  //   [ ... ]             # bordered node
  //   ( ... )             # borderless node
  // Leading and trailing '-' inside the brackets are alignment markers
  // and are stripped from the label. Empty input falls back to the
  // default example.
  const DEFAULT_EXAMPLE = "[- Pylon WYSIWYG -]";
  // Default outer dimensions when the author hasn't declared a size
  // in the frontmatter. Chosen to match the WYSIWYG preview pane and
  // to give flow chains a generous canvas before the auto-wrap-to-
  // vertical kicks in.
  const DEFAULT_SIZE = { w: 60, h: 40 };

  const FRONTMATTER_RE = /^---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*\r?\n?/;

  const parseFrontmatter = (text) => {
    const meta = {};
    for (const line of text.split(/\r?\n/)) {
      const m = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*?)\s*$/);
      if (!m) continue;
      const [, key, raw] = m;
      if (key === "size") {
        const s = raw.match(/^(\d+)\s*[xX]\s*(\d+)$/);
        if (s) meta.size = { w: parseInt(s[1], 10), h: parseInt(s[2], 10) };
      } else if (key === "theme") {
        meta.theme = raw;
      }
    }
    return meta;
  };

  // Find the matching close bracket for s[start] (which must be '[' or '(').
  // Counts only the same bracket kind -- well-formed input has () and [] pairs
  // nested independently.
  const findMatching = (s, start) => {
    const open = s[start];
    const close = open === "[" ? "]" : open === "(" ? ")" : null;
    if (!close) return -1;
    let depth = 0;
    for (let i = start; i < s.length; i++) {
      if (s[i] === open) depth++;
      else if (s[i] === close) {
        depth--;
        if (depth === 0) return i;
      }
    }
    return -1;
  };

  // Extract leading / trailing '-' alignment springs from an inner string.
  // The dash must be flush with the bracket -- '[-' and '-]' count,
  // '[ -' and '- ]' do not (those are literal dashes inside the label).
  const ALIGN_LEFT_RE = /^-\s+/;
  const ALIGN_RIGHT_RE = /\s+-$/;
  const extractAlign = (inner) => {
    const hasLeft = ALIGN_LEFT_RE.test(inner);
    const hasRight = ALIGN_RIGHT_RE.test(inner);
    let align = "center";
    if (hasLeft && !hasRight) align = "right";
    else if (!hasLeft && hasRight) align = "left";
    const clean = inner
      .replace(ALIGN_LEFT_RE, "")
      .replace(ALIGN_RIGHT_RE, "")
      .trim();
    return { align, clean };
  };

  // Flow-edge tokens recognised between sibling nodes on the same line:
  //
  //   ->         length=1 right
  //   <-         length=1 left
  //   <->        length=1 bidirectional
  //   -->        length=2 right
  //   <-->       length=2 bidirectional  (arrow on both sides)
  //   --->       length=3 right
  //   ...and so on. The general form is <?-+>? with at least one arrow.
  //   Pure dashes with no arrow head stay as literal text (the labelled
  //   edge form '-- ( label ) -->' lands in a later commit).
  //
  // Note on '-': inside brackets flush with the wall (e.g. '[- x -]') a
  // lone dash is an alignment spring (unchanged). Between siblings on a
  // line, '-' is an edge-line fragment and never aligns. The older
  // README hint of "'-' defines alignment between nodes" is dropped --
  // alignment between nodes is not a DSL feature.
  const EDGE_RE = /^<?-+>?/;

  // Post-pass over a line's items to stitch a labelled-edge pattern:
  //
  //   [ A ] -- ( friend ) --> [ B ]
  //         ^^^^^^^^^^^^^^^^^^^^^^ three tokens merged into one edge
  //
  // The pattern is (dashes-only-text OR edge) + borderless-node + (dashes-only-text OR edge).
  // At least one of the outer pieces must be an arrow-bearing edge,
  // otherwise there is no direction (just dashes around a label is
  // ambiguous and we leave it as text).
  const DASH_ONLY_RE = /^-+$/;
  const stitchLabeledEdges = (items) => {
    const out = [];
    let i = 0;
    const isDashes = (x) =>
      x && x.type === "text" && DASH_ONLY_RE.test(x.content);
    const isEdge = (x) => x && x.type === "edge";
    const isBorderlessBox = (x) =>
      x && x.type === "box" && x.bordered === false;
    while (i < items.length) {
      const a = items[i];
      const b = items[i + 1];
      const c = items[i + 2];
      if (
        (isDashes(a) || isEdge(a)) &&
        isBorderlessBox(b) &&
        (isDashes(c) || isEdge(c)) &&
        (isEdge(a) || isEdge(c))
      ) {
        const hasL =
          isEdge(a) && (a.direction === "left" || a.direction === "both");
        const hasR =
          isEdge(c) && (c.direction === "right" || c.direction === "both");
        const direction = hasL && hasR ? "both" : hasL ? "left" : "right";
        const leftLen = a.type === "text" ? a.content.length : a.length;
        const rightLen = c.type === "text" ? c.content.length : c.length;
        out.push({
          type: "edge",
          direction,
          leftLen,
          rightLen,
          label: b,
        });
        i += 3;
        continue;
      }
      out.push(a);
      i++;
    }
    return out;
  };

  // Split the inner content of a box into an ordered list of items.
  // Items are child nodes, text runs, edge tokens, or -- when a source
  // line contains at least one edge and two or more other items -- a
  // row wrapper that groups the line's items for horizontal layout.
  // Newlines separate items vertically.
  const parseItems = (s) => {
    const items = [];
    let lineItems = [];
    let textBuf = "";

    const flushText = () => {
      const t = textBuf.trim();
      if (t) lineItems.push({ type: "text", content: t });
      textBuf = "";
    };

    const flushLine = () => {
      flushText();
      if (lineItems.length === 0) return;
      lineItems = stitchLabeledEdges(lineItems);
      const hasEdge = lineItems.some((it) => it.type === "edge");
      if (hasEdge) {
        items.push({ type: "row", items: lineItems });
      } else {
        // No edge on this line -- flatten items into the parent stack
        // so existing vertical-stack behaviour is preserved.
        for (const it of lineItems) items.push(it);
      }
      lineItems = [];
    };

    let i = 0;
    while (i < s.length) {
      const c = s[i];
      if (c === "[" || c === "(") {
        flushText();
        const end = findMatching(s, i);
        if (end < 0) {
          textBuf += c;
          i++;
          continue;
        }
        lineItems.push(parseBracketedNode(s.slice(i, end + 1)));
        i = end + 1;
      } else if (c === "\n") {
        flushLine();
        i++;
      } else if (c === "<" || c === "-") {
        const match = s.slice(i).match(EDGE_RE);
        const token = match ? match[0] : "";
        const hasLeft = token.startsWith("<");
        const hasRight = token.endsWith(">");
        if (token && (hasLeft || hasRight)) {
          flushText();
          const dashes = token.length - (hasLeft ? 1 : 0) - (hasRight ? 1 : 0);
          const direction =
            hasLeft && hasRight ? "both" : hasRight ? "right" : "left";
          lineItems.push({
            type: "edge",
            direction,
            length: Math.max(1, dashes),
          });
          i += token.length;
          continue;
        }
        textBuf += c;
        i++;
      } else {
        textBuf += c;
        i++;
      }
    }
    flushLine();
    return items;
  };

  // Parse a single box, given the slice that starts with '[' or '(' and
  // ends with the matching ']' or ')'.
  const parseBracketedNode = (s) => {
    const open = s[0];
    const bordered = open === "[";
    const close = bordered ? "]" : ")";
    if (!s.endsWith(close)) {
      return {
        type: "box",
        bordered: true,
        align: "center",
        items: [{ type: "text", content: s.trim() }],
      };
    }
    const inner = s.slice(1, -1);
    const { align, clean } = extractAlign(inner);
    return {
      type: "box",
      bordered,
      align,
      items: parseItems(clean),
    };
  };

  const parse = (source) => {
    const src = (source ?? "").trim();
    if (!src) return parse(DEFAULT_EXAMPLE);

    let meta = {};
    let body = src;
    const fm = src.match(FRONTMATTER_RE);
    if (fm) {
      meta = parseFrontmatter(fm[1]);
      body = src.slice(fm[0].length).trim();
      if (!body) body = DEFAULT_EXAMPLE;
    }

    // Top-level is conceptually always an implicit '( ... )' -- a
    // borderless container -- so multiple sibling nodes stack:
    //
    //   [Hello]
    //   (World)
    //
    // parses as two top-level items. When the user writes exactly one
    // bracketed node we use it directly so SVG / PNG can paint a real
    // vector rect; for anything else (bare text, a row of chained
    // nodes, or multiple items) we wrap in the implicit root. Bare
    // single text gets a bordered wrap for the legacy "type some text
    // and get a box" behaviour; everything else is borderless.
    const items = parseItems(body);
    let root;
    if (items.length === 1 && items[0].type === "box") {
      root = items[0];
    } else {
      const bareText = items.length === 1 && items[0].type === "text";
      root = {
        type: "box",
        bordered: bareText,
        align: "center",
        items,
      };
    }
    if (!meta.size) meta.size = { ...DEFAULT_SIZE };
    root.meta = meta;
    return root;
  };

  // ---- display width ----------------------------------------------------
  // Count terminal/monospace-column width so borders line up when the
  // label contains CJK, fullwidth punctuation, or emoji. Based on the
  // Unicode East Asian Width property: the wide ranges listed here count
  // as 2, zero-width ranges (combining marks, variation selectors,
  // zero-width joiners) count as 0, everything else counts as 1.
  // Compound emoji (ZWJ sequences, flags, skin-tone modifiers) are
  // counted conservatively wide -- the box may come out a cell loose.
  const EAW_WIDE = [
    [0x1100, 0x115f],
    [0x2329, 0x232a],
    [0x2e80, 0x303e],
    [0x3041, 0x33ff],
    [0x3400, 0x4dbf],
    [0x4e00, 0x9fff],
    [0xa000, 0xa4cf],
    [0xac00, 0xd7a3],
    [0xf900, 0xfaff],
    [0xfe30, 0xfe4f],
    [0xff00, 0xff60],
    [0xffe0, 0xffe6],
    [0x1f300, 0x1f64f],
    [0x1f680, 0x1f6ff],
    [0x1f900, 0x1f9ff],
    [0x20000, 0x2fffd],
    [0x30000, 0x3fffd],
  ];

  const EAW_ZERO = [
    [0x0300, 0x036f],
    [0x0483, 0x0489],
    [0x1ab0, 0x1aff],
    [0x1dc0, 0x1dff],
    [0x200b, 0x200f],
    [0x2028, 0x202e],
    [0x20d0, 0x20ff],
    [0xfe00, 0xfe0f],
    [0xfe20, 0xfe2f],
    [0xfeff, 0xfeff],
  ];

  const inRanges = (cp, ranges) => {
    for (const [lo, hi] of ranges) {
      if (cp < lo) return false;
      if (cp <= hi) return true;
    }
    return false;
  };

  const charWidth = (cp) => {
    if (cp < 0x20 || (cp >= 0x7f && cp < 0xa0)) return 0;
    if (inRanges(cp, EAW_ZERO)) return 0;
    if (inRanges(cp, EAW_WIDE)) return 2;
    return 1;
  };

  const displayWidth = (str) => {
    let w = 0;
    for (const ch of str) w += charWidth(ch.codePointAt(0));
    return w;
  };

  // ---- layout / row rendering ------------------------------------------
  // Recursively reduce an AST into a flat array of text rows. Each row is
  // a string of display cells ready to paint. Nested boxes render as their
  // own multi-row box (ASCII glyphs + alignment); borderless wrappers just
  // pass their items' rows through. Every backend (ASCII / SVG / PNG)
  // consumes these rows so they all agree on shape and alignment.
  //
  //  - natural-pad  3 cells on each side of content in a bordered box
  //  - cell -> px   10 wide, 20 tall for SVG / PNG
  const NATURAL_PAD = 3;
  const CELL_PX_W = 10;
  const CELL_PX_H = 20;

  // Pad a single row (string) to targetW display cells, placing the row
  // according to align ('left' | 'center' | 'right'). Reserve minPad
  // cells on the pushed-against edge so left / right alignment keeps
  // the content off the border wall unless the slack is too tight.
  const padRow = (row, targetW, align, minPad = 1) => {
    const rowW = displayWidth(row);
    const slack = Math.max(0, targetW - rowW);
    if (slack === 0) return row;
    if (slack <= 2 * minPad || align === "center") {
      const l = Math.floor(slack / 2);
      return " ".repeat(l) + row + " ".repeat(slack - l);
    }
    if (align === "right") {
      return " ".repeat(slack - minPad) + row + " ".repeat(minPad);
    }
    return " ".repeat(minPad) + row + " ".repeat(slack - minPad);
  };

  // Truncate a row (if it's wider than targetW) along grapheme-ish
  // boundaries, counting display width. Cheap; good enough for MVP.
  const clipRow = (row, targetW) => {
    if (displayWidth(row) <= targetW) return row;
    let w = 0;
    let out = "";
    for (const ch of row) {
      const cw = charWidth(ch.codePointAt(0));
      if (w + cw > targetW) break;
      out += ch;
      w += cw;
    }
    return out;
  };

  // Distribute the "extra" vertical rows (beyond the content) above and
  // below the body so multi-row content sits vertically centered in a
  // larger sized box.
  const vertPad = (rows, targetH, contentW) => {
    const extra = Math.max(0, targetH - rows.length);
    if (extra === 0) return rows;
    const top = Math.floor(extra / 2);
    const bot = extra - top;
    const blank = " ".repeat(contentW);
    return [
      ...new Array(top).fill(blank),
      ...rows,
      ...new Array(bot).fill(blank),
    ];
  };

  // Render an item to its natural rows. Rows (horizontal chains) and
  // edges are lowered inside renderRowRows; they do not appear standalone.
  // When `maxW` is passed, a row wider than `maxW` wraps at item
  // boundaries into multiple stacked sub-rows.
  const renderItemRows = (item, bc, maxW) => {
    if (item.type === "text") return [item.content];
    if (item.type === "box") return renderBoxRows(item, bc);
    if (item.type === "row") return renderRowRows(item, bc, maxW);
    return [];
  };

  // Direction booleans for an edge. `head` is the rightward arrow
  // head (shown at the target end of a '→' or below a vertical '▼');
  // `tail` is the leftward arrow head.
  const edgeHeads = (direction) => ({
    head: direction === "right" || direction === "both",
    tail: direction === "left" || direction === "both",
  });

  // Render an edge's label rows on first access and cache them on the
  // edge node so the horizontal and vertical paths don't re-invoke
  // renderBoxRows for the same label.
  const labelRowsOf = (edge, bc) => {
    if (!edge._labelRows) edge._labelRows = renderBoxRows(edge.label, bc);
    return edge._labelRows;
  };

  // Format the text form of an edge token. Edges render at a fixed
  // length regardless of how many dashes the author typed, so '->',
  // '-->', and '--->' all produce the same arrow. Labelled edges use
  // two dashes on each side of the label so it breathes against the
  // adjacent box borders; unlabelled edges use a single dash plus the
  // arrow head.
  const edgeString = (edge, bc) => {
    const { head, tail } = edgeHeads(edge.direction);
    const leftArrow = tail ? bc.arrowL : "";
    const rightArrow = head ? bc.arrowR : "";
    if (edge.label) {
      const labelText = labelRowsOf(edge, bc)[0] ?? "";
      const dashes = bc.h.repeat(2);
      const pad = "  "; // two spaces on each side so the label breathes
      return leftArrow + dashes + pad + labelText + pad + dashes + rightArrow;
    }
    return leftArrow + bc.h + rightArrow;
  };

  // Render a row (horizontal chain of node / edge items) to display
  // rows. Node items render to their own rows; shorter nodes get
  // blank-row padding so every column shares the row's tallest
  // height; edges are placed on the midline row and blank strings on
  // the others.
  //
  // When `maxW` is given (the containing box has an explicit
  // meta.size.w) AND the horizontal layout would overflow, the row
  // rotates 90° -- nodes stack vertically, edges become down / up
  // arrows centered over the widest node's column. A long
  // '[A]->[B]->[C]->[D]' becomes
  //
  //   [A]
  //    ▼
  //   [B]
  //    ▼
  //   [C]
  //    ▼
  //   [D]
  //
  // instead of being clipped to fit the frame. Labelled edges keep
  // their label inline on its own row next to the arrow.
  const renderRowRows = (row, bc, maxW) => {
    const parts = row.items.map((it) => {
      if (it.type === "edge") {
        const text = edgeString(it, bc);
        return { kind: "edge", edge: it, text, width: displayWidth(text) };
      }
      const rows = renderItemRows(it, bc);
      const width = rows.reduce((m, r) => Math.max(m, displayWidth(r)), 0);
      return { kind: "block", rows, width };
    });

    const totalWidth = parts.reduce((sum, p) => sum + p.width, 0);
    if (maxW !== undefined && totalWidth > maxW) {
      return renderVerticalChain(parts, bc);
    }

    const h = parts.reduce(
      (m, p) => (p.kind === "block" ? Math.max(m, p.rows.length) : m),
      1,
    );
    const midline = Math.floor((h - 1) / 2);
    const columns = parts.map((p) => {
      if (p.kind === "edge") {
        const blank = " ".repeat(p.width);
        return Array.from({ length: h }, (_, r) =>
          r === midline ? p.text : blank,
        );
      }
      const blank = " ".repeat(p.width);
      const padded = p.rows.map((r) => padRow(r, p.width, "left"));
      const extra = h - padded.length;
      const top = Math.floor(extra / 2);
      const bot = extra - top;
      return [
        ...new Array(top).fill(blank),
        ...padded,
        ...new Array(bot).fill(blank),
      ];
    });
    const out = [];
    for (let r = 0; r < h; r++) {
      let line = "";
      for (const col of columns) line += col[r];
      out.push(line);
    }
    return out;
  };

  // Vertical-chain fallback for overflowing rows. Each node's own
  // rows are centered within the widest node's column; each edge
  // turns into a 1-or-3-row run carrying a down / up / both arrow
  // and, when present, the edge label.
  const renderVerticalChain = (parts, bc) => {
    const blocks = parts.filter((p) => p.kind === "block");
    if (blocks.length === 0) return [];
    const maxW = blocks.reduce((m, p) => Math.max(m, p.width), 1);
    const result = [];
    for (const p of parts) {
      if (p.kind === "block") {
        for (const r of p.rows) result.push(padRow(r, maxW, "center"));
        continue;
      }
      const { head: down, tail: up } = edgeHeads(p.edge.direction);
      // Top row carries the arrow head for an upward edge, otherwise
      // a '│' continuing the line down from the previous node. Bottom
      // row mirrors that for a downward edge.
      result.push(padRow(up ? "▲" : "│", maxW, "center"));
      if (p.edge.label) {
        result.push(padRow(labelRowsOf(p.edge, bc)[0] ?? "", maxW, "center"));
      }
      result.push(padRow(down ? "▼" : "│", maxW, "center"));
    }
    return result;
  };

  // Render a box AST to a flat array of rows. When targetW / targetH are
  // provided, the box is forced to those outer dimensions; otherwise it
  // auto-sizes to its content plus natural padding.
  //
  // A node gets natural side-padding when it's either:
  //   - bordered (always -- the padding sits between content and border)
  //   - a borderless container holding two or more items (so align
  //     springs on the wrapper have visible space to push the stack).
  // Single-item borderless wrappers stay transparent: (World) prints
  // "World" and ([x]) prints the contents of [x] verbatim.
  const renderBoxRows = (ast, bc, { targetW, targetH } = {}) => {
    const { items = [], align = "center", bordered } = ast;

    const hasPad = bordered || items.length > 1;
    const padBudget = hasPad ? 2 * NATURAL_PAD : 0;
    const borderBudget = bordered ? 2 : 0;

    // When targetW is set we pass maxW down to any row items so a long
    // flow chain wraps at item boundaries to fit the sized frame. The
    // maxW is the inside of the final frame, minus the natural side
    // padding so the wrapped chunks don't touch the border wall.
    const sizedContentW =
      targetW !== undefined
        ? Math.max(1, targetW - borderBudget - (hasPad ? 2 * NATURAL_PAD : 0))
        : undefined;

    const itemRows = items.flatMap((it) =>
      renderItemRows(it, bc, sizedContentW),
    );
    const naturalContentW = itemRows.reduce(
      (max, r) => Math.max(max, displayWidth(r)),
      1,
    );

    const naturalOuterW = naturalContentW + padBudget + borderBudget;
    const naturalOuterH = itemRows.length + borderBudget;

    const outerW = targetW ?? naturalOuterW;
    const outerH = targetH ?? naturalOuterH;

    const contentW = Math.max(0, outerW - borderBudget);
    const contentH = Math.max(0, outerH - borderBudget);

    const clipped = itemRows.map((r) => clipRow(r, contentW));
    const paddedH = vertPad(clipped, contentH, contentW);
    const padded = paddedH.map((r) => padRow(r, contentW, align));

    if (!bordered) return padded;

    const hLine = bc.h.repeat(contentW);
    return [
      bc.tl + hLine + bc.tr,
      ...padded.map((r) => bc.v + r + bc.v),
      bc.bl + hLine + bc.br,
    ];
  };

  // Top-level render entry: honors meta.size on the root node only.
  const renderRows = (ast) => {
    const bc = boxChars(ast);
    const size = ast.meta?.size;
    return renderBoxRows(ast, bc, { targetW: size?.w, targetH: size?.h });
  };

  // Shared helpers for the SVG / PNG backends. Each row becomes one line
  // of paint; when the root has a real vector border we skip the outer
  // glyphs (rows[0] and rows[-1]) and strip the side glyphs so the text
  // is not doubled over the vector rect.
  const maxRowWidth = (rows) =>
    rows.reduce((m, r) => Math.max(m, displayWidth(r)), 0);

  const paintableBody = (ast, row) => (ast.bordered ? row.slice(1, -1) : row);

  const paintableRange = (ast, rows) => ({
    first: ast.bordered ? 1 : 0,
    last: ast.bordered ? rows.length - 1 : rows.length,
  });

  const FONT_STACK_MONO = "ui-monospace, Menlo, Consolas, monospace";
  const FONT_SIZE_PX = CELL_PX_H * 0.7;

  const drawCanvasBorder = (ctx, w, h, color) => {
    ctx.lineWidth = 1.5;
    ctx.strokeStyle = color;
    if (ctx.roundRect) {
      ctx.beginPath();
      ctx.roundRect(0.75, 0.75, w - 1.5, h - 1.5, 3);
      ctx.stroke();
    } else {
      ctx.strokeRect(0.75, 0.75, w - 1.5, h - 1.5);
    }
  };

  // Build a canvas that paints the rows. Shared by the display renderer
  // (wraps the canvas in an <img>) and the PNG exporter (converts it to
  // a Blob).
  const paintCanvas = (ast, rows, color) => {
    const w = maxRowWidth(rows) * CELL_PX_W;
    const h = rows.length * CELL_PX_H;
    const dpr = window.devicePixelRatio || 1;
    const canvas = document.createElement("canvas");
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = w + "px";
    canvas.style.height = h + "px";
    const ctx = canvas.getContext("2d");
    ctx.scale(dpr, dpr);
    if (ast.bordered) drawCanvasBorder(ctx, w, h, color);
    ctx.fillStyle = color;
    ctx.font = `${FONT_SIZE_PX}px ${FONT_STACK_MONO}`;
    ctx.textBaseline = "middle";
    ctx.textAlign = "left";
    const range = paintableRange(ast, rows);
    const x = ast.bordered ? CELL_PX_W : 0;
    for (let i = range.first; i < range.last; i++) {
      ctx.fillText(paintableBody(ast, rows[i]), x, (i + 0.5) * CELL_PX_H);
    }
    return { canvas, w, h };
  };

  // ---- themes -----------------------------------------------------------
  // Known themes and their effect:
  //   simple         default (Unicode box chars, CSS auto-palette)
  //   ascii          ASCII-only box chars (+ - |) for the ASCII backend
  //   dark / light   force the CSS palette regardless of prefers-color-scheme
  // Unknown themes are passed through to the data-theme attribute so
  // adopters can add custom palettes in their own stylesheet.
  const UNICODE_BOX = {
    tl: "┌",
    tr: "┐",
    bl: "└",
    br: "┘",
    h: "─",
    v: "│",
    // Filled triangle arrowheads (U+25C0 / U+25B6). Bold, immediately
    // readable as arrows; render with a small gap before the '│'
    // border of the next node in some fonts, which the project
    // accepts as a style trade-off.
    arrowL: "◀",
    arrowR: "▶",
  };
  const ASCII_BOX = {
    tl: "+",
    tr: "+",
    bl: "+",
    br: "+",
    h: "-",
    v: "|",
    arrowL: "<",
    arrowR: ">",
  };
  const boxChars = (ast) =>
    ast.meta?.theme === "ascii" ? ASCII_BOX : UNICODE_BOX;

  // ---- renderers --------------------------------------------------------
  //
  // Each backend takes the rows from renderRows(ast) and paints them in
  // its medium. The rows already carry any nested box borders drawn with
  // ASCII glyphs, so SVG / PNG render them as a grid of monospace text
  // (plus an outer vector rect when the root is bordered).
  const renderers = {
    ascii(ast) {
      const rows = renderRows(ast);
      const pre = document.createElement("pre");
      pre.className = "pylon-ascii";
      // Per-cell DOM so CJK / emoji stay aligned regardless of font.
      for (let i = 0; i < rows.length; i++) {
        if (i > 0) pre.append(document.createTextNode("\n"));
        for (const ch of rows[i]) {
          const cw = charWidth(ch.codePointAt(0));
          if (cw === 0) {
            pre.append(document.createTextNode(ch));
            continue;
          }
          const cell = document.createElement("span");
          cell.className = "pylon-ascii-cell";
          cell.style.width = cw + "ch";
          cell.textContent = ch;
          pre.append(cell);
        }
      }
      return pre;
    },

    svg(ast) {
      const rows = renderRows(ast);
      const w = maxRowWidth(rows) * CELL_PX_W;
      const h = rows.length * CELL_PX_H;
      const ns = "http://www.w3.org/2000/svg";
      const svg = document.createElementNS(ns, "svg");
      svg.setAttribute("width", w);
      svg.setAttribute("height", h);
      svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
      svg.classList.add("pylon-svg");
      // Root border as a real vector rect; nested inner borders are
      // left as text glyphs in the rows.
      if (ast.bordered) {
        const rect = document.createElementNS(ns, "rect");
        rect.setAttribute("x", 1);
        rect.setAttribute("y", 1);
        rect.setAttribute("width", w - 2);
        rect.setAttribute("height", h - 2);
        rect.setAttribute("rx", 3);
        rect.setAttribute("fill", "none");
        rect.setAttribute("stroke", "currentColor");
        rect.setAttribute("stroke-width", "1");
        svg.append(rect);
      }
      const range = paintableRange(ast, rows);
      const x = ast.bordered ? CELL_PX_W : 0;
      for (let i = range.first; i < range.last; i++) {
        const t = document.createElementNS(ns, "text");
        t.setAttribute("x", x);
        t.setAttribute("y", (i + 0.5) * CELL_PX_H);
        t.setAttribute("text-anchor", "start");
        t.setAttribute("dominant-baseline", "middle");
        t.setAttribute("font-family", "monospace");
        t.setAttribute("font-size", FONT_SIZE_PX);
        // SVG2 deprecates xml:space; modern browsers look at the CSS
        // white-space property to keep runs of spaces from collapsing.
        // Without this, rows that differ in leading-whitespace (e.g.
        // "(a) -> [b]" where row 0 is padding+box and row 1 starts
        // with the label) slide left and the box borders misalign.
        t.setAttribute("style", "white-space: pre");
        t.setAttribute("xml:space", "preserve");
        t.setAttribute("fill", "currentColor");
        t.textContent = paintableBody(ast, rows[i]);
        svg.append(t);
      }
      return svg;
    },

    png(ast, opts = {}) {
      const { canvas, w, h } = paintCanvas(
        ast,
        renderRows(ast),
        opts.color || "#000",
      );
      const img = document.createElement("img");
      img.src = canvas.toDataURL("image/png");
      img.width = w;
      img.height = h;
      img.className = "pylon-png";
      return img;
    },
  };

  // ---- exporters --------------------------------------------------------
  // Produce the raw payload that the copy / download buttons hand to the
  // clipboard or the filesystem. Each returns either {text, mime, ext} or
  // {blob, mime, ext}. PNG is async because canvas.toBlob is callback-based.
  const exporters = {
    ascii(ast) {
      return {
        text: renderRows(ast).join("\n"),
        mime: "text/plain",
        ext: "txt",
      };
    },

    svg(ast) {
      const svg = renderers.svg(ast);
      const text =
        '<?xml version="1.0" encoding="UTF-8"?>\n' +
        new XMLSerializer().serializeToString(svg);
      return { text, mime: "image/svg+xml", ext: "svg" };
    },

    async png(ast, opts = {}) {
      const { canvas } = paintCanvas(
        ast,
        renderRows(ast),
        opts.color || "#000",
      );
      const blob = await new Promise((res) => canvas.toBlob(res, "image/png"));
      return { blob, mime: "image/png", ext: "png" };
    },
  };

  // ---- custom element ---------------------------------------------------
  const FORMATS = ["ascii", "svg", "png"];

  class PylonElement extends HTMLElement {
    static get observedAttributes() {
      return ["format", "wysiwyg", "src"];
    }

    constructor() {
      super();
      this._source = "";
      this._format = null; // dropdown override; null means fall back to attribute
      this._viewHost = null;
      this._editor = null;
      this._formatSelect = null;
    }

    connectedCallback() {
      this._source = this.getAttribute("src") ?? this.textContent ?? "";
      this._mount();
    }

    attributeChangedCallback(name) {
      if (!this.isConnected) return;
      if (name === "src") {
        this._source = this.getAttribute("src") ?? "";
      }
      if (name === "format") {
        this._format = null;
      }
      this._mount();
    }

    _currentFormat() {
      const attr = (this.getAttribute("format") ?? "ascii").toLowerCase();
      const fmt = this._format ?? attr;
      return FORMATS.includes(fmt) ? fmt : "ascii";
    }

    _mount() {
      this.innerHTML = "";
      if (this.hasAttribute("wysiwyg")) {
        this._mountWysiwyg();
      } else {
        this._mountView();
      }
      this._render();
    }

    _mountView() {
      this._viewHost = document.createElement("div");
      this._viewHost.className = "pylon-view";
      this.append(this._viewHost);
      this._editor = null;
      this._formatSelect = null;
    }

    _mountWysiwyg() {
      // Left pane: plaintext editor. Placeholder matches the stub
      // parser's empty-source fallback so the two panes agree at rest.
      const editor = document.createElement("textarea");
      editor.className = "pylon-editor";
      editor.placeholder = DEFAULT_EXAMPLE;
      editor.value = this._source;
      editor.rows = Math.max(6, this._source.split("\n").length + 1);
      editor.addEventListener("input", () => {
        this._source = editor.value;
        this._render();
      });
      this._editor = editor;
      this.append(editor);

      // Right pane: guide + toolbar + rendered output.
      const right = document.createElement("div");
      right.className = "pylon-right";

      // Compact reference so users don't have to leave the pane to
      // recall the frontmatter keys and alignment syntax.
      const guide = document.createElement("div");
      guide.className = "pylon-guide";
      guide.innerHTML =
        '<div><span class="pylon-guide-key">size:</span> <code>WxH</code>' +
        '<span class="pylon-guide-sep">·</span>' +
        '<span class="pylon-guide-key">theme:</span> <code>simple | ascii | dark | light</code></div>' +
        '<div><span class="pylon-guide-key">align:</span> <code>[- x -]</code> center' +
        '<span class="pylon-guide-sep">·</span>' +
        "<code>[- x&nbsp;&nbsp;]</code> right" +
        '<span class="pylon-guide-sep">·</span>' +
        "<code>[&nbsp;&nbsp;x -]</code> left</div>";

      const toolbar = document.createElement("div");
      toolbar.className = "pylon-toolbar";

      const copyBtn = this._makeToolbarButton("Copy", () =>
        this._copy(copyBtn),
      );
      const downloadBtn = this._makeToolbarButton("Download", () =>
        this._download(),
      );
      toolbar.append(copyBtn, downloadBtn);

      const select = document.createElement("select");
      select.className = "pylon-format-select";
      const current = this._currentFormat();
      for (const fmt of FORMATS) {
        const opt = document.createElement("option");
        opt.value = fmt;
        opt.textContent = fmt.toUpperCase();
        if (fmt === current) opt.selected = true;
        select.append(opt);
      }
      select.addEventListener("change", () => {
        this._format = select.value;
        this._render();
      });
      this._formatSelect = select;
      toolbar.append(select);

      this._viewHost = document.createElement("div");
      this._viewHost.className = "pylon-view";

      right.append(guide, toolbar, this._viewHost);
      this.append(right);
    }

    _render() {
      if (!this._viewHost) return;
      const renderer = renderers[this._currentFormat()] ?? renderers.ascii;
      const ast = parse(this._source);
      const theme = ast.meta?.theme;
      if (theme) {
        this.dataset.theme = theme;
      } else {
        delete this.dataset.theme;
      }
      const color = getComputedStyle(this._viewHost).color;
      this._viewHost.innerHTML = "";
      this._viewHost.append(renderer(ast, { color }));
    }

    _makeToolbarButton(label, onClick) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "pylon-btn";
      btn.textContent = label;
      btn.addEventListener("click", onClick);
      return btn;
    }

    async _currentExport() {
      const format = this._currentFormat();
      const ast = parse(this._source);
      const color = this._viewHost
        ? getComputedStyle(this._viewHost).color
        : "#000";
      return await exporters[format](ast, { color });
    }

    async _copy(btn) {
      try {
        const out = await this._currentExport();
        if (out.text) {
          await navigator.clipboard.writeText(out.text);
        } else if (out.blob && window.ClipboardItem) {
          await navigator.clipboard.write([
            new ClipboardItem({ [out.mime]: out.blob }),
          ]);
        } else {
          throw new Error("unsupported");
        }
        if (btn) this._flashButton(btn, "Copied");
      } catch (e) {
        if (btn) this._flashButton(btn, "Failed");
      }
    }

    async _download() {
      const out = await this._currentExport();
      const blob = out.blob ?? new Blob([out.text], { type: out.mime });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `pylon.${out.ext}`;
      a.style.display = "none";
      document.body.append(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    }

    _flashButton(btn, text) {
      const original = btn.textContent;
      btn.textContent = text;
      btn.disabled = true;
      setTimeout(() => {
        btn.textContent = original;
        btn.disabled = false;
      }, 900);
    }
  }

  if (!customElements.get("pylon-chart")) {
    customElements.define("pylon-chart", PylonElement);
  }
})();
