// Package lua provides a lexer, parser, and abstract syntax tree for the
// Lua 5.1 language, with the LuaJIT superset accepted by default:
// `goto`/`::label::`, the `\z` string escape, and `//` integer division.
//
// The package was originally written to support branch-coverage
// instrumentation in neospec (github.com/jedi-knights/neospec), but is
// published standalone. It is intentionally scoped to source analysis and
// does not include a Lua evaluator; use github.com/yuin/gopher-lua if you
// need to run Lua from Go.
//
// # Status
//
// Pre-1.0. The lexer is complete; the parser is under construction. Public
// API is subject to change until v1.
package lua
