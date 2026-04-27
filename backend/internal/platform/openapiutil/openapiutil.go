package openapiutil

import "github.com/danielgtaylor/huma/v2"

func EnsureSchema(oapi *huma.OpenAPI, name string, schema *huma.Schema) {
	if oapi == nil || schema == nil || name == "" || oapi.Components.Schemas == nil {
		return
	}
	if _, exists := oapi.Components.Schemas.Map()[name]; exists {
		return
	}
	oapi.Components.Schemas.Map()[name] = schema
}

func EnsureExample(oapi *huma.OpenAPI, name string, example *huma.Example) {
	if oapi == nil || example == nil || name == "" {
		return
	}
	if oapi.Components.Examples == nil {
		oapi.Components.Examples = map[string]*huma.Example{}
	}
	if _, exists := oapi.Components.Examples[name]; exists {
		return
	}
	oapi.Components.Examples[name] = example
}

func SchemaRef(name string) *huma.Schema {
	return &huma.Schema{Ref: "#/components/schemas/" + name}
}

func ExampleRef(name string) *huma.Example {
	return &huma.Example{Ref: "#/components/examples/" + name}
}

func JSONResponse(description string, schema *huma.Schema) *huma.Response {
	return &huma.Response{
		Description: description,
		Content: map[string]*huma.MediaType{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

func ApplyBearerSecurity(op *huma.Operation, scopes []string) {
	if op == nil {
		return
	}
	if scopes == nil {
		scopes = []string{}
	}
	op.Security = []map[string][]string{
		{"bearer": scopes},
	}
	if len(scopes) == 0 {
		return
	}
	if op.Extensions == nil {
		op.Extensions = map[string]any{}
	}
	op.Extensions["x-koditon-scopes"] = scopes
}
