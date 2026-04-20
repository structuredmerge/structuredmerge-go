module github.com/structuredmerge/structuredmerge-go

go 1.26

require (
	github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go v1.6.2
	github.com/pelletier/go-toml/v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go => ../tree-sitter-language-pack/packages/go
