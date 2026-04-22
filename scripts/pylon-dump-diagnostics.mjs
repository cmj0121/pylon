// pylon-dump-diagnostics.mjs — JS-side counterpart to
// cmd/pylon-dump-diagnostics for the cross-implementation
// diagnostic-set parity gate.
//
// Loads dist/pylon.min.js through the same VNode DOM shim
// src/js/smoke.mjs uses, renders a single .pylon input via the
// pylon-chart custom element, and captures:
//
//   * root._errors — the toast-path array (surfaced by the element
//     via _toastEl.textContent, newline-joined when multiple)
//   * inline `⚠` lines scanned from the rendered output
//
// Emits JSON to stdout in the canonical shape:
//
//   { "errors": [...], "inlineWarnings": [...] }
//
// Both arrays are sorted lexicographically before emit so array-order
// drift can't false-positive the diff.
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const bundle = resolve(here, "../dist/pylon.min.js");

// Minimal DOM shim — matches src/js/smoke.mjs so the bundle has
// enough surface to register the custom element and render into
// in-memory VNodes.
class VNode {
  constructor(tag) {
    this.tagName = tag;
    this.children = [];
    this.attrs = {};
    this.dataset = {};
    this.style = {};
    this.innerHTML = "";
    this._textContent = "";
    this._value = "";
    this.classList = {
      add: (c) => (this.className = ((this.className || "") + " " + c).trim()),
      contains: () => false,
    };
  }
  setAttribute(k, v) {
    this.attrs[k] = v;
  }
  getAttribute(k) {
    return this.attrs[k] ?? null;
  }
  hasAttribute(k) {
    return k in this.attrs;
  }
  removeAttribute(k) {
    delete this.attrs[k];
  }
  remove() {}
  append(...kids) {
    for (const k of kids) this.children.push(k);
  }
  appendChild(k) {
    this.children.push(k);
  }
  addEventListener() {}
  get value() {
    return this._value;
  }
  set value(v) {
    this._value = v;
  }
  get textContent() {
    return this._textContent ?? "";
  }
  set textContent(v) {
    this._textContent = v;
  }
}

globalThis.window = globalThis;
globalThis.HTMLElement = VNode;
globalThis.customElements = {
  _defs: {},
  get(n) {
    return this._defs[n];
  },
  define(n, c) {
    this._defs[n] = c;
  },
};
globalThis.document = {
  createElement: (t) => new VNode(t),
  createTextNode: (t) => {
    const n = new VNode("#text");
    n._textContent = t;
    return n;
  },
};
globalThis.getComputedStyle = () => ({ color: "#0f1c2d" });
globalThis.requestAnimationFrame = (cb) => setTimeout(cb, 0);
globalThis.cancelAnimationFrame = (id) => clearTimeout(id);

// Run the bundle in a fresh scope that still sees globalThis.
new Function(readFileSync(bundle, "utf8"))();

const Element = globalThis.customElements._defs["pylon-chart"];
if (!Element) {
  console.error("pylon-dump-diagnostics: pylon-chart element not registered");
  process.exit(1);
}

// renderText mirrors the helper in src/js/smoke.mjs: render the source
// via the custom element, collect the text content of the output
// <pre>, and return it.
function renderText(src) {
  const el = new Element();
  el._source = src;
  el._viewHost = new VNode("div");
  el._format = "ascii";
  el._mount = () => {};
  el._render();
  const pre = el._viewHost.children.find((c) => c.tagName === "pre");
  let out = "";
  const emit = (n) => {
    if (!n) return;
    if (n.tagName === "#text") {
      out += n._textContent;
      return;
    }
    if (n.children?.length) {
      for (const c of n.children) emit(c);
      return;
    }
    out += n._textContent || "";
  };
  if (pre) {
    for (const c of pre.children) emit(c);
  }
  // Toast carries the concatenated toast-path errors.
  const toast = el.children?.find((c) => c.className === "pylon-toast");
  const toastText = toast ? toast.textContent : "";
  return { rendered: out, toastText };
}

// extractInlineWarning pulls the `⚠ …` message out of a single line
// of rendered output. The rendered box wraps the message in box
// glyphs + padding (`│   ⚠ msg   │`); this strips both sides back to
// the bare message.
function extractInlineWarning(line) {
  const i = line.indexOf("⚠");
  if (i < 0) return null;
  const tail = line.slice(i);
  // Walk backward past trailing spaces and any box side / theme-ascii
  // side characters to find the real message end.
  let end = tail.length;
  while (end > 0) {
    const c = tail[end - 1];
    if (c === " " || c === "│" || c === "|") {
      end--;
      continue;
    }
    break;
  }
  return tail.slice(0, end);
}

// collectDiagnostics returns { errors, inlineWarnings } for a single
// pylon source string.
function collectDiagnostics(src) {
  const { rendered, toastText } = renderText(src);
  const errors = toastText ? toastText.split("\n").filter(Boolean) : [];
  const inlineWarnings = [];
  for (const line of rendered.split("\n")) {
    const w = extractInlineWarning(line);
    if (w !== null) inlineWarnings.push(w);
  }
  return {
    errors: [...new Set(errors)].sort(),
    inlineWarnings: [...new Set(inlineWarnings)].sort(),
  };
}

// Entry point: read input (file path arg or `-` for stdin), dump JSON.
const inputArg = process.argv[2];
let source;
if (!inputArg || inputArg === "-") {
  source = readFileSync(0, "utf8");
} else {
  source = readFileSync(inputArg, "utf8");
}

const diag = collectDiagnostics(source);
process.stdout.write(JSON.stringify(diag, null, 2) + "\n");
