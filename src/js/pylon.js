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

  // ---- renderers --------------------------------------------------------
  const renderers = {
    ascii(ast) {
      const { label, bordered } = ast;
      const pre = document.createElement("pre");
      pre.className = "pylon-ascii";
      if (bordered) {
        const border = "+" + "-".repeat(label.length + 2) + "+";
        const body = "| " + label + " |";
        pre.textContent = [border, body, border].join("\n");
      } else {
        pre.textContent = label;
      }
      return pre;
    },

    svg(ast) {
      const { label, bordered } = ast;
      const w = label.length * 10 + 24;
      const h = 44;
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
      const w = label.length * 10 + 24;
      const h = 44;
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
        ctx.strokeRect(0.75, 0.75, w - 1.5, h - 1.5);
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
  }

  if (!customElements.get("pylon-chart")) {
    customElements.define("pylon-chart", PylonElement);
  }
})();
