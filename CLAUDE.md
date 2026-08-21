# go-lua-parser — Project Instructions

## What this is

A Lua 5.1 lexer, parser, and AST for Go tools. The LuaJIT superset is
accepted by default: `goto` / `::label::`, the `\z` string escape, and `//`
integer division. Written to support branch-coverage instrumentation in
[neospec](https://github.com/jedi-knights/neospec), published standalone so
other Go tooling that reasons about Lua source can consume it directly.

Package path: `github.com/jedi-knights/go-lua-parser` (package name `lua`,
flat layout — no `cmd/` or `internal/`; the module ships one library
package). Pre-1.0; public API is subject to change until `v1.0.0`.

## Design philosophy anchors

### 1. Accept what LuaJIT accepts

The parser accepts the Lua 5.1 grammar plus the three LuaJIT extensions
above. It does not offer a "strict 5.1 mode" flag and does not accept
Lua 5.2+ syntax beyond what LuaJIT already permits. The reason is
downstream: neospec instruments plugins that run on LuaJIT-embedded
Neovim, so accepting a superset LuaJIT rejects would produce false
positives at the coverage layer.

Semantic checks the reference Lua compiler enforces but the grammar does
not (e.g. `break` outside a loop, `return` not being the last statement in
its block) are **not** enforced here — those belong to callers that need
them (`gopher-lua` for evaluation, `selene` for linting).

### 2. Source analysis only

This is a source-structure library: tokens with positions, AST nodes,
depth-first visitor. It does not evaluate, compile to bytecode, or
implement the Lua standard library. If a downstream needs to run Lua,
they use [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua); this
library exists so tools that need to *reason about* Lua source do not
have to rebuild lexing/parsing from scratch.

### 3. Error recovery over failing fast

The parser uses panic-mode error recovery (`sync()` in `parser.go`) so a
single syntax error does not abort the whole parse. This matters because
the primary consumer (neospec) needs coverage data for the parseable
portion of a file even when a downstream edit broke a later line.

## What NOT to build

- **A Lua evaluator or bytecode compiler.** Out of scope — that is
  `gopher-lua`'s job.
- **Lua standard library shims.** Not a runtime.
- **Lua 5.2+-only syntax** (integer `//` is fine because LuaJIT accepts
  it; `goto` is fine because LuaJIT accepts it; bitwise operators from
  5.3 and `<const>` from 5.4 are not, unless LuaJIT ever ships them).
- **Style/lint rules on top of the AST.** That is `selene`'s and
  `stylua`'s territory. Keep this library at "here is the shape of the
  source"; opinions about the source belong upstream in the consumer.
- **A CLI.** Consumers embed this as a Go library. No `cmd/` directory.

## Stack

- Go 1.27
- Flat single-package layout at the module root (`package lua`).
- Files: `lexer.go`, `parser.go`, `parser_stat.go`, `parser_expr.go`,
  `ast.go`, `visitor.go`, `token.go`, `errors.go`, `doc.go`.
- Tests are table-driven; a real-world corpus lives in `testdata/`
  (currently the three embedded Lua files from neospec's runtime harness).
- Lint: `golangci-lint` v2 (config in `.golangci.yml`).

## Hard constraints

- **Line coverage ≥ 84%** (currently 84.0%). CI gates on the LCOV number
  from `lcov --summary`, so a regression fails the PR. Bump the threshold
  in `.github/workflows/ci.yml` when coverage ratchets up.
- **Cyclomatic complexity ≤ 14** per function. Enforced by
  `.golangci.yml` (`gocyclo.min-complexity: 15`).
- **New code ships tests.** The corpus test in `parser_corpus_test.go`
  guards against regressions on real-world plugin source; add cases
  there when a bug reproduces from a real file.
- **Public API is documented.** Every exported symbol has a godoc
  comment starting with the symbol name — `revive`'s `exported` rule
  enforces this.

## Local dev

- `make lint` / `make test` — mirror CI exactly (`-race`, LCOV threshold).
- `make test-html` — generate the HTML coverage report locally under
  `htmlcov/` using the same `genhtml` pipeline the Badge workflow deploys
  to Pages.
- `.githooks/pre-push` runs `golangci-lint` before push; enable with
  `git config core.hooksPath .githooks`.

## CI / release

Pipeline is aligned with [neospec](https://github.com/jedi-knights/neospec)
and modeled on [yoda.nvim](https://github.com/jedi-knights/yoda.nvim)'s
coverage reporting:

1. `ci.yml` — lint + test + LCOV coverage threshold + publish test
   results. Runs on every PR and every push to main.
2. `release.yml` — `jedi-knights/go-semantic-release@v0` computes the
   next semver tag from conventional commits and pushes CHANGELOG/VERSION
   back to main. The tag is the release artifact for a Go library;
   consumers resolve via `go get @vX.Y.Z`.
3. `badge.yaml` — regenerates LCOV, deploys the `genhtml` HTML report to
   GitHub Pages, and updates the coverage badge in `README.md` with a
   tiered color (brightgreen ≥90 / green ≥80 / yellow ≥70 / orange ≥60 /
   red). Serialized via `concurrency: pages`.

No GoReleaser / Docker / `action.yml` — this is a library, not a binary
or a composite action.

## Commit discipline

- Conventional Commits (see `~/.claude/CLAUDE.md`). `feat` / `fix` /
  breaking-change footers drive semver via semantic-release; other types
  no-op.
- One PR = one `type(scope)` pair.
- Main is protected by a ruleset requiring PR flow (0 approvals). Direct
  push is blocked — always branch → commit → push → PR → merge.
