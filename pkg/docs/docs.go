package docs

import (
	_ "embed"
)

//go:embed tend-examples.xml
var TendExamplesXML []byte

// GetTendExamples returns the content of the embedded tend-examples.xml file.
func GetTendExamples() []byte {
	return TendExamplesXML
}