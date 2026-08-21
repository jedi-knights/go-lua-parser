# Contributing to go-lua-parser

Thanks for considering a contribution — bug reports, docs fixes, and PRs are all welcome.

This guide covers the *process*. For local dev setup (Makefile targets, coverage tooling, pre-push hook), see [README.md § Development](README.md#development).

## Table of Contents

- [Ways to contribute](#ways-to-contribute)
- [Reporting bugs](#reporting-bugs)
- [Proposing features](#proposing-features)
- [Development workflow](#development-workflow)
- [Commit messages](#commit-messages)
- [Pull requests](#pull-requests)
- [Code style](#code-style)
- [Testing](#testing)
- [License](#license)

## Ways to contribute

- **Bug reports** — open an issue with a minimal reproducer (source snippet + expected/actual output).
- **Docs** — fixes to godoc, README, or CLAUDE.md are shipped as regular PRs.
- **Grammar / parser fixes** — if this library rejects Lua that LuaJIT accepts, that is a bug; if it accepts something LuaJIT rejects, that is also a bug. Include the offending source in the issue or PR.
- **New AST helpers** — small pure functions on existing node types (e.g., `NumberExpr.Float()`) are usually straightforward to review. Bigger structural changes need a discussion issue first.

## Reporting bugs

Open an issue at <https://github.com/jedi-knights/go-lua-parser/issues> with:

1. **What you did** — the Go call site or the Lua source you fed to `Parse`/`NewLexer`.
2. **What you expected.**
3. **What actually happened** — the returned error, the AST shape, or the diff between LuaJIT and this parser.
4. **Version** — the `v0.x.y` you resolved via `go get`, or the commit SHA if you're on `main`.

Please avoid: "it doesn't work" without a source snippet. Lua's grammar has enough edge cases that we can't guess the input from the symptom.

## Proposing features

For anything that isn't a bug fix or small helper, open an issue first so we can align on scope before you write code. In particular, please read [CLAUDE.md § What NOT to build](CLAUDE.md#what-not-to-build) — the project has explicit out-of-scope areas (Lua evaluation, standard library, lint rules on top of the AST, CLI). PRs in those areas will be closed without merge.

## Development workflow

1. Fork the repo (or ask for push access if you're a regular contributor).
2. Create a branch off the latest `main`:
   ```bash
   git checkout main && git pull
   git checkout -b <type>/<short-desc>   # e.g. fix/parser-string-escape
   ```
3. Make the change. Add tests. Keep changes small and focused.
4. Run the local gates:
   ```bash
   make lint            # go vet + golangci-lint
   make test-ci         # same test invocation CI runs
   make coverage-check COVERAGE_THRESHOLD=84
   ```
5. Commit using [Conventional Commits](#commit-messages).
6. Push and open a PR against `main`.

### Local prerequisites

Everything in [README.md § Development](README.md#development). Short version: Go 1.27+, plus `lcov` and `gcov2lcov` if you want to run the coverage targets locally.

Enable the pre-push hook once so lint failures surface locally, not in CI:

```bash
git config core.hooksPath .githooks
```

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/) — `semantic-release` reads them to compute the next version tag on every merge to `main`.

- `feat: ...` — new user-visible functionality → **minor** bump.
- `fix: ...` — bug fix → **patch** bump.
- `feat!:` / `fix!:` / any type with `BREAKING CHANGE:` footer → **major** bump.
- `docs:`, `test:`, `ci:`, `build:`, `chore:`, `refactor:`, `perf:`, `style:` — do not bump version.

Format:

```
<type>(<scope>): <lowercase, imperative, no period>

Optional body explaining *why*, wrapped at ~72 chars.

BREAKING CHANGE: <what changed and how to adapt>  # only for majors
```

Examples from this repo's history:

- `feat(parser): panic-mode error recovery via sync()`
- `test(parser): add real-world Lua corpus test`
- `ci: gate on LCOV line coverage instead of Go statement coverage`

## Pull requests

- **One PR = one concern.** If your description contains "and", split it.
- **Branch off `main`** (or rebase before opening the PR if `main` moved).
- **CI must be green** — Lint and Test both pass, coverage stays ≥ 84%.
- **New code ships tests** — see [Testing](#testing) below. Prefer black-box tests through `Parse`/`NewLexer`/`Walk` over reaching into unexported internals.
- **PR description** — say *why*, not just *what*. The diff shows what changed; the description should explain the motivation, the alternatives considered, and any user-visible behavior change.

`main` is protected by a ruleset requiring the PR flow (0 approvals required, but direct push is blocked). Once CI is green, squash-merge is the default.

## Code style

- Follow Go conventions — [Effective Go](https://go.dev/doc/effective_go) plus the checks in `.golangci.yml`.
- **Cyclomatic complexity ≤ 14** per function (enforced by `gocyclo`). Split large functions instead of raising the cap.
- **Exported symbols carry godoc** starting with the symbol name (enforced by `revive`).
- **American English** in comments and docs (`marshaling`, not `marshalling`).
- Don't add abstractions for hypothetical future flexibility — this library values direct, boring code over cleverness.

## Testing

- **Table-driven tests** for anything with more than two inputs. See `lexer_test.go` and `parser_test.go` for the pattern used in this repo.
- **Real-world coverage** — if you fix a bug that reproduces from a plugin author's file, add that file (or a minimal reduction) to `testdata/` and cover it in `parser_corpus_test.go`. That's where regressions on realistic input surface fastest.
- **Line coverage floor is 84%**. `make coverage-check COVERAGE_THRESHOLD=84` locally reproduces the CI gate; if your PR would drop below, CI will block it.
- Run `make test-html` to open a local HTML coverage report and find uncovered lines.

## License

By contributing you agree that your contribution is licensed under the [MIT License](LICENSE) that covers the rest of the project.
