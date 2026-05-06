# Go PLAN

## Objective

Build a Go module/package family for the merge stack with tree-sitter as the
primary analysis backend for MVP releases, emphasizing straightforward
deployment, readable APIs, and fixture-based conformance with the Ruby stack.

## License

Planned dual license for all new Go merge-stack modules:

- `AGPL-3.0-only`
- `PolyForm-Small-Business-1.0.0`

Reference:

- `LICENSE_TEMPLATE_PLAN.md`

## Scope Boundary

Initial focus:

1. tree-sitter adapter package
2. core merge model
3. plain text merge MVP
4. JSON and JSONC merge MVP
5. shared-fixture conformance runner

Deferred:

- `kettle-jem`-style scaffolding
- full merge-family parity beyond MVP
- non-tree-sitter native parser experiments
- PEG-backend parity beyond the initial TOML family slice

## Proposed Module Family

Initial package/module candidates:

- `tree-haver-go`
- `ast-merge-go`
- `plain-merge-go`
- `json-merge-go`

Possible later modules:

- `toml-merge-go`
- `yaml-merge-go`
- `markdown-merge-go`
- `merge-ruleset-go`
- `gomod-template-go`

## Ruby Mapping

Reference Ruby siblings to study first:

- `tree_haver`
- `ast-merge`
- `json-merge`

MVP parity target:

- parser acquisition and diagnostics from `tree_haver`
- merge contracts from `ast-merge`
- strict JSON/JSONC behavior from `json-merge`

## Tree-Sitter Strategy

Primary backend:

- Go tree-sitter bindings plus generated grammar packages

Current host constraint:

- the published `github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go`
  module is not yet usable as a drop-in backend in this workspace because it
  requires Go `1.26` and its published module payload omits the `include/` and
  `lib/` artifacts referenced by its own CGO bindings
- first PEG candidate for a second backend path: `pigeon`
- current TOML family backend plurality:
  - semantic parser: `go-toml/v2`
  - PEG syntax-validation parser: `pigeon`

Requirements:

- clean parser lifecycle management
- grammar selection abstraction
- stable node wrappers or adapters
- shared diagnostic shape for conformance runner

## MVP Deliverables

### 1. `tree-haver-go`

- parser registry
- grammar loading
- parse result/diagnostic reporting

### 2. `ast-merge-go`

- merge result structs
- diagnostic structs
- matching/refinement interfaces
- freeze region model

### 3. `plain-merge-go`

- normalized text segmentation
- block matching
- threshold-based similarity API

### 4. `json-merge-go`

- strict JSON/JSONC merge support
- comments support where grammar/runtime provides it
- targeted recovery hook boundaries

### 5. Fixture Runner

- reads shared fixtures from workspace
- compares expected output and diagnostics

## Non-Goals For V1

- direct feature parity with all Ruby merge gems
- scaffolding/templating packages
- broad ecosystem integration before fixture parity

## Open Questions

1. Is comment-preservation support feasible enough for v1?
2. Should Go prioritize library embedding or CLI tooling first?

## Decisions

- Use one monorepo rooted at `structuredmerge-go`.
- Start with one root Go module and multiple publishable packages.
- Keep `tree-haver` focused on reusable parser frameworks such as tree-sitter
  and `pigeon`.
- Keep one-trick parser integrations such as `go/parser` inside merge-family
  packages, not in `tree-haver`.
- Use the same family-facing TOML fixtures across both `go-toml/v2` and
  `pigeon` backend paths unless a backend restriction is declared explicitly.

## First Implementation Sequence

1. define Go merge result and diagnostic types
2. implement `tree-haver-go`
3. implement fixture runner
4. implement `plain-merge-go`
5. implement `json-merge-go`
