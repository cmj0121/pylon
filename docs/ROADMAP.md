# Pylon roadmap

Tracks the gaps between the current `HEAD` and a shippable v1.0.0.
Entries are grouped by category; each carries a status and a pointer
back to the SPEC, LSP doc, or source file that already acknowledges
the limitation. Use this as the live source of truth for "what's
left"; annotated tag messages remain the record of what actually
shipped.

`Status` values:

- **closing** — work in flight on a named branch.
- **deferred** — known, scoped, and intentionally out of the current
  cut.
- **documented** — limitation is acknowledged in SPEC; any change is
  a scope expansion, not a bug fix.

## Functional parity

1. **Go CLI chart rendering.** `| bar`, `| hbar`, `| vbar`, and
   `| text` are parsed by the Go AST but unrendered, so the "same
   source renders identically in the browser, on the command line,
   and inside your editor" promise only holds for flow diagrams.
   Status: **closing** in `feat/go-chart-renderers`.
   Ref: [`SPEC.md` §Known limitations](SPEC.md#known-limitations),
   [`../src/go/README.md` §Known limitations](../src/go/README.md#known-limitations).

2. **LSP semantic tokens — partial.** Only `[`, `]`, `(`, `)`,
   `&ref`, and `@ref` are emitted today. Edges, frontmatter
   keys/numbers/strings, `:: name`, `| renderer`, alignment dashes,
   and the `---` fence lines still fall back to the Vim regex
   plugin. Status: deferred to `feat/pylon-lsp-ux`.
   Ref: [`LSP.md` §Semantic tokens](LSP.md#semantic-tokens).

3. **LSP navigation surface missing.** No hover, go-to-definition,
   completion, or rename across `&ref` / `@ref` / `:: name`. These
   are the features editor users expect once diagnostics work; v1
   ships without them. Status: deferred to `feat/pylon-lsp-ux`.
   Ref: [`LSP.md` §Roadmap](LSP.md#roadmap).

4. **Document-symbol per-key spans.** A `data:` block with multiple
   series (`@counter`, `@sales`, …) reports identical ranges for
   every key because the frontmatter parser does not track per-key
   spans. Status: deferred — requires parser changes.
   Ref: [`LSP.md` §Document symbols](LSP.md#document-symbols).

5. **Editor packaging is Vim-only.** The `pylon-lsp` binary works
   with any LSP client, but only the Neovim Lua loader ships in
   tree. VS Code and Zed extensions are pending. Status: deferred to
   `feat/pylon-lsp-ux`.
   Ref: [`LSP.md` §Editor setup](LSP.md#editor-setup).

6. **No `--strict` CLI flag.** The `pylon` CLI prints diagnostics
   to stderr but always exits `0`; CI users cannot gate a build on
   Pylon diagnostics without parsing stderr. Status: deferred.
   Ref: [`LSP.md` §CLI diagnostic mode](LSP.md#cli-diagnostic-mode).

## Release engineering

1. **No release workflow.** `.github/workflows/` contains only
   `ci.yml` (PR-triggered) and `pages.yml` (tag-triggered, Pages
   only). No cross-platform binary build and no `gh release` asset
   upload; every user needs Go 1.25+ to install the CLI. Status:
   deferred.

2. **No `--version` flag.** Neither `pylon` nor `pylon-lsp` reports
   a build-time version. Bug reports cannot pin a build. Status:
   deferred.

3. **No npm / JS package.** `src/js/` has no `package.json`; web
   consumers copy `dist/pylon.min.js` and `dist/pylon.css` by hand.
   No npm, unpkg, or jsDelivr path for the custom element. Status:
   deferred.

4. **No `CHANGELOG.md`.** Release notes live only in annotated tag
   messages (`git show v0.2.0`). Fine for two releases, poor at
   v1.0 scale. Status: deferred.

## Project-health polish

1. **Missing contributor docs.** No `CONTRIBUTING.md`,
   `CODE_OF_CONDUCT.md`, or `SECURITY.md` at the repo root. Standard
   for an MIT-licensed public repo that invites external PRs.
   Status: deferred.

2. **CI runs only on `pull_request`.** No main-branch build, no
   nightly parity sweep, no build artifacts attached to commits
   for bisect. Status: deferred.
   Ref: [`../.github/workflows/ci.yml`](../.github/workflows/ci.yml).

3. **No SVG/PNG parity gate.** `scripts/pylon-parity.sh` only
   diffs ASCII output between the Go CLI and the JS reference; a
   future change that breaks SVG or PNG layout without affecting
   ASCII would pass CI silently. Surfaced by tenth-man review of
   `feat/go-chart-renderers`. Status: deferred.
   Ref: [`../scripts/pylon-parity.sh`](../scripts/pylon-parity.sh).

4. **PNG falls back to tofu on CJK / emoji.** The embedded font is
   JetBrains Mono only; glyphs outside its coverage render as
   replacement characters. Either ship a fallback font chain or
   make the "Latin-only" scope explicit at v1.0. Status:
   documented.
   Ref: [`SPEC.md` §Known limitations](SPEC.md#known-limitations).
