// Headless ASCII renderer used by the cross-implementation parity gate.
//
// Loads dist/pylon.min.js through a minimal DOM shim, renders the given
// .pylon file in ascii mode, and prints the result to stdout with no
// trailing newline (matching Go's pylon.RenderASCII output).
//
// Usage: node scripts/pylon-render-js.mjs <path/to/fixture.pylon>
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const bundle = resolve(here, "../dist/pylon.min.js");

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

new Function(readFileSync(bundle, "utf8"))();

const Element = globalThis.customElements._defs["pylon-chart"];
if (!Element) {
  console.error("pylon-chart custom element not registered");
  process.exit(2);
}

const path = process.argv[2];
if (!path) {
  console.error("usage: pylon-render-js.mjs <file.pylon>");
  process.exit(2);
}

const src = readFileSync(path, "utf8");

const el = new Element();
el._source = src;
el._viewHost = new VNode("div");
el._format = "ascii";
el._mount = () => {};
el._render();

const pre = el._viewHost.children.find((c) => c.tagName === "pre");
if (!pre) {
  console.error("renderer produced no <pre> output");
  process.exit(2);
}

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
for (const c of pre.children) emit(c);

// Strip a single trailing newline if the harness/renderer added one --
// Go's RenderASCII emits no trailing newline, and the parity gate compares
// byte-for-byte.
if (out.endsWith("\n")) out = out.slice(0, -1);

process.stdout.write(out);
