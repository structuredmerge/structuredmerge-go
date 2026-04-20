module github.com/structuredmerge/structuredmerge-go

go 1.26

require (
	github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go v1.6.2
	github.com/pelletier/go-toml/v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
)

replace github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go => ../tree-sitter-language-pack/packages/go
