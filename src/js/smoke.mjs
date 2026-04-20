// Smoke-test the minified bundle.
//
// Loads dist/pylon.min.js through a minimal DOM shim, renders a small
// set of fixtures, and exits non-zero if any assertion fails. Assertions
// are structural (row count, glyph presence) rather than exact strings
// so rendering tweaks don't tax fixtures.
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const bundle = resolve(here, "../../dist/pylon.min.js");

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

// Function-constructor eval so the bundle runs in a fresh scope but
// still sees globalThis (where our shim lives).
new Function(readFileSync(bundle, "utf8"))();

const Element = globalThis.customElements._defs["pylon-chart"];
if (!Element) {
  console.error("FAIL: pylon-chart custom element not registered");
  process.exit(1);
}

const render = (src) => {
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
  for (const c of pre.children) emit(c);
  // Append any toast text so fixtures can assert on surfaced errors.
  const toast = el.children?.find((c) => c.className === "pylon-toast");
  if (toast) out += "\n[toast] " + toast.textContent;
  return out;
};

const fixtures = [
  {
    name: "single bordered node",
    src: "[- a -]",
    expect: (o) =>
      o.includes("┌") && o.includes("│   a   │") && o.includes("└"),
  },
  {
    name: "flow chain with two forward arrows",
    src: "[ Start ] -> [ Process ] -> [ End ]",
    expect: (o) =>
      (o.match(/─▶/g) || []).length === 2 &&
      ["Start", "Process", "End"].every((s) => o.includes(s)),
  },
  {
    name: "same-row self-loop renders under the box",
    src: "[- a :: x -] -> &x",
    expect: (o) => o.includes("▲") && o.includes("└───┘"),
  },
  {
    name: "two cross-box arcs nest into compact 3-row bundle",
    src: "[- a :: x -] -> [- b :: y -] -> &x -> &y",
    // 3 box rows + 3 arc rows is the compact shape. Tolerate ±1 so a
    // future vertical-padding nudge doesn't falsely fail.
    expect: (o) => {
      const lines = o.split("\n");
      return (
        lines.length >= 5 &&
        lines.length <= 7 &&
        (o.match(/▲/g) || []).length === 2 &&
        o.includes("└──────────────┘")
      );
    },
  },
  {
    name: "self-loop hops above when mixed with cross-box",
    src: "[- a :: x -] -> [- b -] -> &x -> &x",
    expect: (o) =>
      o.includes("▼") && /^\s+┌/.test(o) && o.includes("└──────────┘"),
  },
  {
    name: "ascii theme swaps unicode glyphs for + - | < >",
    src: "---\ntheme: ascii\n---\n[- Hello -]",
    expect: (o) =>
      o.includes("+") && o.includes("|   Hello   |") && !o.includes("┌"),
  },
  {
    name: "text renderer stringifies a resolved @ref",
    src: "---\ndata:\n  - x: 1\n    y: 10\n---\n[- @data | text -]",
    expect: (o) => o.includes('[{"x":1,"y":10}]'),
  },
  {
    name: "bar renderer draws horizontal bars with value labels",
    src: "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n  - x: 3\n    y: 15\n---\n[ @data | bar ]",
    expect: (o) => {
      const bars = (o.match(/█/g) || []).length;
      return (
        bars >= 12 &&
        o.includes("(10)") &&
        o.includes("(20)") &&
        o.includes("(15)")
      );
    },
  },
  {
    name: "bar renderer rejects raw-string input inline",
    src: "[ hello | bar ]",
    expect: (o) => o.includes("⚠ bar: use @ref"),
  },
  {
    name: "bar renderer accepts unquoted-string x labels",
    src: "---\ndata:\n  - x: apples\n    y: 10\n  - x: bananas\n    y: 20\n---\n[ @data | bar ]",
    expect: (o) => {
      const bars = (o.match(/█/g) || []).length;
      return (
        bars >= 10 &&
        o.includes("  apples") &&
        o.includes("bananas") &&
        o.includes("(10)") &&
        o.includes("(20)")
      );
    },
  },
  {
    name: "bar renderer rejects duplicate x values inline",
    src: "---\ndata:\n  - x: a\n    y: 2\n  - x: a\n    y: 10\n---\n[ @data | bar ]",
    // The error prefix mirrors the renderer name the user actually
    // typed -- `bar` stays `bar`, it does not leak the `hbar` alias
    // target.
    expect: (o) => o.includes('⚠ bar: duplicate x "a"') && !o.includes("█"),
  },
  {
    name: "tab-indented data: frontmatter rejects with toast",
    src: "---\ndata:\n\t- x: 1\n\t  y: 10\n---\n[- @data | text -]",
    expect: (o) =>
      o.includes("[toast]") && o.includes("Unsupported data: frontmatter"),
  },
  {
    name: "vbar renderer draws vertical bars",
    src: "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n  - x: 3\n    y: 15\n---\n[ @data | vbar ]",
    expect: (o) => {
      if (!o.includes("\u2588")) return false;
      if (!o.includes("(10)") || !o.includes("(20)") || !o.includes("(15)")) {
        return false;
      }
      // x labels "1", "2", "3" on the line above the (y) value row.
      const lines = o.split("\n");
      const valLine = lines.findIndex((l) => l.includes("(10)"));
      if (valLine < 1) return false;
      const labelLine = lines[valLine - 1];
      return (
        labelLine.includes("1") &&
        labelLine.includes("2") &&
        labelLine.includes("3")
      );
    },
  },
  {
    name: "bar is an alias for hbar",
    src: null,
    expect: () => true,
    // Custom runner: rendered twice, compared.
    custom: () => {
      const base =
        "---\ndata:\n  - x: 1\n    y: 10\n  - x: 2\n    y: 20\n---\n";
      const a = render(base + "[ @data | bar ]");
      const b = render(base + "[ @data | hbar ]");
      return a === b;
    },
  },
  {
    name: "vbar renderer rejects duplicate x values inline",
    src: "---\ndata:\n  - x: a\n    y: 2\n  - x: a\n    y: 10\n---\n[ @data | vbar ]",
    expect: (o) =>
      o.includes('\u26a0 vbar: duplicate x "a"') && !o.includes("\u2588"),
  },
];

let failed = 0;
for (const f of fixtures) {
  try {
    if (f.custom) {
      if (!f.custom()) {
        failed++;
        console.error(`FAIL: ${f.name}`);
      } else {
        console.log(`ok  : ${f.name}`);
      }
      continue;
    }
    const out = render(f.src);
    if (!f.expect(out)) {
      failed++;
      console.error(`FAIL: ${f.name}`);
      console.error("  source: " + JSON.stringify(f.src));
      console.error(
        "  output:\n" +
          out
            .split("\n")
            .map((l) => "    " + l)
            .join("\n"),
      );
    } else {
      console.log(`ok  : ${f.name}`);
    }
  } catch (err) {
    failed++;
    console.error(`FAIL: ${f.name}: ${err.message}`);
  }
}
if (failed > 0) {
  console.error(`\n${failed} of ${fixtures.length} fixtures failed`);
  process.exit(1);
}
console.log(`\n${fixtures.length} fixtures passed`);
