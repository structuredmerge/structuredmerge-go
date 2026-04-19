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
