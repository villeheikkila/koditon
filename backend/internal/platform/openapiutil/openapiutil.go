package openapiutil

import "github.com/danielgtaylor/huma/v2"

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
