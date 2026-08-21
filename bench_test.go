package lua_test

import (
	"testing"

	lua "github.com/jedi-knights/go-lua-parser"
)

// benchCorpus is a representative sample of Lua source that exercises
// long strings, table constructors, method calls, control flow, and
// operator precedence — the same shapes the corpus test parses. Keeping
// the input in-file avoids testdata I/O in the benchmark hot path so
// numbers reflect lexer/parser work, not disk.
var benchCorpus = []byte(`
local M = {}

function M.new(opts)
    opts = opts or {}
    return setmetatable({
        name = opts.name or "anon",
        count = opts.count or 0,
        tags = opts.tags or {},
    }, { __index = M })
end

function M:add(delta)
    self.count = self.count + (delta or 1)
    return self
end

function M:describe()
    local parts = {}
    for i, t in ipairs(self.tags) do
        parts[#parts + 1] = string.format("[%d]%s", i, t)
    end
    return self.name .. " x" .. self.count .. " " .. table.concat(parts, ",")
end

local function iter(t)
    local i = 0
    return function()
        i = i + 1
        if i <= #t then return t[i] end
    end
end

local msg = [==[
long-string with level markers and
multiple lines to exercise the lexer
]==]

for x in iter({"a", "b", "c"}) do
    if x == "b" then
        goto skip
    end
    print(x, msg)
    ::skip::
end

return M
`)

func BenchmarkLexer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		l := lua.NewLexer("bench.lua", benchCorpus)
		for {
			tok := l.Next()
			if tok.Kind == lua.TokenEOF {
				break
			}
		}
	}
	b.SetBytes(int64(len(benchCorpus)))
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := lua.Parse("bench.lua", benchCorpus); err != nil {
			b.Fatalf("unexpected parse error: %v", err)
		}
	}
	b.SetBytes(int64(len(benchCorpus)))
}
