# Pylon roadmap

Tracks the gaps between the current `HEAD` and a shippable v1.0.0.
Entries are grouped by category; each carries a status and a pointer
back to the SPEC, LSP doc, or source file that already acknowledges
the limitation. Use this as the live source of truth for "what's
left"; annotated tag messages remain the record of what actually
shipped.

`Status` values:

- **closing** — work in flight on a named branch.
- **closed** — shipped; kept in the list as a historical marker so
  future reviewers can trace when each gap was resolved.
- **deferred** — known, scoped, and intentionally out of the current
  cut.
- **documented** — limitation is acknowledged in SPEC; any change is
  a scope expansion, not a bug fix.

## Functional parity

1. **Go CLI chart rendering — CLOSED.** All renderers ship in the Go
   CLI; the parity gate (`scripts/pylon-parity.sh`) locks every
   fixture under `src/go/pkg/pylon/testdata/` byte-for-byte against
   the JS reference. The "same source renders identically" promise
   holds for flow diagrams and charts alike.
   Status: **closed**.
   Ref: [`../src/go/pkg/pylon/render_chart.go`](../src/go/pkg/pylon/render_chart.go).

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

6. **`--strict` CLI flag — CLOSED.** `pylon --strict` exits `2`
   when `Validate` produced any diagnostics; default remains `0`
   so existing scripts are unaffected. Closes the CI-gating gap.
   Status: **closed**.
   Ref: [`../src/go/cmd/pylon/main.go`](../src/go/cmd/pylon/main.go),
   [`LSP.md` §CLI diagnostic mode](LSP.md#cli-diagnostic-mode).

7. **`| banner` JS parity — CLOSED.** The block-letter banner
   renderer ships in both `src/go/pkg/pylon/render_banner.go` and
   `src/js/pylon.js` with five font tables (default, ASCII,
   monospace, digital, mini) inlined byte-for-byte. Banner
   fixtures sit under `src/go/pkg/pylon/testdata/` and participate
   in `scripts/pylon-parity.sh`. Shipped on `feat/banner-js`.
   Status: **closed**.
   Ref: [`../src/go/pkg/pylon/render_banner.go`](../src/go/pkg/pylon/render_banner.go),
   [`../src/js/pylon.js`](../src/js/pylon.js).

## Release engineering

1. **Release workflow — CLOSED.** `.github/workflows/release.yml`
   ships pre-built `pylon` and `pylon-lsp` binaries for 4 platforms
   (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`)
   on every `v*` tag push via goreleaser, with SHA256 checksums and
   tar.gz archives. Pre-release tags (e.g. `v0.3.0-rc.1`) land as
   GitHub pre-releases automatically. Windows and other platforms
   remain `go install` territory. Status: **closed**.
   Ref: [`../.goreleaser.yaml`](../.goreleaser.yaml),
   [`../.github/workflows/release.yml`](../.github/workflows/release.yml).

2. **`--version` flag — CLOSED.** Both `pylon` and `pylon-lsp`
   print a build-time version string (populated by the Makefile
   via `git describe --tags --always --dirty`). `go install` users
   without the Makefile get the literal `dev` fallback — a proper
   release workflow (gap #1 in this section) will ship pre-built
   binaries with real version strings baked in. Status: **closed**.
   Ref: [`../src/go/cmd/pylon/main.go`](../src/go/cmd/pylon/main.go),
   [`../src/go/cmd/pylon-lsp/main.go`](../src/go/cmd/pylon-lsp/main.go),
   [`../src/go/Makefile`](../src/go/Makefile).

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
