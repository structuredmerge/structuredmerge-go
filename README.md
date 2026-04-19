# Structured Merge Go

Monorepo for the Go implementation of the Structured Merge library family.

Initial packages:

- `treehaver`
- `astmerge`
- `textmerge`
- `jsonmerge`

## Development

Standard repo tasks are exposed through `mise` and native Go tooling:

- `mise run format`
- `mise run format-check`
- `mise run lint`
- `mise run typecheck`
- `mise run test`
- `mise run check`

The Go monorepo uses:

- `gofmt` for formatting
- `golangci-lint` for linting
- `go test ./... -run '^$'` for type-checking/compilation
- `go test ./...` for unit and integration tests

The current tree-sitter backend path uses the sibling
`../tree-sitter-language-pack` checkout through a local `replace` in
`go.mod`. Repo tasks build its `ts-pack-ffi` crate first and compile with the
`tspack_dev` build tag so the Go binding can link against the local fork while
the upstream packaging fix is pending.

Integration tests consume the shared fixture corpus from the sibling
`../fixtures` repository instead of copying fixture data into this monorepo.
