# StructuredMerge Go

StructuredMerge Go provides Go packages for building merge-aware tools that need
portable structured-merge contracts without leaving the Go runtime.

The module includes the core AST/review contracts, parser substrate support,
format-specific merge libraries, binary/ZIP planning helpers, provider adapters,
and a Go packaging recipe library.

Project links:

- Website: <https://structuredmerge.org>
- Implementations: <https://structuredmerge.org/implementations.html>
- Specification: <https://github.com/structuredmerge/structuredmerge-spec>
- Shared fixtures: <https://github.com/structuredmerge/structuredmerge-fixtures>

## Install

```sh
go get github.com/structuredmerge/structuredmerge-go
```

Import only the packages your tool needs:

```go
import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/jsonmerge"
)
```

## Command

The Go implementation ships the implementation-specific `smorg-go` command. Use
that name in git configuration unless a package manager or local install has
provided a `smorg` symlink.

Package-manager formulas may expose the selected implementation as `smorg`.
For a local user-created symlink:

```sh
ln -s "$(command -v smorg-go)" ~/.local/bin/smorg
```

```sh
git config merge.smorg-go.driver 'smorg-go merge-driver %O %A %B %P'
git config diff.smorg-go.command 'smorg-go diff-driver'
smorg-go conflicts diff path/to/file-with-conflicts.go
smorg-go languages --gitattributes
```

`merge-driver` updates Git's `%A` file by default, or writes to `--output` when
used outside git. `diff-driver` accepts both the two-argument local form and the
seven- or nine-argument forms Git passes to external diff commands.
`conflicts diff` reports conflict-marker regions in a file that already contains
Git conflict markers.

Current semantic merge-driver coverage is fixture-backed for JSON and for the
first Go source-language slice. Other language and format paths should be treated
as git-compatible command surfaces until their `ast-merge-git` coverage is
promoted.

## Packages

Core:

- [`treehaver`](https://github.com/structuredmerge/structuredmerge-go/tree/main/treehaver) - parser substrate, byte ranges, backend adapters, and binary tree contracts.
- [`astmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/astmerge) - AST merge contracts, diagnostics, planning, review, replay, and nested-merge vocabulary.
- [`asttemplate`](https://github.com/structuredmerge/structuredmerge-go/tree/main/asttemplate) - template/session transport contracts.

Format libraries:

- [`plainmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/plainmerge)
- [`jsonmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/jsonmerge)
- [`yamlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/yamlmerge)
- [`tomlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/tomlmerge)
- [`markdownmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/markdownmerge)
- [`rubymerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/rubymerge)
- [`gomerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/gomerge)
- [`rustmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/rustmerge)
- [`typescriptmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/typescriptmerge)
- [`binarymerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/binarymerge)
- [`zipmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/zipmerge)

Provider and recipe libraries:

- [`goccygoyamlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goccygoyamlmerge)
- [`pigeontomlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/pigeontomlmerge)
- [`goldmarkmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goldmarkmerge)
- [`goparsermerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goparsermerge)
- [`kettlegomodder`](https://github.com/structuredmerge/structuredmerge-go/tree/main/kettlegomodder)

## Portability

The Go packages are developed against the shared StructuredMerge fixtures. Those
fixtures define the cross-language behavior expected from the Go, TypeScript,
Rust, and Ruby implementations. Conformance checks live in package tests and in
the shared spec/fixture tooling rather than in a static launch-status document.

## Development

Common checks:

- `mise run check`
- `go test ./...`

The current tree-sitter backend path uses the sibling
`../tree-sitter-language-pack` checkout through a local `replace` in `go.mod`.
Repo tasks build its `ts-pack-core-ffi` crate first and compile with the
`tspack_dev` build tag while the upstream packaging fix is pending.
