package docs

import (
	_ "embed"
)

//go:embed docs.json
var DocsJSON []byte

// GetDocsJSON returns the content of the embedded docs.json file.
func GetDocsJSON() []byte {
	return DocsJSON
}
