package api_test

import (
	"fmt"

	"github.com/MontFerret/specs/pkg/api"
)

func ExampleParseDocumentation() {
	text := `Decode decodes XML content into a normalized document object.

@param data {String|Binary} XML content.
@return {Object} Normalized XML document.`

	documentation, err := api.ParseDocumentation(text)
	if err != nil {
		panic(err)
	}

	fmt.Println(documentation.Description)
	parameterType := documentation.Parameters[0].Type
	fmt.Println(documentation.Parameters[0].Name, parameterType.Types[0].Name, parameterType.Types[1].Name)
	fmt.Println(documentation.Return.Type.Name)
	// Output:
	// Decode decodes XML content into a normalized document object.
	// data String Binary
	// Object
}
