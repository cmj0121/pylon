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

  const FRONTMATTER_RE = /^---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*\r?\n?/;

  // YAML subset scalar parser: numbers (\d+(\.\d+)?), double-quoted
  // strings, and unquoted text (treated as a literal trimmed string).
  // YAML reserved words (true/false/null/~/yes/no) are kept as literal
  // strings -- we stay subset-strict rather than interpret booleans.
  const YAML_NUMBER_RE = /^-?\d+(?:\.\d+)?$/;
  const parseYamlScalar = (raw) => {
    const s = raw.trim();
    if (s === "") return { ok: false };
    if (YAML_NUMBER_RE.test(s)) return { ok: true, value: parseFloat(s) };
    if (s.length >= 2 && s[0] === '"' && s[s.length - 1] === '"') {
      const inner = s.slice(1, -1);
      // Reject anything with control chars or embedded quotes; we do
      // not implement YAML escaping for v0.2.0.
      if (inner.includes('"') || inner.includes("\\")) return { ok: false };
      return { ok: true, value: inner };
    }
    return { ok: true, value: s };
  };

  // Parse the body that follows `data:` into either a flat series
  // (array of {x,y,...} maps) or a map of named series. Returns
  // { ok: true, value } or { ok: false } on unsupported shape.
  //
  // Caller provides `lines`: the raw lines AFTER the `data:` declaration
  // (the header itself is already consumed). Each line is the original
  // text; tabs in any leading indent reject the whole parse.
  //
  // Shape detection:
  //   - No non-blank indented lines  -> null (empty data).
  //   - First non-blank line is `  - ...` -> flat series.
  //   - First non-blank line is `  name:` -> map of named series.
  const parseDataSection = (lines) => {
    // Find the first non-blank line and its indent.
    let first = -1;
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].trim() !== "") {
        first = i;
        break;
      }
    }
    if (first === -1) return { ok: true, value: null };

    // Reject tabs in any leading whitespace. Must run before the
    // baseIndent==0 early-return so a tab-led line (whose leading-space
    // count is zero) still rejects the whole frontmatter.
    for (const line of lines) {
      const lead = line.match(/^(\s*)/)[1];
      if (lead.includes("\t")) return { ok: false };
    }

    const baseIndent = lines[first].match(/^( *)/)[1].length;
    if (baseIndent === 0) return { ok: true, value: null };

    const firstContent = lines[first].slice(baseIndent);
    if (firstContent.startsWith("- ") || firstContent === "-") {
      return parseSeriesList(lines, baseIndent);
    }
    if (/^[A-Za-z_][A-Za-z0-9_]*\s*:/.test(firstContent)) {
      return parseSeriesMap(lines, baseIndent);
    }
    return { ok: false };
  };

  // Parse a block-style list of maps at the given indent:
  //   - x: 1
  //     y: 10
  //   - x: 2
  //     y: 20
  // Every list item is a `- ` line optionally followed by further
  // indented key:value lines that belong to the same map.
  const parseSeriesList = (lines, baseIndent) => {
    const items = [];
    let current = null;
    let childIndent = -1;
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line.trim() === "") continue;
      const lead = line.match(/^( *)/)[1].length;
      if (lead < baseIndent) return { ok: false };
      const content = line.slice(baseIndent);
      if (lead === baseIndent) {
        if (!content.startsWith("- ") && content !== "-") return { ok: false };
        current = {};
        items.push(current);
        const rest = content === "-" ? "" : content.slice(2);
        if (rest !== "") {
          const m = rest.match(/^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$/);
          if (!m) return { ok: false };
          const val = parseYamlScalar(m[2]);
          if (!val.ok) return { ok: false };
          current[m[1]] = val.value;
          // Child indent is baseIndent + 2 (the '- ' width).
          if (childIndent === -1) childIndent = baseIndent + 2;
        }
      } else {
        if (!current) return { ok: false };
        if (childIndent === -1) childIndent = lead;
        if (lead !== childIndent) return { ok: false };
        const kv = content
          .slice(lead - baseIndent)
          .match(/^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$/);
        if (!kv) return { ok: false };
        const val = parseYamlScalar(kv[2]);
        if (!val.ok) return { ok: false };
        current[kv[1]] = val.value;
      }
    }
    return { ok: true, value: items };
  };

  // Parse a map of named series at the given indent:
  //   counter:
  //     - x: 1
  //       y: 10
  //   sales:
  //     - x: 1
  //       y: 100
  // Each top-level key names a series; its value MUST be a list of maps
  // at a deeper indent.
  const parseSeriesMap = (lines, baseIndent) => {
    const out = {};
    let currentKey = null;
    let childLines = null;
    let childIndent = -1;
    const flushChild = () => {
      if (currentKey === null) return true;
      if (!childLines || childLines.length === 0) return false;
      const sub = parseSeriesList(childLines, childIndent);
      if (!sub.ok) return false;
      out[currentKey] = sub.value;
      currentKey = null;
      childLines = null;
      childIndent = -1;
      return true;
    };
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line.trim() === "") {
        if (childLines) childLines.push(line);
        continue;
      }
      const lead = line.match(/^( *)/)[1].length;
      if (lead < baseIndent) return { ok: false };
      if (lead === baseIndent) {
        if (!flushChild()) return { ok: false };
        const content = line.slice(baseIndent);
        const m = content.match(/^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*$/);
        if (!m) return { ok: false };
        currentKey = m[1];
        childLines = [];
      } else {
        if (currentKey === null) return { ok: false };
        if (childIndent === -1) childIndent = lead;
        if (lead < childIndent) return { ok: false };
        childLines.push(line);
      }
    }
    if (!flushChild()) return { ok: false };
    return { ok: true, value: out };
  };

  const parseFrontmatter = (text) => {
    const meta = {};
    const errors = [];
    const lines = text.split(/\r?\n/);
    let i = 0;
    while (i < lines.length) {
      const line = lines[i];
      // Skip blank lines quickly.
      if (line.trim() === "") {
        i++;
        continue;
      }
      // `data:` triggers the indent-aware sub-parser. We don't try to
      // parse data: inline -- even `data: []` is out of scope (flow
      // style is excluded from the subset).
      if (/^\s*data\s*:\s*$/.test(line)) {
        // Gather all following lines until we hit a top-level key or
        // EOF. A "top-level key" is a key: line with zero leading
        // whitespace (same indent level as `data:` itself). We tolerate
        // indented continuation lines between, including blanks.
        const sectionLines = [];
        let j = i + 1;
        while (j < lines.length) {
          const l = lines[j];
          if (l.trim() === "") {
            sectionLines.push(l);
            j++;
            continue;
          }
          // Top-level key means zero leading whitespace plus `ident:`.
          if (/^[A-Za-z_][A-Za-z0-9_]*\s*:/.test(l)) break;
          sectionLines.push(l);
          j++;
        }
        const parsed = parseDataSection(sectionLines);
        if (parsed.ok) {
          meta.data = parsed.value;
        } else {
          errors.push("Unsupported data: frontmatter shape");
        }
        i = j;
        continue;
      }
      const m = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*?)\s*$/);
      if (!m) {
        i++;
        continue;
      }
      const [, key, raw] = m;
      if (key === "size") {
        const s = raw.match(/^(\d+)\s*[xX]\s*(\d+)$/);
        if (s) meta.size = { w: parseInt(s[1], 10), h: parseInt(s[2], 10) };
      } else if (key === "theme") {
        meta.theme = raw;
      }
      i++;
    }
    if (errors.length) meta._errors = errors;
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

  // Named node and reference tokens:
  //
  //   [ value :: name ] -- declare a node named `name`; the trailing
  //                        '::' + identifier is stripped from the label.
  //                        Using '::' (not a single colon) avoids
  //                        collision with natural labels like
  //                        '[Status: ok]' or '[- 12:00 -]'.
  //   &name             -- reference a previously declared node. A ref
  //                        renders as the name itself (inline text) so
  //                        the declaration remains the only place the
  //                        full content appears. References are *not*
  //                        clones -- they are pointers back to the
  //                        original binding.
  //
  // Duplicate declarations and unresolved references are pushed onto
  // `root._errors` so the host element can surface them (as a toast,
  // not a native alert).
  const NAME_RE = /::\s*([A-Za-z_]\w*)\s*$/;
  const REF_RE = /^&([A-Za-z_]\w*)/;

  // Data references `@ident` inside a box body. The sigil is greedy
  // about letters but conservative about boundaries -- `@` is only a
  // sigil when preceded by `[`, `(`, whitespace, or start-of-string
  // AND followed by an ident then whitespace / `|` / `]` / `)` /
  // end-of-string. This keeps `user@example.com` literal.
  const DATAREF_IDENT_RE = /^([A-Za-z_][A-Za-z0-9_]*)/;
  const isDataRefBoundary = (prevCh, nextCh) => {
    if (prevCh !== "" && !/[\s[(]/.test(prevCh)) return false;
    if (nextCh === "" || nextCh === undefined) return true;
    return /[\s|\])]/.test(nextCh);
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
      } else if (c === "&") {
        const m = s.slice(i).match(REF_RE);
        if (m) {
          flushText();
          lineItems.push({ type: "ref", name: m[1] });
          i += m[0].length;
          continue;
        }
        textBuf += c;
        i++;
      } else if (c === "@") {
        const prevCh = i === 0 ? "" : s[i - 1];
        const identMatch = s.slice(i + 1).match(DATAREF_IDENT_RE);
        if (identMatch) {
          const nextCh = s[i + 1 + identMatch[0].length] ?? "";
          if (isDataRefBoundary(prevCh, nextCh)) {
            flushText();
            lineItems.push({ type: "dataRef", name: identMatch[1] });
            i += 1 + identMatch[0].length;
            continue;
          }
        }
        textBuf += c;
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

  // Trailing `| <renderer>` marker on a box body. Whitespace on both
  // sides of the pipe is required so it doesn't collide with labels
  // that naturally contain '|' (e.g. "foo | bar" meaning "or"). The
  // FIRST pipe wins -- a body like `@data | bar | text` parses as
  // renderer `bar` (extras like `| text` are treated as literal text
  // stripped along with the first marker). We match the leftmost
  // ` | <ident>` in a chain that extends to end-of-string.
  const RENDERER_RE =
    /\s+\|\s+([A-Za-z_][A-Za-z0-9_]*)(?:\s+\|\s+[A-Za-z_][A-Za-z0-9_]*)*\s*$/;

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
    let body = clean;
    let name;
    const m = clean.match(NAME_RE);
    if (m) {
      name = m[1];
      body = clean.slice(0, m.index).trimEnd();
    }
    let renderer;
    const r = body.match(RENDERER_RE);
    if (r) {
      renderer = r[1];
      body = body.slice(0, r.index).trimEnd();
    }
    const box = {
      type: "box",
      bordered,
      align,
      items: parseItems(body),
    };
    if (name) box.name = name;
    if (renderer) box.renderer = renderer;
    return box;
  };

  // Walk the AST and collect every `box.name` into `map`. A repeated
  // name is recorded in `errors` -- the first occurrence wins so that
  // `&name` lookups remain deterministic.
  const collectNames = (item, map, errors) => {
    if (!item || typeof item !== "object") return;
    if (item.type === "box" && item.name) {
      if (map.has(item.name)) {
        errors.push(`Duplicate node name: ${item.name}`);
      } else {
        map.set(item.name, item);
      }
    }
    if (Array.isArray(item.items)) {
      for (const child of item.items) collectNames(child, map, errors);
    }
    if (item.type === "edge" && item.label) {
      collectNames(item.label, map, errors);
    }
  };

  // Resolve `ref` nodes. Three paths:
  //
  //   1. Same-row arc: when the declaration lives in the *same* row
  //      as the ref, the ref is tagged as `sameRowArc` and drawn as
  //      a U-shaped arc beneath the row. The arms anchor on the
  //      target box (arrow end) and the source box (the box that
  //      precedes the ref's edge). When target === source the arc
  //      collapses to the existing self-loop shape. The edge that
  //      led into the ref is consumed.
  //
  //   2. Cross-row: the declaration is in a different top-level row.
  //      Routed through the right-side gutter by overlayCrossRowGutter.
  //
  //   3. Inline text: no matching declaration, or another ref has
  //      already claimed the target's gutter lane. Renders as the
  //      name as plain text (or the literal `&name` on unresolved
  //      with an error).
  const tagSameRowArcs = (row) => {
    const items = row.items;
    const boxByName = new Map();
    for (const c of items) {
      if (c?.type === "box" && c.name) boxByName.set(c.name, c);
    }
    // What the current ref's arrow flows *out of*. Typically the box
    // two slots back (`[src] <edge> &ref`), but in a chain like
    // `&x -> &y` the previous ref's target stands in for `[src]`.
    const boxLike = (it) => {
      if (it?.type === "box") return it;
      if (it?.type === "ref" && it.target) return it.target;
      return null;
    };
    for (let i = 0; i < items.length; i++) {
      const ref = items[i];
      if (!(ref?.type === "ref")) continue;
      const target = boxByName.get(ref.name);
      if (!target) continue;
      let sourceBox = null;
      if (i >= 2 && items[i - 1]?.type === "edge") {
        sourceBox = boxLike(items[i - 2]);
      }
      if (!sourceBox && i >= 1) {
        sourceBox = boxLike(items[i - 1]);
      }
      ref.sameRowArc = true;
      ref.target = target;
      ref.sourceBox = sourceBox || target;
      if (i > 0 && items[i - 1]?.type === "edge") {
        items[i - 1]._consumedBySameRowArc = true;
      }
    }
  };

  const resolveRefText = (ref, map, errors) => {
    if (!map.has(ref.name)) {
      errors.push(`Undefined ref: &${ref.name}`);
      return { type: "text", content: "&" + ref.name };
    }
    return { type: "text", content: ref.name };
  };

  // Cross-row tagging: a ref whose declaration lives in a *different*
  // row is rendered as a right-side gutter arrow instead of inline
  // text. Each target gets at most one gutter arrow -- later refs to
  // the same target fall back to inline name text, because multiple
  // arrows converging on the same column don't stack cleanly in an
  // ASCII grid. The edge immediately preceding the claimed ref is
  // consumed (it becomes the dash line into the gutter).
  const markCrossRowRefs = (row, map, claimed) => {
    for (let i = 0; i < row.items.length; i++) {
      const ref = row.items[i];
      if (!(ref && ref.type === "ref" && !ref.sameRowArc)) continue;
      if (!map.has(ref.name)) continue;
      const target = map.get(ref.name);
      if (claimed.has(target)) continue;
      claimed.add(target);
      ref.crossRow = true;
      ref.target = target;
      if (i > 0 && row.items[i - 1]?.type === "edge") {
        row.items[i - 1]._consumedByCrossRow = true;
      }
    }
  };

  const resolveRefs = (item, map, errors, claimed) => {
    if (!item || typeof item !== "object") return;
    if (!claimed) claimed = new Set();
    if (item.type === "row") {
      tagSameRowArcs(item);
      markCrossRowRefs(item, map, claimed);
    }
    if (Array.isArray(item.items)) {
      item.items = item.items.map((c) => {
        if (c && c.type === "ref") {
          if (c.sameRowArc || c.crossRow) return c;
          return resolveRefText(c, map, errors);
        }
        resolveRefs(c, map, errors, claimed);
        return c;
      });
    }
    if (item.type === "edge" && item.label) {
      if (item.label.type === "ref") {
        item.label = resolveRefText(item.label, map, errors);
      } else {
        resolveRefs(item.label, map, errors, claimed);
      }
    }
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
    root.meta = meta;

    const nameMap = new Map();
    const errors = [];
    if (Array.isArray(meta._errors)) errors.push(...meta._errors);
    collectNames(root, nameMap, errors);
    resolveRefs(root, nameMap, errors);
    if (errors.length) {
      root._errors = [...new Set(errors)];
    }
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
    // Same-row and cross-row refs (and the edges that led into them)
    // render as arcs -- beneath the row for same-row arcs, in the
    // right-side gutter for cross-row refs -- not inline.
    const linearItems = row.items.filter(
      (it) =>
        !(it && it.type === "ref" && (it.sameRowArc || it.crossRow)) &&
        !it._consumedBySameRowArc &&
        !it._consumedByCrossRow,
    );
    const parts = linearItems.map((it) => {
      if (it.type === "edge") {
        const text = edgeString(it, bc);
        return {
          kind: "edge",
          edge: it,
          text,
          width: displayWidth(text),
          item: it,
        };
      }
      const rows = renderItemRows(it, bc);
      const width = rows.reduce((m, r) => Math.max(m, displayWidth(r)), 0);
      return { kind: "block", rows, width, item: it };
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

    // Same-row refs render as U-arcs. Self-loops stay below when
    // they're alone, and hop above when mixed with a cross-box arc --
    // otherwise their source arm threads through the cross-box rows as
    // a long dangling line. Two cross-box arcs between the same pair
    // fold into a shared arrow row with nested U's (inner on near cols,
    // outer on far cols); everything else takes one arrow + one corner
    // row per arc. A post-pass extends below-arc arms upward into
    // blank cells so they trail back to the declaration; the arrow
    // head rides up with the arm when extended.
    const arcs = row.items.filter(
      (it) => it && it.type === "ref" && it.sameRowArc,
    );
    if (arcs.length > 0 && totalWidth > 0) {
      const partStart = new Map();
      {
        let c = 0;
        for (const p of parts) {
          partStart.set(p, c);
          c += p.width;
        }
      }
      const blockPart = (item) =>
        parts.find((p) => p.kind === "block" && p.item === item);

      const belowResolved = [];
      const selfResolved = [];
      let hasCrossBox = false;
      for (const ref of arcs) {
        const tgtPart = blockPart(ref.target);
        if (!tgtPart) continue;
        const tgtStart = partStart.get(tgtPart);
        const tgtW = tgtPart.width;
        if (ref.sourceBox === ref.target) {
          selfResolved.push({ tgtStart, tgtW });
          continue;
        }
        const srcPart = blockPart(ref.sourceBox);
        if (!srcPart) continue;
        const srcStart = partStart.get(srcPart);
        const srcW = srcPart.width;
        hasCrossBox = true;
        belowResolved.push({
          tgtStart,
          tgtW,
          srcStart,
          srcW,
          lo: Math.min(tgtStart, srcStart),
          hi: Math.max(tgtStart, srcStart),
        });
      }
      const aboveResolved = hasCrossBox ? selfResolved : [];
      if (!hasCrossBox) belowResolved.push(...selfResolved);

      const compact =
        belowResolved.length === 2 &&
        belowResolved.every((r) => r.srcStart !== undefined) &&
        belowResolved[0].lo === belowResolved[1].lo &&
        belowResolved[0].hi === belowResolved[1].hi;

      const belowSpecs = [];
      for (let i = 0; i < belowResolved.length; i++) {
        const r = belowResolved[i];
        let armL, armR, tgtArm;
        if (r.srcStart === undefined) {
          armL = r.tgtStart + 2;
          armR = r.tgtStart + r.tgtW - 3;
          tgtArm = armR;
        } else if (compact) {
          const isTgtLeft = r.tgtStart < r.srcStart;
          const leftStart = isTgtLeft ? r.tgtStart : r.srcStart;
          const leftW = isTgtLeft ? r.tgtW : r.srcW;
          const rightStart = isTgtLeft ? r.srcStart : r.tgtStart;
          const rightW = isTgtLeft ? r.srcW : r.tgtW;
          if (i === 0) {
            armL = leftStart + leftW - 3;
            armR = rightStart + 2;
          } else {
            armL = leftStart + 2;
            armR = rightStart + rightW - 3;
          }
          tgtArm = isTgtLeft ? armL : armR;
        } else {
          const tgtCol = r.tgtStart + r.tgtW - 3;
          const srcCol = r.srcStart + r.srcW - 3;
          armL = Math.min(tgtCol, srcCol);
          armR = Math.max(tgtCol, srcCol);
          tgtArm = tgtCol;
        }
        if (armR > armL) belowSpecs.push({ armL, armR, tgtArm });
      }

      const blankCells = () => new Array(totalWidth).fill(" ");
      const paintArrow = (cells, spec, head) => {
        cells[spec.armL] = spec.tgtArm === spec.armL ? head : bc.v;
        cells[spec.armR] = spec.tgtArm === spec.armR ? head : bc.v;
      };
      const paintCorner = (cells, spec, left, right) => {
        cells[spec.armL] = left;
        for (let c = spec.armL + 1; c < spec.armR; c++) cells[c] = bc.h;
        cells[spec.armR] = right;
      };
      const extendArmsUp = (grid, specs) => {
        for (const spec of specs) {
          for (const c of [spec.armL, spec.armR]) {
            const isArrow = c === spec.tgtArm;
            let top = spec.arrowRow;
            for (let r = spec.arrowRow - 1; r >= 0; r--) {
              if ((grid[r][c] ?? " ") !== " ") break;
              top = r;
            }
            for (let r = top; r < spec.arrowRow; r++) grid[r][c] = bc.v;
            if (isArrow && top < spec.arrowRow) {
              grid[spec.arrowRow][c] = bc.v;
              grid[top][c] = "▲";
            }
          }
        }
      };

      let bodyOut = out;

      if (compact && belowSpecs.length === 2) {
        belowSpecs.sort((a, b) => a.armR - a.armL - (b.armR - b.armL));
        const arrowRowIdx = bodyOut.length;
        const arrow = blankCells();
        for (const spec of belowSpecs) {
          paintArrow(arrow, spec, "▲");
          spec.arrowRow = arrowRowIdx;
        }
        bodyOut.push(arrow.join(""));
        for (const spec of belowSpecs) {
          const corner = blankCells();
          paintCorner(corner, spec, bc.bl, bc.br);
          spec.cornerRow = bodyOut.length;
          bodyOut.push(corner.join(""));
        }
        const grid = bodyOut.map((r) => Array.from(r));
        // Outer arms drop through inner corner rows; staggered cols
        // guarantee they land on blanks.
        for (const spec of belowSpecs) {
          for (const c of [spec.armL, spec.armR]) {
            for (let r = arrowRowIdx + 1; r < spec.cornerRow; r++) {
              if ((grid[r][c] ?? " ") === " ") grid[r][c] = bc.v;
            }
          }
        }
        extendArmsUp(grid, belowSpecs);
        bodyOut = grid.map((arr) => arr.join(""));
      } else if (belowSpecs.length > 0) {
        for (const spec of belowSpecs) {
          const arrow = blankCells();
          paintArrow(arrow, spec, "▲");
          spec.arrowRow = bodyOut.length;
          bodyOut.push(arrow.join(""));
          const corner = blankCells();
          paintCorner(corner, spec, bc.bl, bc.br);
          bodyOut.push(corner.join(""));
        }
        const grid = bodyOut.map((r) => Array.from(r));
        extendArmsUp(grid, belowSpecs);
        bodyOut = grid.map((arr) => arr.join(""));
      }

      // Above-arcs sit flush against the box top (arrow becomes `▼`),
      // so no extend-up is needed.
      const aboveRows = [];
      for (const r of aboveResolved) {
        const armL = r.tgtStart + 2;
        const armR = r.tgtStart + r.tgtW - 3;
        if (armR <= armL) continue;
        const spec = { armL, armR, tgtArm: armR };
        const corner = blankCells();
        paintCorner(corner, spec, bc.tl, bc.tr);
        const arm = blankCells();
        paintArrow(arm, spec, "▼");
        aboveRows.push(corner.join(""), arm.join(""));
      }
      return aboveRows.concat(bodyOut);
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

  // ---- chart renderers -------------------------------------------------
  //
  // A `| renderer` marker on a box body routes rendering through this
  // table (distinct from the top-level `renderers` map, which is the
  // output backend -- ASCII / SVG / PNG). Each chart renderer takes
  // the resolved data ref value and the theme's box-drawing chars,
  // and returns an array of text strings that REPLACE the box's items
  // (one text item per row). Errors short-circuit to a single inline
  // `⚠ ...` row.
  const BAR_WIDTH_DEFAULT = 10;
  const BAR_HEIGHT_DEFAULT = 10;
  const BAR_GLYPH = "\u2588";

  // Progress primitive. Fixed 20-cell budget; glyphs swap between
  // Unicode (full block + light shade) and ASCII (`#` + `.`) based
  // on reference equality with ASCII_BOX. Mirrors Go's
  // render_progress.go constants byte-for-byte so the two sides
  // cannot drift on layout.
  const PROGRESS_BUDGET = 20;
  const PROGRESS_FILLED_DEFAULT = "\u2588"; // FULL BLOCK
  const PROGRESS_EMPTY_DEFAULT = "\u2591"; // LIGHT SHADE
  const PROGRESS_FILLED_ASCII = "#";
  const PROGRESS_EMPTY_ASCII = ".";

  // clampPercent silently clamps into [0, 100]; NaN collapses to 0
  // so a malformed scalar renders deterministic output instead of
  // leaking "NaN%" into the grid. The validator still surfaces the
  // shape error upstream.
  const clampPercent = (y) => {
    if (typeof y !== "number" || Number.isNaN(y)) return 0;
    if (y < 0) return 0;
    if (y > 100) return 100;
    return y;
  };

  // progressGlyphs picks (filled, empty) glyphs for the active theme.
  // Reference-equality mirrors banner's `bc === ASCII_BOX` check.
  const progressGlyphs = (bc) =>
    bc === ASCII_BOX
      ? [PROGRESS_FILLED_ASCII, PROGRESS_EMPTY_ASCII]
      : [PROGRESS_FILLED_DEFAULT, PROGRESS_EMPTY_DEFAULT];

  // renderProgressRow formats "{bar-20} {pct-4}" -- the tail shared
  // by both scalar and series forms. padStart(3) produces "100",
  // " 75", "  0"; appended "%" produces the 4-cell column.
  const renderProgressRow = (y, filled, empty) => {
    const pct = clampPercent(y);
    let cells = Math.round((pct / 100) * PROGRESS_BUDGET);
    if (cells < 0) cells = 0;
    if (cells > PROGRESS_BUDGET) cells = PROGRESS_BUDGET;
    const bar = filled.repeat(cells) + empty.repeat(PROGRESS_BUDGET - cells);
    const pctStr = String(Math.round(pct)).padStart(3, " ") + "%";
    return `${bar} ${pctStr}`;
  };

  // renderProgressSeries lays out N rows: "{label-right-pad} {bar} {pct}".
  // labelW is the widest x column so every row lines up.
  const renderProgressSeries = (series, filled, empty) => {
    let labelW = 0;
    const labels = series.map((e) => {
      const s = String(e.x);
      if (s.length > labelW) labelW = s.length;
      return s;
    });
    return series.map((e, i) => {
      const label = labels[i].padStart(labelW, " ");
      return `${label} ${renderProgressRow(e.y, filled, empty)}`;
    });
  };

  // parseProgressScalar reads the concatenated text of a renderer
  // body as a finite number. Returns null when the body is empty,
  // all-whitespace, or carries any non-numeric character -- the
  // validator uses the null case to surface the
  // `\u26a0 progress: expected number` diagnostic.
  const parseProgressScalar = (text) => {
    if (typeof text !== "string") return null;
    const trimmed = text.trim();
    if (trimmed === "") return null;
    const n = Number(trimmed);
    if (!Number.isFinite(n)) return null;
    return n;
  };

  // Shared validation for bar-family renderers (hbar, vbar). Returns an
  // inline-error row array on failure or `null` on success. `name` is
  // the invoked renderer name so the error text carries it.
  const validateBarSeries = (refValue, name) => {
    if (!Array.isArray(refValue)) {
      return [`\u26a0 ${name}: expected [{x,y}]`];
    }
    if (refValue.length === 0) return [`\u26a0 ${name}: empty series`];
    const seenX = new Set();
    for (const entry of refValue) {
      if (!entry || typeof entry !== "object") {
        return [`\u26a0 ${name}: expected [{x,y}]`];
      }
      if (!("x" in entry) || !("y" in entry)) {
        return [`\u26a0 ${name}: expected [{x,y}]`];
      }
      if (typeof entry.y !== "number" || Number.isNaN(entry.y)) {
        return [`\u26a0 ${name}: expected [{x,y}]`];
      }
      if (entry.y < 0) return [`\u26a0 ${name}: negative y`];
      const xKey = String(entry.x);
      if (seenX.has(xKey)) {
        return [`\u26a0 ${name}: duplicate x "${xKey}"`];
      }
      seenX.add(xKey);
    }
    return null;
  };

  // Single pass over the series: collect label / value strings and
  // track max label width, max value width, and max y. Callers use
  // this instead of five separate reduce passes.
  const prepareBarSeries = (refValue) => {
    const labels = [];
    const values = [];
    let labelW = 0;
    let valueW = 0;
    let maxY = 0;
    for (const e of refValue) {
      const lab = String(e.x);
      const val = `(${e.y})`;
      labels.push(lab);
      values.push(val);
      if (lab.length > labelW) labelW = lab.length;
      if (val.length > valueW) valueW = val.length;
      if (e.y > maxY) maxY = e.y;
    }
    return { labels, values, labelW, valueW, maxY };
  };

  // Horizontal-bar body shared by the `hbar` renderer and its `bar`
  // alias. `name` is threaded through so validation errors carry the
  // renderer name the user actually invoked.
  const renderHBar = (refValue, bc, budgetW, name) => {
    const err = validateBarSeries(refValue, name);
    if (err) return err;
    const { labels, values, labelW, valueW, maxY } = prepareBarSeries(refValue);
    const budget = Math.max(1, budgetW ?? BAR_WIDTH_DEFAULT);
    const barCells = (y) => {
      if (maxY === 0) return 0;
      return Math.round((y / maxY) * budget);
    };
    return refValue.map((entry, i) => {
      const label = labels[i].padStart(labelW, " ");
      const bars = BAR_GLYPH.repeat(barCells(entry.y));
      const valStr = values[i].padStart(valueW, " ");
      const body = (
        bars + " ".repeat(Math.max(1, budget - bars.length + 1))
      ).slice(0, budget + 1);
      return `${label} ${bc.v} ${body}${valStr} ${bc.v}`;
    });
  };

  // Banner glyph tables. Both maps MUST carry the same key set; the
  // renderer falls back to '?' on miss so an asymmetric key would
  // silently diverge from the Go side. bannerFontDefault mirrors
  // pyfiglet's ansi_shadow (plus hand-crafted space/apostrophe/double-
  // quote); bannerFontASCII is the `#`+space fallback for terminals
  // that can't render box-drawing glyphs. Ported byte-for-byte from
  // src/go/pkg/pylon/render_banner_fonts.go.
  const bannerFontDefault = {
    A: [" █████╗ ", "██╔══██╗", "███████║", "██╔══██║", "██║  ██║", "╚═╝  ╚═╝"],
    B: ["██████╗ ", "██╔══██╗", "██████╔╝", "██╔══██╗", "██████╔╝", "╚═════╝ "],
    C: [" ██████╗", "██╔════╝", "██║     ", "██║     ", "╚██████╗", " ╚═════╝"],
    D: ["██████╗ ", "██╔══██╗", "██║  ██║", "██║  ██║", "██████╔╝", "╚═════╝ "],
    E: ["███████╗", "██╔════╝", "█████╗  ", "██╔══╝  ", "███████╗", "╚══════╝"],
    F: ["███████╗", "██╔════╝", "█████╗  ", "██╔══╝  ", "██║     ", "╚═╝     "],
    G: [
      " ██████╗ ",
      "██╔════╝ ",
      "██║  ███╗",
      "██║   ██║",
      "╚██████╔╝",
      " ╚═════╝ ",
    ],
    H: ["██╗  ██╗", "██║  ██║", "███████║", "██╔══██║", "██║  ██║", "╚═╝  ╚═╝"],
    I: ["██╗", "██║", "██║", "██║", "██║", "╚═╝"],
    J: ["     ██╗", "     ██║", "     ██║", "██   ██║", "╚█████╔╝", " ╚════╝ "],
    K: ["██╗  ██╗", "██║ ██╔╝", "█████╔╝ ", "██╔═██╗ ", "██║  ██╗", "╚═╝  ╚═╝"],
    L: ["██╗     ", "██║     ", "██║     ", "██║     ", "███████╗", "╚══════╝"],
    M: [
      "███╗   ███╗",
      "████╗ ████║",
      "██╔████╔██║",
      "██║╚██╔╝██║",
      "██║ ╚═╝ ██║",
      "╚═╝     ╚═╝",
    ],
    N: [
      "███╗   ██╗",
      "████╗  ██║",
      "██╔██╗ ██║",
      "██║╚██╗██║",
      "██║ ╚████║",
      "╚═╝  ╚═══╝",
    ],
    O: [
      " ██████╗ ",
      "██╔═══██╗",
      "██║   ██║",
      "██║   ██║",
      "╚██████╔╝",
      " ╚═════╝ ",
    ],
    P: ["██████╗ ", "██╔══██╗", "██████╔╝", "██╔═══╝ ", "██║     ", "╚═╝     "],
    Q: [
      " ██████╗ ",
      "██╔═══██╗",
      "██║   ██║",
      "██║▄▄ ██║",
      "╚██████╔╝",
      " ╚══▀▀═╝ ",
    ],
    R: ["██████╗ ", "██╔══██╗", "██████╔╝", "██╔══██╗", "██║  ██║", "╚═╝  ╚═╝"],
    S: ["███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"],
    T: [
      "████████╗",
      "╚══██╔══╝",
      "   ██║   ",
      "   ██║   ",
      "   ██║   ",
      "   ╚═╝   ",
    ],
    U: [
      "██╗   ██╗",
      "██║   ██║",
      "██║   ██║",
      "██║   ██║",
      "╚██████╔╝",
      " ╚═════╝ ",
    ],
    V: [
      "██╗   ██╗",
      "██║   ██║",
      "██║   ██║",
      "╚██╗ ██╔╝",
      " ╚████╔╝ ",
      "  ╚═══╝  ",
    ],
    W: [
      "██╗    ██╗",
      "██║    ██║",
      "██║ █╗ ██║",
      "██║███╗██║",
      "╚███╔███╔╝",
      " ╚══╝╚══╝ ",
    ],
    X: ["██╗  ██╗", "╚██╗██╔╝", " ╚███╔╝ ", " ██╔██╗ ", "██╔╝ ██╗", "╚═╝  ╚═╝"],
    Y: [
      "██╗   ██╗",
      "╚██╗ ██╔╝",
      " ╚████╔╝ ",
      "  ╚██╔╝  ",
      "   ██║   ",
      "   ╚═╝   ",
    ],
    Z: ["███████╗", "╚══███╔╝", "  ███╔╝ ", " ███╔╝  ", "███████╗", "╚══════╝"],
    0: [
      " ██████╗ ",
      "██╔═████╗",
      "██║██╔██║",
      "████╔╝██║",
      "╚██████╔╝",
      " ╚═════╝ ",
    ],
    1: [" ██╗", "███║", "╚██║", " ██║", " ██║", " ╚═╝"],
    2: ["██████╗ ", "╚════██╗", " █████╔╝", "██╔═══╝ ", "███████╗", "╚══════╝"],
    3: ["██████╗ ", "╚════██╗", " █████╔╝", " ╚═══██╗", "██████╔╝", "╚═════╝ "],
    4: ["██╗  ██╗", "██║  ██║", "███████║", "╚════██║", "     ██║", "     ╚═╝"],
    5: ["███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"],
    6: [
      " ██████╗ ",
      "██╔════╝ ",
      "███████╗ ",
      "██╔═══██╗",
      "╚██████╔╝",
      " ╚═════╝ ",
    ],
    7: ["███████╗", "╚════██║", "    ██╔╝", "   ██╔╝ ", "   ██║  ", "   ╚═╝  "],
    8: [" █████╗ ", "██╔══██╗", "╚█████╔╝", "██╔══██╗", "╚█████╔╝", " ╚════╝ "],
    9: [" █████╗ ", "██╔══██╗", "╚██████║", " ╚═══██║", " █████╔╝", " ╚════╝ "],
    " ": ["    ", "    ", "    ", "    ", "    ", "    "],
    ".": ["   ", "   ", "   ", "   ", "██╗", "╚═╝"],
    ",": ["   ", "   ", "   ", "   ", "▄█╗", "╚═╝"],
    "!": ["██╗", "██║", "██║", "╚═╝", "██╗", "╚═╝"],
    "?": [
      "██████╗ ",
      "╚════██╗",
      "  ▄███╔╝",
      "  ▀▀══╝ ",
      "  ██╗   ",
      "  ╚═╝   ",
    ],
    "-": ["      ", "      ", "█████╗", "╚════╝", "      ", "      "],
    "'": ["██╗", "██║", "╚═╝", "   ", "   ", "   "],
    '"': ["██╗ ██╗", "██║ ██║", "╚═╝ ╚═╝", "       ", "       ", "       "],
  };

  const bannerFontASCII = {
    A: [" ### ", "#   #", "#   #", "#####", "#   #", "#   #"],
    B: ["#### ", "#   #", "#### ", "#   #", "#   #", "#### "],
    C: [" ####", "#    ", "#    ", "#    ", "#    ", " ####"],
    D: ["#### ", "#   #", "#   #", "#   #", "#   #", "#### "],
    E: ["#####", "#    ", "#### ", "#    ", "#    ", "#####"],
    F: ["#####", "#    ", "#### ", "#    ", "#    ", "#    "],
    G: [" ####", "#    ", "#    ", "#  ##", "#   #", " ### "],
    H: ["#   #", "#   #", "#####", "#   #", "#   #", "#   #"],
    I: ["###", " # ", " # ", " # ", " # ", "###"],
    J: ["  ###", "   # ", "   # ", "   # ", "#  # ", " ##  "],
    K: ["#   #", "#  # ", "###  ", "#  # ", "#   #", "#   #"],
    L: ["#    ", "#    ", "#    ", "#    ", "#    ", "#####"],
    M: ["#   #", "## ##", "# # #", "#   #", "#   #", "#   #"],
    N: ["#   #", "##  #", "# # #", "#  ##", "#   #", "#   #"],
    O: [" ### ", "#   #", "#   #", "#   #", "#   #", " ### "],
    P: ["#### ", "#   #", "#### ", "#    ", "#    ", "#    "],
    Q: [" ### ", "#   #", "#   #", "# # #", "#  # ", " ## #"],
    R: ["#### ", "#   #", "#### ", "# #  ", "#  # ", "#   #"],
    S: [" ####", "#    ", " ### ", "    #", "    #", "#### "],
    T: ["#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "],
    U: ["#   #", "#   #", "#   #", "#   #", "#   #", " ### "],
    V: ["#   #", "#   #", "#   #", "#   #", " # # ", "  #  "],
    W: ["#   #", "#   #", "#   #", "# # #", "## ##", "#   #"],
    X: ["#   #", " # # ", "  #  ", "  #  ", " # # ", "#   #"],
    Y: ["#   #", " # # ", "  #  ", "  #  ", "  #  ", "  #  "],
    Z: ["#####", "   # ", "  #  ", " #   ", "#    ", "#####"],
    0: [" ### ", "#   #", "#  ##", "# # #", "##  #", " ### "],
    1: ["  #  ", " ##  ", "# #  ", "  #  ", "  #  ", "#####"],
    2: [" ### ", "#   #", "   # ", "  #  ", " #   ", "#####"],
    3: [" ### ", "#   #", "   # ", "  ## ", "#   #", " ### "],
    4: ["   # ", "  ## ", " # # ", "#####", "   # ", "   # "],
    5: ["#####", "#    ", "#### ", "    #", "#   #", " ### "],
    6: [" ### ", "#    ", "#### ", "#   #", "#   #", " ### "],
    7: ["#####", "    #", "   # ", "  #  ", " #   ", "#    "],
    8: [" ### ", "#   #", " ### ", "#   #", "#   #", " ### "],
    9: [" ### ", "#   #", "#   #", " ####", "    #", " ### "],
    " ": ["   ", "   ", "   ", "   ", "   ", "   "],
    ".": ["   ", "   ", "   ", "   ", "   ", " # "],
    ",": ["   ", "   ", "   ", "   ", " # ", " # "],
    "!": [" # ", " # ", " # ", " # ", "   ", " # "],
    "?": [" ### ", "#   #", "   # ", "  #  ", "     ", "  #  "],
    "-": ["     ", "     ", "     ", "#####", "     ", "     "],
    "'": [" # ", " # ", "   ", "   ", "   ", "   "],
    '"': ["# #", "# #", "   ", "   ", "   ", "   "],
  };

  const chartRenderers = {
    text(refValue) {
      return [JSON.stringify(refValue)];
    },

    hbar(refValue, bc, budgetW) {
      return renderHBar(refValue, bc, budgetW, "hbar");
    },

    // v0.1 alias: `bar` keeps the original horizontal semantics by
    // delegating to the shared horizontal body. The name is passed
    // through so validation errors read `⚠ bar:` to match what the
    // user actually typed.
    bar(refValue, bc, budgetW) {
      return renderHBar(refValue, bc, budgetW, "bar");
    },

    // Literal-text banner (6-row block letters). `text` is already
    // uppercased and stripped of dataRefs by applyChartRenderer. The
    // ASCII theme is detected via bc reference equality with ASCII_BOX;
    // unknown runes fall back to '?' so fixtures stay deterministic.
    banner(text, bc) {
      if (text === "") return ["", "", "", "", "", ""];
      const font = bc === ASCII_BOX ? bannerFontASCII : bannerFontDefault;
      const rows = ["", "", "", "", "", ""];
      for (const r of text) {
        const glyph = font[r] || font["?"];
        for (let i = 0; i < 6; i++) rows[i] += glyph[i];
      }
      return rows;
    },

    // Vertical bars. Mirrors `hbar`'s 10-cell budget, but the budget is
    // the chart HEIGHT rather than width. Each series entry becomes a
    // single-column bar, centered within a column whose width accounts
    // for its label / `(value)` text. Footer rows carry the labels.
    vbar(refValue) {
      const err = validateBarSeries(refValue, "vbar");
      if (err) return err;
      const { labels, values, maxY } = prepareBarSeries(refValue);
      const colW = refValue.map((_, i) =>
        Math.max(labels[i].length, values[i].length, 3),
      );
      const budget = BAR_HEIGHT_DEFAULT;
      const barH = refValue.map((e) => {
        if (maxY === 0) return 0;
        return Math.round((e.y / maxY) * budget);
      });
      const rows = [];
      for (let r = 0; r < budget; r++) {
        const fromBottom = budget - r; // 1-based distance from baseline
        let line = "";
        for (let i = 0; i < refValue.length; i++) {
          const glyph = barH[i] >= fromBottom ? BAR_GLYPH : " ";
          line += padRow(glyph, colW[i], "center");
        }
        rows.push(line);
      }
      rows.push(labels.map((s, i) => padRow(s, colW[i], "center")).join(""));
      rows.push(values.map((s, i) => padRow(s, colW[i], "center")).join(""));
      return rows;
    },

    // Progress bars. Two shapes:
    //   - scalar: `[ 75 | progress ]` → single row "{bar-20} {pct-4}".
    //     applyChartRenderer passes the pre-parsed number through as
    //     `input` (wrapped `{ kind: "scalar", y }`).
    //   - series: `[ @ref | progress ]` → N rows, "{label} {bar} {pct}"
    //     per entry. Passed in as `{ kind: "series", series }` after
    //     validateBarSeries has cleared it.
    // Out-of-range y values silently clamp to [0, 100] inside the row
    // formatter; NaN maps to 0.
    progress(input, bc) {
      const [filled, empty] = progressGlyphs(bc);
      if (input && input.kind === "series") {
        return renderProgressSeries(input.series, filled, empty);
      }
      const y = input && typeof input.y === "number" ? input.y : 0;
      return [renderProgressRow(y, filled, empty)];
    },
  };

  // Inspect a renderer-tagged box and rewrite its items to either the
  // renderer's output rows or an inline `⚠ error` row. Returns the
  // rewritten items array; leaves non-renderer boxes untouched.
  //
  // `dataMap` is the resolved frontmatter `meta.data` (flat list or
  // map of named series). `budgetW` is the content-width budget the
  // containing box is willing to devote to the renderer's visuals.
  const applyChartRenderer = (box, dataMap, bc, budgetW) => {
    const name = box.renderer;
    const handler = chartRenderers[name];
    const items = box.items || [];
    const inlineError = (msg) => [{ type: "text", content: msg }];

    if (!handler) return inlineError(`\u26a0 unknown renderer: ${name}`);

    // Banner is literal-text only: concatenate Text items, ignore any
    // dataRefs (mirrors Go's collectBannerText). Uppercase before
    // handing to the renderer so font-table lookup is case-insensitive
    // at the source; unknown runes fall back to the '?' glyph inside
    // the handler itself.
    if (name === "banner") {
      const rawText = items
        .filter((it) => it && it.type === "text")
        .map((it) => it.content)
        .join("")
        .toUpperCase();
      const rendered = handler(rawText, bc);
      return rendered.map((r) => ({ type: "text", content: r }));
    }

    // A renderer-tagged body admits exactly one of:
    //   - a single dataRef item       (renderer sees refValue)
    //   - a single text/literal item  (renderer sees raw string)
    //   - a mix / multi-item body     (treat as raw text w/ first text)
    // Any other shape falls back to "raw string" path.
    const dataRefs = items.filter((it) => it && it.type === "dataRef");
    const hasRef = dataRefs.length > 0;

    // Progress has a scalar path (no dataRef): parse the literal text
    // as a number, then hand the renderer `{kind:"scalar", y}`. The
    // series path below reuses the shared bar-series validator with
    // "progress" threaded through so its shape errors read
    // `\u26a0 progress: expected [{x,y}]` etc.
    if (name === "progress" && !hasRef) {
      const rawText = items
        .filter((it) => it && it.type === "text")
        .map((it) => it.content)
        .join("");
      const y = parseProgressScalar(rawText);
      if (y === null) {
        return inlineError(`\u26a0 progress: expected number`);
      }
      const rendered = handler({ kind: "scalar", y }, bc);
      return rendered.map((r) => ({ type: "text", content: r }));
    }

    if (hasRef) {
      const ref = dataRefs[0];
      if (!dataMap || typeof dataMap !== "object") {
        return inlineError(`\u26a0 @${ref.name} not found`);
      }
      let refValue;
      // Flat list: only `@data` resolves.
      if (Array.isArray(dataMap)) {
        if (ref.name !== "data") {
          return inlineError(`\u26a0 @${ref.name} not found`);
        }
        refValue = dataMap;
      } else {
        if (!(ref.name in dataMap)) {
          return inlineError(`\u26a0 @${ref.name} not found`);
        }
        refValue = dataMap[ref.name];
      }
      // Progress series routes through validateBarSeries so its
      // shape diagnostics share the bar-family wire text -- just
      // with "progress" as the renderer name.
      if (name === "progress") {
        const err = validateBarSeries(refValue, "progress");
        if (err) return inlineError(err[0]);
        const rendered = handler({ kind: "series", series: refValue }, bc);
        return rendered.map((r) => ({ type: "text", content: r }));
      }
      const rendered = handler(refValue, bc, budgetW);
      return rendered
        ? rendered.map((r) => ({ type: "text", content: r }))
        : items;
    }

    // No dataRef. Text renderer keeps the literal content; others
    // reject the raw-string input.
    if (name !== "text") {
      return inlineError(`\u26a0 ${name}: use @ref`);
    }
    return items;
  };

  // Walk the AST *before* rendering; for any box carrying a renderer
  // tag, rewrite its items. Also flags any box that holds a dataRef
  // WITHOUT a renderer -- that's the "bare @ref" error case.
  const applyChartRenderers = (item, dataMap, bc, budgetW) => {
    if (!item || typeof item !== "object") return;
    if (item.type === "box") {
      if (item.renderer) {
        item.items = applyChartRenderer(item, dataMap, bc, budgetW);
      } else if (Array.isArray(item.items)) {
        const bareRef = item.items.find((it) => it && it.type === "dataRef");
        if (bareRef) {
          item.items = [
            {
              type: "text",
              content: `\u26a0 @${bareRef.name}: requires | renderer`,
            },
          ];
        }
      }
    }
    if (Array.isArray(item.items)) {
      for (const child of item.items)
        applyChartRenderers(child, dataMap, bc, budgetW);
    }
    if (item.type === "edge" && item.label) {
      applyChartRenderers(item.label, dataMap, bc, budgetW);
    }
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

    // Rendered-chart boxes originally held a single source item (a
    // dataRef or literal text); the post-rewrite items array is the
    // chart's output rows, which should not re-trigger natural padding.
    // Mirror Go: hasPad uses the SOURCE item count, so a `( X | banner )`
    // prints flush against the frame edges.
    const srcLen = ast.renderer ? 1 : items.length;
    const hasPad = bordered || srcLen > 1;
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

    // `meta.size` is a MAXIMUM, not a fixed frame. When content is
    // smaller than the declared size the box stays tight to its
    // content; when content is larger, the outer is clamped to the
    // declared size (flow chains have already wrapped via
    // `sizedContentW`; unwrappable text is clipped by `clipRow`).
    const outerW =
      targetW !== undefined ? Math.min(naturalOuterW, targetW) : naturalOuterW;
    const outerH =
      targetH !== undefined ? Math.min(naturalOuterH, targetH) : naturalOuterH;

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

  // Overlay right-side gutter arrows for cross-row refs. Called after
  // the regular render produces `rows`. For each cross-row ref on a
  // top-level row, draws an arc from the source row (dashes + corner
  // entering the gutter) to the target box's row (arrow entering the
  // box + dashes + corner). Widens all rows to accommodate the gutter.
  //
  // Scope is deliberately shallow -- only top-level items are tracked,
  // so a ref nested inside a nested box won't route. That matches the
  // current user-facing surface (multi-line source yields top-level
  // row items), and keeps the tracker free of deep position bookkeeping.
  const GUTTER_EXTRA = 3;
  const overlayCrossRowGutter = (rows, ast, bc) => {
    const items = ast.items || [];
    const hasCross = items.some(
      (it) =>
        it?.type === "row" &&
        it.items?.some((c) => c?.type === "ref" && c.crossRow),
    );
    if (!hasCross) return rows;

    const bordered = ast.bordered;
    const hasPad = bordered || items.length > 1;
    const padBudget = hasPad ? 2 * NATURAL_PAD : 0;
    const borderBudget = bordered ? 2 : 0;
    const targetW = ast.meta?.size?.w;
    const sizedContentW =
      targetW !== undefined
        ? Math.max(1, targetW - borderBudget - padBudget)
        : undefined;

    // Per-item render + absolute-start tracking (mirrors renderBoxRows'
    // stacking). Re-rendering is cheap relative to overall work and
    // keeps the overlay decoupled from the main render path.
    const perItemRows = new Map();
    const itemStart = new Map();
    let cursor = 0;
    for (const item of items) {
      const r = renderItemRows(item, bc, sizedContentW);
      perItemRows.set(item, r);
      itemStart.set(item, cursor);
      cursor += r.length;
    }

    const borderTop = bordered ? 1 : 0;
    const paddedRows = rows.length - 2 * borderTop;
    const extraTop = Math.max(0, Math.floor((paddedRows - cursor) / 2));
    const rowOffset = borderTop + extraTop;

    // Horizontal column math: `rows` are already padded to the root's
    // outer width. For each top-level item the renderer center-pads
    // its own width within the root's content area, so we re-derive
    // that leftPad here to translate inner columns into absolute ones.
    const rowWidth = rows.reduce((m, r) => Math.max(m, displayWidth(r)), 0);
    const borderLeft = bordered ? 1 : 0;
    const contentInnerW = rowWidth - 2 * borderLeft;
    const centerPad = (itemW) =>
      Math.max(0, Math.floor((contentInnerW - itemW) / 2));
    const absCol = (itemW, innerCol) =>
      borderLeft + centerPad(itemW) + innerCol;

    // Named-box absolute positions. A top-level named box sits
    // alone; a named box inside a row gets its startCol from the
    // cumulative linear widths before it.
    const boxAbs = new Map();
    for (const item of items) {
      const base = itemStart.get(item) + rowOffset;
      if (item.type === "box" && item.name) {
        const r = perItemRows.get(item);
        const itemW = r.reduce((m, x) => Math.max(m, displayWidth(x)), 0);
        boxAbs.set(item, {
          start: base,
          height: r.length,
          startCol: absCol(itemW, 0),
          width: itemW,
        });
      } else if (item.type === "row") {
        const linearItems = item.items.filter(
          (it) =>
            !(it && it.type === "ref" && (it.sameRowArc || it.crossRow)) &&
            !it._consumedBySameRowArc &&
            !it._consumedByCrossRow,
        );
        const parts = linearItems.map((it) => {
          if (it.type === "edge") {
            return {
              kind: "edge",
              item: it,
              width: displayWidth(edgeString(it, bc)),
            };
          }
          const subR = renderItemRows(it, bc);
          return {
            kind: "block",
            item: it,
            rows: subR,
            width: subR.reduce((m, x) => Math.max(m, displayWidth(x)), 0),
            height: subR.length,
          };
        });
        const rowLinearW = parts.reduce((s, p) => s + p.width, 0);
        let innerCol = 0;
        for (const p of parts) {
          if (p.kind === "block" && p.item.type === "box" && p.item.name) {
            boxAbs.set(p.item, {
              start: base,
              height: p.height,
              startCol: absCol(rowLinearW, innerCol),
              width: p.width,
            });
          }
          innerCol += p.width;
        }
        // Cache row metadata so the exits pass below doesn't
        // reconstruct parts a second time.
        item._linearParts = parts;
        item._rowLinearW = rowLinearW;
      }
    }

    // Collect cross-row exits with source midline + exit column.
    const exits = [];
    for (const item of items) {
      if (item.type !== "row") continue;
      const base = itemStart.get(item) + rowOffset;
      const parts = item._linearParts || [];
      const rowLinearW = item._rowLinearW ?? 0;
      const h = parts.reduce(
        (m, p) => (p.kind === "block" ? Math.max(m, p.height) : m),
        1,
      );
      const midlineOffset = Math.floor((h - 1) / 2);
      for (const ref of item.items) {
        if (ref?.type !== "ref" || !ref.crossRow) continue;
        const tgt = boxAbs.get(ref.target);
        if (!tgt) continue;
        exits.push({
          srcRow: base + midlineOffset,
          srcCol: absCol(rowLinearW, rowLinearW),
          tgtRow: tgt.start + Math.floor((tgt.height - 1) / 2),
          tgtCol: tgt.startCol + tgt.width,
        });
      }
      delete item._linearParts;
      delete item._rowLinearW;
    }
    if (exits.length === 0) return rows;

    const gutterCol = rowWidth + GUTTER_EXTRA - 1;
    const newWidth = gutterCol + 1;

    const grid = rows.map((r) => {
      const arr = Array.from(r);
      while (arr.length < newWidth) arr.push(" ");
      return arr;
    });

    for (const ex of exits) {
      for (let c = ex.srcCol; c < gutterCol; c++) grid[ex.srcRow][c] = bc.h;
      for (let c = ex.tgtCol + 1; c < gutterCol; c++) grid[ex.tgtRow][c] = bc.h;
      grid[ex.tgtRow][ex.tgtCol] = bc.arrowL;
      if (ex.srcRow > ex.tgtRow) {
        grid[ex.srcRow][gutterCol] = bc.br;
        grid[ex.tgtRow][gutterCol] = bc.tr;
        for (let r = ex.tgtRow + 1; r < ex.srcRow; r++)
          if (grid[r][gutterCol] === " ") grid[r][gutterCol] = bc.v;
      } else {
        grid[ex.srcRow][gutterCol] = bc.tr;
        grid[ex.tgtRow][gutterCol] = bc.br;
        for (let r = ex.srcRow + 1; r < ex.tgtRow; r++)
          if (grid[r][gutterCol] === " ") grid[r][gutterCol] = bc.v;
      }
    }
    return grid.map((arr) => arr.join(""));
  };

  // Top-level render entry: honors meta.size on the root node only.
  const renderRows = (ast) => {
    const bc = boxChars(ast);
    const size = ast.meta?.size;
    // One-shot rewrite for any `| renderer`-tagged boxes. Idempotent --
    // tagged boxes get their items replaced in place, so a second call
    // would re-apply to already-rendered text items (harmless but
    // pointless). We reset `renderer` on the box after consumption.
    // Bar width budget: default 10 cells; when the root has a narrower
    // explicit size, shrink. (Wider sizes stay at 10 -- the author
    // asked for a bar chart, not a giant.)
    const barBudget =
      size?.w !== undefined ? Math.min(BAR_WIDTH_DEFAULT, size.w) : undefined;
    applyChartRenderers(ast, ast.meta?.data, bc, barBudget);
    const rows = renderBoxRows(ast, bc, {
      targetW: size?.w,
      targetH: size?.h,
    });
    return overlayCrossRowGutter(rows, ast, bc);
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
        "<code>[&nbsp;&nbsp;x -]</code> left</div>" +
        '<div><span class="pylon-guide-key">name:</span> <code>[x :: foo]</code>' +
        '<span class="pylon-guide-sep">·</span>' +
        '<span class="pylon-guide-key">ref:</span> <code>&amp;foo</code></div>';

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
      const errors = ast._errors;
      if (errors?.length) {
        this._showToast(errors.join("\n"));
      } else {
        this._clearToast();
      }
    }

    // Component-scoped toast. Single instance: a new error replaces
    // the previous toast and resets the auto-dismiss timer. The
    // message is compared to the currently shown text so rapid edits
    // that keep triggering the same error don't cause a visible
    // flicker each keystroke.
    _showToast(message) {
      if (this._toastEl && this._toastEl.textContent === message) {
        this._resetToastTimer();
        return;
      }
      this._clearToast();
      const toast = document.createElement("div");
      toast.className = "pylon-toast";
      toast.setAttribute("role", "alert");
      toast.textContent = message;
      this.append(toast);
      this._toastEl = toast;
      this._resetToastTimer();
    }

    _resetToastTimer() {
      if (this._toastTimer) clearTimeout(this._toastTimer);
      this._toastTimer = setTimeout(() => this._clearToast(), 3200);
    }

    _clearToast() {
      if (this._toastTimer) {
        clearTimeout(this._toastTimer);
        this._toastTimer = null;
      }
      if (this._toastEl) {
        this._toastEl.remove();
        this._toastEl = null;
      }
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
