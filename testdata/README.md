# Parser corpus

Realistic Lua source used to smoke-test the parser end-to-end. Anything the
parser rejects here is a bug — these files are hand-written idiomatic Lua
that runs under LuaJIT in production.

## Files

- `neospec_reporter.lua`, `neospec_harness.lua`, `neospec_coverage_hook.lua`
  — snapshots from [neospec](https://github.com/jedi-knights/neospec)'s
  embedded Lua harness. MIT-licensed by Jedi Knights, same license as this
  repo. Copied verbatim; not kept in sync — they exist as a stable corpus,
  not as a live dependency.

## Adding new fixtures

Drop a `.lua` file in this directory and `TestParseCorpus` will pick it up
automatically. Prefer real code (a plugin file, a config, a library) over
synthetic examples — real Lua has more surprising shapes than anything
handcrafted.
