package lua_test

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/jedi-knights/go-lua-parser"
)

// TestParseCorpus parses every .lua file under testdata/ and fails on any
// syntax error. This is the acceptance test that says the parser can
// handle real Lua — see testdata/README.md for what belongs here.
func TestParseCorpus(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	parsed := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lua" {
			continue
		}
		path := filepath.Join("testdata", entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			chunk, perr := lua.Parse(path, src)
			if perr != nil {
				t.Fatalf("parse errors in %s:\n%v", path, perr)
			}
			if chunk == nil || chunk.Block == nil {
				t.Fatalf("nil chunk/block for %s", path)
			}
			if len(chunk.Block.Statements) == 0 && chunk.Block.Return == nil {
				t.Errorf("%s: parsed to empty block — likely all input was skipped", path)
			}
		})
		parsed++
	}
	if parsed == 0 {
		t.Fatal("no .lua files found in testdata/ — the corpus is empty")
	}
}
