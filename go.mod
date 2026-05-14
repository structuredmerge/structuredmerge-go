module github.com/structuredmerge/structuredmerge-go

go 1.26

require (
	github.com/goccy/go-yaml v1.19.2
	github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go v1.8.0-rc.26
	github.com/pelletier/go-toml/v2 v2.3.0
	github.com/yuin/goldmark v1.8.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dave/dst v0.27.4
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/tools v0.1.12 // indirect
)

replace github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go => ../tree-sitter-language-pack/packages/go
