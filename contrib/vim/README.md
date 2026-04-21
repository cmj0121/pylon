# Pylon Vim plugin

Syntax highlighting for [Pylon](https://github.com/cmj0121/pylon) diagram
source files. Recognised extension: `*.pylon`.

The plugin lives under `contrib/vim/` rather than at the repo root so the
top level of `cmj0121/pylon` stays focused on its JS library. That means
plugin managers need to be told where the runtimepath root actually is.

## Install

### vim-plug

```vim
Plug 'cmj0121/pylon', { 'rtp': 'contrib/vim' }
```

`rtp` points vim-plug at the subdirectory that contains `syntax/` and
`ftdetect/`, not the repo root.

### lazy.nvim

lazy.nvim has no first-class "subdirectory" key. Use `init` to prepend the
plugin's own install path with the `contrib/vim` suffix onto
`runtimepath`:

```lua
{
  'cmj0121/pylon',
  init = function()
    local root = vim.fn.stdpath('data') .. '/lazy/pylon/contrib/vim'
    vim.opt.runtimepath:prepend(root)
  end,
  ft = 'pylon',
}
```

(If you pin lazy's plugin dir elsewhere, adjust the first argument to
`runtimepath:prepend`.)

### Manual / native packages

Clone the repo and copy `contrib/vim/*` into a native-package path:

```sh
git clone https://github.com/cmj0121/pylon.git /tmp/pylon
mkdir -p ~/.vim/pack/pylon/start/pylon
cp -R /tmp/pylon/contrib/vim/* ~/.vim/pack/pylon/start/pylon/
```

Then restart Vim. Neovim users substitute `~/.config/nvim/pack/...`.

### Make target

If you already have the repo checked out, the bundled `Makefile` wraps the
copy/remove steps:

```sh
make -C contrib/vim install              # installs to ~/.config/nvim/pack/pylon/start/pylon
make -C contrib/vim install VIM_PACK=$HOME/.vim/pack/pylon/start/pylon
make -C contrib/vim install VIM_PACK=$HOME/.local/share/nvim/site/pack/pylon/start/pylon
make -C contrib/vim uninstall
```

`VIM_PACK` defaults to Neovim's config-home native package path (under
`$HOME/.config/nvim/`, which Neovim includes on `runtimepath` by default);
override it for Vim 8+'s `~/.vim/pack/...` layout, the XDG data-home
`~/.local/share/nvim/site/pack/...` location, or any other runtimepath
root.

## What gets highlighted

| Group               | Matches                                                     |
| ------------------- | ----------------------------------------------------------- |
| `pylonFrontmatter`  | YAML-subset block between the two `---` fences at file head |
| `pylonFMKey`        | identifier keys inside frontmatter (`size:`, `data:`, ...)  |
| `pylonFMNumber`     | numeric scalars inside frontmatter                          |
| `pylonFMString`     | double-quoted strings inside frontmatter                    |
| `pylonBracket`      | `[`, `]`, `(`, `)` around nodes                             |
| `pylonEdge`         | `->`, `<-`, `<->`, `-->`, `<-->`, ... arrow tokens          |
| `pylonAlignMarker`  | leading / trailing `-` flush with a bracket                 |
| `pylonNameDeclare`  | `:: ident` trailing name declaration                        |
| `pylonNodeRef`      | `&ident` node back-references                               |
| `pylonDataRef`      | `@ident` data references (boundary-checked)                 |
| `pylonRendererPipe` | trailing `\| renderer` on a node body                       |

See `sample.pylon` for a file that exercises every group.

## Parser drift warning

The regexes in `syntax/pylon.vim` duplicate grammar rules from
`src/js/pylon.js`. They are a best-effort visual approximation, not a
second parser. If the two diverge, the JS side is canonical -- open an
issue or a PR and we'll resync.

The `@ref` boundary in particular is implemented in JS as a
prev-char / next-char predicate
(`isDataRefBoundary`); the Vim equivalent uses `\%(^\|[[(\t ]\)\zs@\k\+\ze\%($\|[\t |\])]\)`
which should be exactly equivalent but has not been exhaustively
fuzz-tested.
