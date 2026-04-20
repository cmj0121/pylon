// Pylon -- tiny custom-element scaffold
//
// Registers a <pylon-chart> custom element. Renders its source
// (textContent or the `src` attribute) in one of three backends:
//
//   <pylon-chart>Hello, World!</pylon-chart>                  // ASCII (default)
//   <pylon-chart format="svg">Hello, World!</pylon-chart>     // inline SVG
//   <pylon-chart format="png">Hello, World!</pylon-chart>     // canvas -> <img>
//
// (The HTML spec requires custom element names to contain a hyphen,
// hence `pylon-chart` rather than `pylon`.)
//
// The parser is a stub for now: the source text becomes the single
// label on a drawn box. Wire in the real parser once the grammar is
// settled.

(() => {
  // ---- stub parser ------------------------------------------------------
  // Replace with the real Pylon parser later. For now, any input becomes
  // a single-box AST whose label is the trimmed source (or "Hello, World!"
  // when the source is empty).
  const parse = (source) => {
    const label = (source ?? "").trim() || "Hello, World!";
    return { type: "box", label };
  };

  // ---- renderers --------------------------------------------------------
  const renderers = {
    ascii(ast) {
      const label = ast.label;
      const border = "+" + "-".repeat(label.length + 2) + "+";
      const body = "| " + label + " |";
      const pre = document.createElement("pre");
      pre.className = "pylon-ascii";
      pre.textContent = [border, body, border].join("\n");
      return pre;
    },

    svg(ast) {
      const label = ast.label;
      const w = label.length * 10 + 24;
      const h = 44;
      const ns = "http://www.w3.org/2000/svg";
      const svg = document.createElementNS(ns, "svg");
      svg.setAttribute("width", w);
      svg.setAttribute("height", h);
      svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
      svg.classList.add("pylon-svg");
      const rect = document.createElementNS(ns, "rect");
      rect.setAttribute("x", 1);
      rect.setAttribute("y", 1);
      rect.setAttribute("width", w - 2);
      rect.setAttribute("height", h - 2);
      rect.setAttribute("fill", "none");
      rect.setAttribute("stroke", "currentColor");
      rect.setAttribute("stroke-width", "1");
      svg.append(rect);
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
      const label = ast.label;
      // Canvas 2D does not understand the "currentColor" keyword, so the
      // caller resolves the computed color and passes it in.
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
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = color;
      ctx.strokeRect(0.75, 0.75, w - 1.5, h - 1.5);
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
  class PylonElement extends HTMLElement {
    static get observedAttributes() {
      return ["format", "src"];
    }

    constructor() {
      super();
      this._source = "";
      this._viewHost = null;
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
      this._mount();
    }

    _mount() {
      this.innerHTML = "";
      this._viewHost = document.createElement("div");
      this._viewHost.className = "pylon-view";
      this.append(this._viewHost);
      this._render();
    }

    _render() {
      if (!this._viewHost) return;
      const format = (this.getAttribute("format") ?? "ascii").toLowerCase();
      const renderer = renderers[format] ?? renderers.ascii;
      const ast = parse(this._source);
      // Resolve currentColor so canvas-based renderers (PNG) can use it.
      const color = getComputedStyle(this._viewHost).color;
      this._viewHost.innerHTML = "";
      this._viewHost.append(renderer(ast, { color }));
    }
  }

  if (!customElements.get("pylon-chart")) {
    customElements.define("pylon-chart", PylonElement);
  }
})();
