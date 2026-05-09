package prices

import (
	"embed"
	"fmt"
	"strings"

	"charm.land/fantasy"
)

//go:embed prompts/*.md
var priceLLMPromptFiles embed.FS

type priceLLMOperation struct {
	SystemTemplate    string
	UserTemplate      string
	SchemaName        string
	SchemaDescription string
	MaxOutputTokens   int64
}

var priceLLMOperations = map[string]priceLLMOperation{
	"postal_neighborhood_match": {
		SystemTemplate:    "prompts/postal_neighborhood_match.system.md",
		UserTemplate:      "prompts/postal_neighborhood_match.user.md",
		SchemaName:        "match_postal_neighborhood",
		SchemaDescription: "Match a Finnish neighborhood name to one available postal code area",
		MaxOutputTokens:   500,
	},
}

func priceLLMOperationConfig(operation string) priceLLMOperation {
	return priceLLMOperations[operation]
}

func priceLLMPrompt(operation string, replacements map[string]string) (fantasy.Prompt, error) {
	config, ok := priceLLMOperations[operation]
	if !ok {
		return nil, fmt.Errorf("unknown price llm operation %q", operation)
	}
	system, err := renderPriceLLMTemplate(config.SystemTemplate, replacements)
	if err != nil {
		return nil, err
	}
	user, err := renderPriceLLMTemplate(config.UserTemplate, replacements)
	if err != nil {
		return nil, err
	}
	return fantasy.Prompt{fantasy.NewSystemMessage(system), fantasy.NewUserMessage(user)}, nil
}

func renderPriceLLMTemplate(path string, replacements map[string]string) (string, error) {
	raw, err := priceLLMPromptFiles.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read price llm prompt template %s: %w", path, err)
	}
	pairs := make([]string, 0, len(replacements)*2)
	for key, value := range replacements {
		pairs = append(pairs, "{{"+key+"}}", value)
	}
	return strings.NewReplacer(pairs...).Replace(string(raw)), nil
}
