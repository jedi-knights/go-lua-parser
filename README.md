# go-lua-parser

[![CI](https://github.com/jedi-knights/go-lua-parser/actions/workflows/ci.yml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/ci.yml)
[![Release](https://github.com/jedi-knights/go-lua-parser/actions/workflows/release.yml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/release.yml)
[![Badge](https://github.com/jedi-knights/go-lua-parser/actions/workflows/badge.yaml/badge.svg)](https://github.com/jedi-knights/go-lua-parser/actions/workflows/badge.yaml)
[![Coverage](https://img.shields.io/badge/Coverage-0%25-lightgrey)](https://jedi-knights.github.io/go-lua-parser/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Lua 5.1 lexer, parser, and AST for Go, with LuaJIT extensions accepted by
default (`goto`/`::label::`, the `\z` string escape, and `//` integer
division).

Written to support branch-coverage instrumentation in
[neospec](https://github.com/jedi-knights/neospec), but published standalone so
other Go tooling that reasons about Lua source can consume it directly.

## Status

Pre-1.0. The public API is subject to change until `v1.0.0`.

- **Lexer:** complete for Lua 5.1 + the LuaJIT superset, including long
  strings and long comments with level markers (`[==[ ... ]==]`).
- **AST:** node types defined for the full grammar.
- **Parser:** under construction. The current release accepts empty programs
  and comment-only files; statement and expression parsing lands in
  follow-up releases.
- **Visitor:** `Walk` provides depth-first traversal over any `Node`.

## Install

```
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

## Scope

- **In scope:** Lua 5.1 source as consumed by LuaJIT and Neovim plugin
  authors. Positional tokens and AST nodes for tooling that needs to reason
  about source structure (coverage, linting, static analysis, refactoring).
- **Out of scope:** Bytecode compilation, evaluation, or the standard
  library. Use [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua) if you
  need to run Lua from Go.

## License

MIT
