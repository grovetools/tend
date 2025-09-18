package docs

import (
	_ "embed"
)

//go:embed tend-docs.json
var TendDocsJSON []byte

// GetTendDocs returns the content of the embedded tend-docs.json file.
func GetTendDocs() []byte {
	return TendDocsJSON
}