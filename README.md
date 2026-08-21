# go-lua-parser

[![CI](https://github.com/jedi-knights/go-lua-parser/actions/workflows/ci.yml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/ci.yml)
[![Release](https://github.com/jedi-knights/go-lua-parser/actions/workflows/release.yml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/release.yml)
[![Badge](https://github.com/jedi-knights/go-lua-parser/actions/workflows/badge.yaml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/badge.yaml)
[![Coverage](https://img.shields.io/badge/Coverage-89%2E3%25-green)](https://jedi-knights.github.io/go-lua-parser/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Lua 5.1 + LuaJIT lexer, parser, and AST for Go tools that reason about Lua source.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Examples](#examples)
- [Configuration](#configuration)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Overview

When you need to *run* Lua from Go you reach for [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua) — it's the obvious pick and it works well. But `gopher-lua` compiles source to bytecode on the way in, and once it's bytecode the source structure is gone: no positions, no comment ranges, no way to tell an `if` from a `while` at the AST level.

If your tool needs *source structure* — a coverage instrumenter marking branch points, a linter walking call expressions, a refactor bot rewriting `require` calls — you're back to writing a lexer and parser. That's not a small job for Lua: long strings with level markers (`[==[ ... ]==]`) mean the lexer has to count `=` before it knows where a string ends; call statements and prefix expressions share syntax; and LuaJIT's superset (`goto`/`::label::`, `\z`, `//`) is what plugin authors actually write, so refusing to accept it produces false positives on real files.

This library exists so you don't have to solve any of that. `Parse(filename, src)` returns a `*Chunk` with positions preserved on every node; `Walk(visitor, chunk)` traverses it depth-first; and the parser continues past syntax errors so a single bad line doesn't blank out coverage for the rest of the file.

Pre-1.0. The public API is subject to change until `v1.0.0`.

## Features

- **LuaJIT superset accepted by default** — `goto`/`::label::`, the `\z` string escape, and `//` integer division. Turning them off is not a supported mode; downstreams that need strict-5.1 rejection layer that check on top. Hex-float literals (`0x1p10`) — another LuaJIT extension — are **not** yet supported; the lexer treats them as a hex integer followed by an identifier.
- **Position-preserving tokens and AST nodes** — every `Token` and every `Node` carries a `Position` (file, line, column, byte offset) suitable for diagnostics and instrumentation output (LCOV, SARIF, JUnit).
- **Depth-first visitor** — `Walk` traverses any `Node`; return `nil` from `Visit` to prune a subtree, return a different `Visitor` to swap traversal state (useful for scope tracking).
- **Panic-mode error recovery** — the parser advances past syntax errors to the next statement boundary, so a `SyntaxErrors` from `Parse` describes independent problems rather than one cascading failure.
- **Corpus-tested** — `parser_corpus_test.go` parses real Lua source from neospec's embedded runtime harness, so grammar regressions surface against realistic input, not just hand-written fixtures.

## Requirements

- Go 1.27 or later (see [`go.mod`](go.mod))

## Installation

```bash
go get github.com/jedi-knights/go-lua-parser
```

## Usage

```go
package main

import (
    "fmt"

    lua "github.com/jedi-knights/go-lua-parser"
)

func main() {
    src := []byte(`local x = "hello"`)
    l := lua.NewLexer("example.lua", src)
    for {
        tok := l.Next()
        if tok.Kind == lua.TokenEOF {
            break
        }
        fmt.Printf("%s %q at %s\n", tok.Kind, tok.Value, tok.Pos)
    }
}
```

Output:

```
local "local" at example.lua:1:1
IDENT "x" at example.lua:1:7
= "=" at example.lua:1:9
STRING "hello" at example.lua:1:11
```

## Examples

Three worked examples, progressively deeper. Each was executed against the library before this README was updated; output blocks show the actual program output.

### Parse and inspect top-level statements

`Parse` returns a `*Chunk` regardless of syntax errors — the returned error, if any, is a `SyntaxErrors` describing every issue found. That means you can iterate the AST even on partial input (useful for editor tooling).

```go
src := []byte(`
local x = 1
local y = 2
print(x + y)
`)
chunk, err := lua.Parse("sample.lua", src)
if err != nil {
    fmt.Println("parse error:", err)
    return
}
for _, stmt := range chunk.Block.Statements {
    fmt.Printf("%T at %s\n", stmt, stmt.Pos())
}
```

Output:

```
*lua.LocalAssignStat at sample.lua:2:1
*lua.LocalAssignStat at sample.lua:3:1
*lua.CallStat at sample.lua:4:1
```

### Walk the AST counting call expressions

`Walk` is depth-first. Return the visitor from `Visit` to descend into a node's children; return `nil` to skip the subtree entirely (useful for skipping nested function bodies, for instance). Direct calls (`f(x)`) are `*CallExpr`; method calls (`x:f(y)`) are a separate `*MethodCallExpr` — extend the type switch if you need both.

```go
type callCounter struct{ n int }

func (c *callCounter) Visit(node lua.Node) lua.Visitor {
    if _, ok := node.(*lua.CallExpr); ok {
        c.n++
    }
    return c
}

src := []byte(`
print(1)
io.write("hi")
math.floor(2.5)
`)
chunk, _ := lua.Parse("count.lua", src)

c := &callCounter{}
lua.Walk(c, chunk)
fmt.Printf("call expressions: %d\n", c.n)
```

Output:

```
call expressions: 3
```

### Parse-with-errors continues past a bad statement

`sync()` skips to the next statement boundary after each error, so a file with one broken line still yields diagnostics for later errors and a `*Chunk` containing everything that parsed cleanly. That's the property a coverage tool relies on: instrument the good statements, surface the bad line to the user, don't blank the whole file.

```go
src := []byte(`
local x = 1
local = "oops"      -- missing name
local y = 2
`)
chunk, err := lua.Parse("bad.lua", src)
if err != nil {
    // err is a SyntaxErrors — a []*SyntaxError with Pos + Message.
    fmt.Println("errors:", err)
}
fmt.Printf("statements parsed anyway: %d\n", len(chunk.Block.Statements))
```

## Configuration

None. The library is a pure Go dependency; there are no environment variables, config files, or build tags to set.

## Development

Local commands mirror CI exactly — `make test-ci` runs the same `go test -race -json` CI runs, and `make coverage-check` gates on the same LCOV line-coverage number.

```bash
make lint            # golangci-lint
make test            # go test -race with coverage.out
make test-ci         # -json variant for CI (produces test-results.json)
make test-coverage   # LCOV line-coverage report
make test-html       # HTML report under htmlcov/ (same generator as the deployed Pages site)
make coverage-check COVERAGE_THRESHOLD=84   # simulate the CI gate locally
make help            # list all targets
```

The `test-coverage`, `test-html`, and `coverage-check` targets need `gcov2lcov` and `lcov` locally:

```bash
brew install lcov
go install github.com/jandelgado/gcov2lcov@latest
```

A pre-push hook that runs `golangci-lint` before every push lives in `.githooks/pre-push`. Enable it once with:

```bash
git config core.hooksPath .githooks
```

### CI pipeline

Three workflows, chained via `workflow_run`:

1. **CI** — lint + test + LCOV coverage gate + test-results publishing. Runs on every PR and push to main.
2. **Release** — `go-semantic-release` computes the next semver tag from conventional commits (`feat`/`fix`/breaking) and pushes CHANGELOG/VERSION back to main. The tag is the release artifact; downstreams resolve via `go get @vX.Y.Z`.
3. **Badge** — regenerates LCOV, deploys the `genhtml` HTML report to GitHub Pages, and updates the coverage badge above with a tiered color.

## Contributing

Issues and pull requests welcome at <https://github.com/jedi-knights/go-lua-parser>. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide — commit format, PR expectations, testing bar, code style.

Quick summary:

- [Conventional Commits](https://www.conventionalcommits.org/) — `feat` and `fix` drive semver via `semantic-release`; other types are no-ops for versioning.
- Coverage floor is 84% (LCOV line coverage). New code should ship tests; CI blocks a regression.
- Cyclomatic complexity is capped at 14 per function (see `.golangci.yml`).
- Main is ruleset-protected; direct pushes are blocked. Open a PR — 0 approvals required, but the PR flow is mandatory.

## License

MIT — see [`LICENSE`](LICENSE).
