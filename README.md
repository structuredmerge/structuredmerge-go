# StructuredMerge Go

Go implementation of the StructuredMerge contract.

This repository is one of four peer launch implementations: [Go](https://github.com/structuredmerge/structuredmerge-go), [TypeScript](https://github.com/structuredmerge/structuredmerge-typescript), [Rust](https://github.com/structuredmerge/structuredmerge-rust), and [Ruby](https://github.com/structuredmerge/structuredmerge-ruby). The language repos are not separate products. They consume the same public spec and shared fixture corpus so tools can choose the runtime surface that fits their environment.

Project links:

- Website: <https://structuredmerge.org>
- Implementations overview: <https://structuredmerge.org/implementations.html>
- Conformance model: <https://structuredmerge.org/conformance.html>
- Specification: <https://github.com/structuredmerge/structuredmerge-spec>
- Shared fixtures: <https://github.com/structuredmerge/structuredmerge-fixtures>

## Workspace

This is a Go module for StructuredMerge packages.

Package directories:

- [`astmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/astmerge)
- [`asttemplate`](https://github.com/structuredmerge/structuredmerge-go/tree/main/asttemplate)
- [`binarymerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/binarymerge)
- [`goccygoyamlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goccygoyamlmerge)
- [`goldmarkmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goldmarkmerge)
- [`gomerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/gomerge)
- [`goparsermerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/goparsermerge)
- [`jsonmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/jsonmerge)
- [`kettlegomodder`](https://github.com/structuredmerge/structuredmerge-go/tree/main/kettlegomodder)
- [`markdownmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/markdownmerge)
- [`pigeontomlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/pigeontomlmerge)
- [`rubymerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/rubymerge)
- [`rustmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/rustmerge)
- [`textmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/textmerge)
- [`tomlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/tomlmerge)
- [`treehaver`](https://github.com/structuredmerge/structuredmerge-go/tree/main/treehaver)
- [`typescriptmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/typescriptmerge)
- [`yamlmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/yamlmerge)
- [`zipmerge`](https://github.com/structuredmerge/structuredmerge-go/tree/main/zipmerge)

## Conformance

Integration tests should consume the shared fixture corpus from the sibling `../structuredmerge-fixtures` checkout. A ruleset, fixture, diagnostic shape, or review outcome should mean the same thing whether exercised through Go, TypeScript, Rust, or Ruby.

Use the spec repository's conformance matrix for the current launch-readiness snapshot:

- <https://github.com/structuredmerge/structuredmerge-spec/blob/main/conformance-matrix.md>
- <https://github.com/structuredmerge/structuredmerge-spec/blob/main/IMPLEMENTATION_STATUS.md>

## Development

Standard repo tasks are exposed through `mise` and native Go tooling.

Common checks:

- `mise run check`
- `go test ./...`

The current tree-sitter backend path uses the sibling `../tree-sitter-language-pack` checkout through a local `replace` in `go.mod`. Repo tasks build its `ts-pack-ffi` crate first and compile with the `tspack_dev` build tag while the upstream packaging fix is pending.

## Status

Early implementation work. Public compatibility claims should be tied to shared fixtures and documented conformance status rather than runtime-specific assumptions.
