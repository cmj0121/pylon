-- contrib/vim/lua/pylon/lsp.lua — Neovim LSP client setup for pylon-lsp.
--
-- Two integration paths are supported out of the box. Both require
-- the `pylon-lsp` binary on $PATH; install it with
--
--   go install github.com/cmj0121/pylon/src/go/cmd/pylon-lsp@latest
--
-- (1) nvim-lspconfig users can register the server with one call:
--
--   require('pylon.lsp').setup()
--
-- (2) Plain Neovim without nvim-lspconfig — drive vim.lsp.start
--     directly on a FileType autocmd:
--
--   vim.api.nvim_create_autocmd('FileType', {
--     pattern = 'pylon',
--     callback = function()
--       vim.lsp.start(require('pylon.lsp').start_opts())
--     end,
--   })
--
-- Today the server provides diagnostics (every SPEC error class),
-- document symbols (`:: name` declarations and `@series` keys), and
-- partial semantic tokens (brackets, `&ref`, `@ref`). Edges, align
-- markers, `:: name`, frontmatter keys, and `| renderer` pipes fall
-- back to the bundled regex syntax plugin until feat/pylon-lsp-ux
-- ports them to the server.

local M = {}

-- root_dir returns the buffer's enclosing directory. Pylon sources
-- are self-contained — there's no project manifest to anchor on —
-- so "the directory of the file" is the safe default. Callers
-- passing this to lspconfig may override for multi-file projects.
local function root_dir(fname)
  if fname == nil or fname == '' then
    return vim.loop.cwd()
  end
  return vim.fs.dirname(fname)
end

-- start_opts returns the table accepted by vim.lsp.start(). Call
-- from a FileType autocmd — the caller's buffer is what the server
-- ends up attached to.
function M.start_opts()
  local bufname = vim.api.nvim_buf_get_name(0)
  return {
    name = 'pylon-lsp',
    cmd = { 'pylon-lsp' },
    filetypes = { 'pylon' },
    root_dir = root_dir(bufname),
  }
end

-- config is the nvim-lspconfig-shaped server definition. Exposed for
-- users who want to plug it into `require('lspconfig.configs')` by
-- hand rather than via M.setup().
M.config = {
  default_config = {
    cmd = { 'pylon-lsp' },
    filetypes = { 'pylon' },
    root_dir = root_dir,
    single_file_support = true,
  },
  docs = {
    description = 'Language Server for Pylon diagram sources.',
  },
}

-- setup registers the pylon server with nvim-lspconfig and calls its
-- setup() with empty overrides. Emits a warning and returns without
-- error when nvim-lspconfig is not available, so the same Lua init
-- snippet works before and after lspconfig is installed.
function M.setup()
  local ok, lspconfig_configs = pcall(require, 'lspconfig.configs')
  if not ok then
    vim.notify(
      'pylon-lsp: nvim-lspconfig not installed; use vim.lsp.start with pylon.lsp.start_opts() instead',
      vim.log.levels.WARN
    )
    return
  end
  if not lspconfig_configs.pylon then
    lspconfig_configs.pylon = M.config
  end
  require('lspconfig').pylon.setup({})
end

return M
