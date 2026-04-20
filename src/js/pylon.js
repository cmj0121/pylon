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
  // a single-box AST:
  //   [ ... ]   bordered node
  //   ( ... )   borderless node
  //   otherwise plain text in a bordered node
  // Leading and trailing '-' inside the brackets are alignment markers and
  // are stripped. Empty input falls back to the default example.
  const DEFAULT_EXAMPLE = "[- Pylon WYSIWYG -]";

  const parse = (source) => {
    const raw = (source ?? "").trim() || DEFAULT_EXAMPLE;

    let bordered = true;
    let inner = raw;
    if (raw.startsWith("[") && raw.endsWith("]")) {
      bordered = true;
      inner = raw.slice(1, -1);
    } else if (raw.startsWith("(") && raw.endsWith(")")) {
      bordered = false;
      inner = raw.slice(1, -1);
    }

    const label = inner
      .replace(/^\s*-\s+/, "")
      .replace(/\s+-\s*$/, "")
      .trim();

    return { type: "box", label, bordered };
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

  // ---- renderers --------------------------------------------------------
  const renderers = {
    ascii(ast) {
      const { label, bordered } = ast;
      const pre = document.createElement("pre");
      pre.className = "pylon-ascii";
      if (!bordered) {
        pre.textContent = label;
        return pre;
      }
      // Unicode box-drawing glyphs + 3-col horizontal padding. Every
      // character is placed in its own fixed-width cell so the border
      // stays aligned regardless of how the browser's fallback font
      // renders CJK or emoji. The raw text (┌─┐│└┘) is preserved in
      // the DOM so selecting and copying the output yields usable
      // ASCII art that can be pasted into a terminal or markdown.
      const dw = displayWidth(label);
      const pad = 3;
      const line = "─".repeat(dw + pad * 2);
      const sp = " ".repeat(pad);
      const rows = [
        "┌" + line + "┐",
        "│" + sp + label + sp + "│",
        "└" + line + "┘",
      ];
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
      const { label, bordered } = ast;
      const dw = displayWidth(label);
      const w = dw * 10 + 32;
      const h = 48;
      const ns = "http://www.w3.org/2000/svg";
      const svg = document.createElementNS(ns, "svg");
      svg.setAttribute("width", w);
      svg.setAttribute("height", h);
      svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
      svg.classList.add("pylon-svg");
      if (bordered) {
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
      const t = document.createElementNS(ns, "text");
      t.setAttribute("x", w / 2);
      t.setAttribute("y", h / 2);
      t.setAttribute("text-anchor", "middle");
      t.setAttribute("dominant-baseline", "middle");
      t.setAttribute("font-family", "monospace");
      t.setAttribute("font-size", "14");
      t.setAttribute("fill", "currentColor");
      t.textContent = label;
      svg.append(t);
      return svg;
    },

    png(ast, opts = {}) {
      const { label, bordered } = ast;
      const color = opts.color || "#000";
      const dw = displayWidth(label);
      const w = dw * 10 + 32;
      const h = 48;
      const dpr = window.devicePixelRatio || 1;
      const canvas = document.createElement("canvas");
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      canvas.style.width = w + "px";
      canvas.style.height = h + "px";
      const ctx = canvas.getContext("2d");
      ctx.scale(dpr, dpr);
      if (bordered) {
        ctx.lineWidth = 1.5;
        ctx.strokeStyle = color;
        if (ctx.roundRect) {
          ctx.beginPath();
          ctx.roundRect(0.75, 0.75, w - 1.5, h - 1.5, 3);
          ctx.stroke();
        } else {
          ctx.strokeRect(0.75, 0.75, w - 1.5, h - 1.5);
        }
      }
      ctx.fillStyle = color;
      ctx.font = "14px ui-monospace, Menlo, Consolas, monospace";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(label, w / 2, h / 2);
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
      const { label, bordered } = ast;
      if (!bordered) {
        return { text: label, mime: "text/plain", ext: "txt" };
      }
      const dw = displayWidth(label);
      const pad = 3;
      const line = "─".repeat(dw + pad * 2);
      const sp = " ".repeat(pad);
      const text = [
        "┌" + line + "┐",
        "│" + sp + label + sp + "│",
        "└" + line + "┘",
      ].join("\n");
      return { text, mime: "text/plain", ext: "txt" };
    },

    svg(ast) {
      const svg = renderers.svg(ast);
      const text =
        '<?xml version="1.0" encoding="UTF-8"?>\n' +
        new XMLSerializer().serializeToString(svg);
      return { text, mime: "image/svg+xml", ext: "svg" };
    },

    async png(ast, opts = {}) {
      const { label, bordered } = ast;
      const color = opts.color || "#000";
      const dw = displayWidth(label);
      const w = dw * 10 + 32;
      const h = 48;
      const dpr = window.devicePixelRatio || 1;
      const canvas = document.createElement("canvas");
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      const ctx = canvas.getContext("2d");
      ctx.scale(dpr, dpr);
      if (bordered) {
        ctx.lineWidth = 1.5;
        ctx.strokeStyle = color;
        if (ctx.roundRect) {
          ctx.beginPath();
          ctx.roundRect(0.75, 0.75, w - 1.5, h - 1.5, 3);
          ctx.stroke();
        } else {
          ctx.strokeRect(0.75, 0.75, w - 1.5, h - 1.5);
        }
      }
      ctx.fillStyle = color;
      ctx.font = "14px ui-monospace, Menlo, Consolas, monospace";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(label, w / 2, h / 2);
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

      // Right pane: toolbar + rendered output.
      const right = document.createElement("div");
      right.className = "pylon-right";

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

      right.append(toolbar, this._viewHost);
      this.append(right);
    }

    _render() {
      if (!this._viewHost) return;
      const renderer = renderers[this._currentFormat()] ?? renderers.ascii;
      const ast = parse(this._source);
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
