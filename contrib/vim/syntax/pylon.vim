" Vim syntax file
" Language: Pylon diagram source
" Maintainer: cmj <cmj0121@gmail.com>
" Filenames: *.pylon
"
" NOTE: this file duplicates grammar rules from src/js/pylon.js. The
" canonical parser is the JS side; treat this syntax file as a best-effort
" visual aid that may drift. Update both when the grammar changes.

if exists("b:current_syntax")
  finish
endif

" Always sync syntax from start-of-file. The frontmatter region below uses
" `\%^` (start-of-file) as its start anchor; without `fromstart`, Vim may
" begin syncing mid-file on long buffers and fail to match that anchor,
" dropping frontmatter highlighting entirely.
syntax sync fromstart

" Case-sensitive: all Pylon identifiers and sigils are case-sensitive.
syntax case match

" ---- Frontmatter region --------------------------------------------------
" A YAML-subset block delimited by `---` fences at the very top of the file.
" `\%^` anchors to start-of-file so only the leading fence opens the region.
" Keys and scalars are contained children so they only match inside.
syntax region pylonFrontmatter
      \ matchgroup=pylonFrontmatterFence
      \ start=/\%^---[ \t]*$/
      \ end=/^---[ \t]*$/
      \ contains=pylonFMKey,pylonFMNumber,pylonFMString,pylonFMListDash

" Key inside frontmatter: leading whitespace + identifier flush to a `:`.
" Matches `size:`, `theme:`, `data:`, plus any nested `x:` / `y:` / series name.
syntax match pylonFMKey /^\s*\zs[A-Za-z_][A-Za-z0-9_]*\ze\s*:/ contained

" List dash inside frontmatter (for `data:` flat-series entries).
syntax match pylonFMListDash /^\s*\zs-\ze\s/ contained

" Numeric scalar: integer or decimal, optionally signed.
syntax match pylonFMNumber /-\?\<\d\+\%(\.\d\+\)\?\>/ contained

" Double-quoted string scalar (JS parser rejects escapes and embedded
" quotes, so the subset is safe to match with a flat character class).
syntax match pylonFMString /"[^"\\]*"/ contained

" ---- Body tokens ---------------------------------------------------------
" These are top-level matches (not contained in any region). The frontmatter
" region above has `keepend` and explicit `contains=`, so none of them leak
" into frontmatter highlighting.

" Brackets around bordered `[...]` and borderless `(...)` nodes.
syntax match pylonBracket /[\[\]()]/

" Alignment marker: a `-` flush with an opening or closing bracket, with
" at least one whitespace between the dash and the label text. Mirrors
" ALIGN_LEFT_RE (/^-\s+/) and ALIGN_RIGHT_RE (/\s+-$/) applied to the
" inside of a bracketed node.
"
" We use `\@<=` lookbehind here (rather than the `\zs` idiom preferred
" elsewhere) because `pylonBracket` already claims `[` and `(` at the
" same column -- vim's syntax engine advances past a matched bracket
" before retrying at the next column, so a `\zs`-anchored pattern that
" begins by consuming the bracket never fires. Lookbehind sidesteps
" this by starting the match AT the dash.
syntax match pylonAlignMarker /[\[(]\@<=-\ze\s/
syntax match pylonAlignMarker /\s\@<=-\ze[\])]/

" Edges: `<?-+>?` with at least one arrow head. Two alternatives cover
" the "starts with <" and "ends with >" cases; pure dashes (no arrow) stay
" as literal text, matching EDGE_RE's arrow-head gate in the JS parser.
syntax match pylonEdge /<-\+>\?/
syntax match pylonEdge /-\+>/

" `::` name declaration at end of a bracketed body: `[- label -] :: name`.
" NAME_RE = /::\s*([A-Za-z_]\w*)\s*$/ in the JS parser. Split into two
" groups so the separator and the name can be colored distinctly. The
" separator's lookahead (`\s\+IDENT\s*$`) gates it to declaration position
" only; the name is `contained` so it only fires after the separator.
syntax match pylonNameSep /::\ze\s\+[A-Za-z_]\w*\%(\s\+-\)\?\s*\%([])]\|$\)/
      \ nextgroup=pylonNameDeclare skipwhite
syntax match pylonNameDeclare /[A-Za-z_][A-Za-z0-9_]*\>/ contained

" Trailing `| renderer` pipe inside a node body. Supports chained renderer
" names (`| bar | hbar`) though only the first is used; the JS parser
" matches the entire trailing chain. The trailing anchor allows optional
" whitespace, an optional right-alignment dash, and the closing `]` / `)`
" so the match fires even inside a bracketed node where `$` would be
" past the closing bracket.
syntax match pylonRendererPipe /\s\+|\s\+[A-Za-z_][A-Za-z0-9_]*\%(\s\+|\s\+[A-Za-z_][A-Za-z0-9_]*\)*\ze\s*-\?\s*[\])]\?\s*$/

" `&ref` -- node reference. Boundary: preceded by `[`, `(`, whitespace, or
" start-of-line; the sigil itself anchors the start of the match with `\zs`.
" Use `\%(...\)` so the surrounding character doesn't get absorbed into
" the highlight group.
syntax match pylonNodeRef /\%(^\|[[(\t ]\)\zs&[A-Za-z_][A-Za-z0-9_]*\>/

" `@ref` -- data reference. Same boundary as `&ref` on the prefix side;
" additionally the suffix must be whitespace, `|`, `]`, `)`, or EoL. This
" keeps `user@example.com` literal because `e` is not a boundary char.
" `\zs` resets the match start to `@` so the preceding char is excluded
" from the highlight, exactly as the PLAN.md risks section prescribes.
syntax match pylonDataRef /\%(^\|[[(\t ]\)\zs@[A-Za-z_][A-Za-z0-9_]*\ze\%($\|[\t |\])]\)/

" ---- Highlight links -----------------------------------------------------
hi def link pylonFrontmatter      Normal
hi def link pylonFrontmatterFence Delimiter
hi def link pylonFMKey            Identifier
hi def link pylonFMNumber         Number
hi def link pylonFMString         String
hi def link pylonFMListDash       Delimiter
hi def link pylonBracket          Delimiter
hi def link pylonEdge             Operator
hi def link pylonAlignMarker      Operator
hi def link pylonNameSep          Operator
hi def link pylonNameDeclare      Define
hi def link pylonNodeRef          Identifier
hi def link pylonDataRef          Constant
hi def link pylonRendererPipe     PreProc

let b:current_syntax = "pylon"
