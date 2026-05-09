package properties

import (
	"embed"
	"fmt"
	"strings"

	"charm.land/fantasy"
)

//go:embed prompts/*.md
var propertyLLMPromptFiles embed.FS

type propertyLLMOperation struct {
	Version           string
	SystemTemplate    string
	UserTemplate      string
	SchemaName        string
	SchemaDescription string
	MaxOutputTokens   int64
}

var propertyLLMOperations = map[string]propertyLLMOperation{
	"renovation_extraction": {
		Version:           "renovation_extraction.v1",
		SystemTemplate:    "prompts/renovation_extraction.system.md",
		UserTemplate:      "prompts/renovation_extraction.user.md",
		SchemaName:        "extract_apartment_renovations",
		SchemaDescription: "Extract structured apartment and housing company renovations from Finnish real-estate listing text",
		MaxOutputTokens:   6000,
	},
	"description_extraction": {
		Version:           "description_extraction.v1",
		SystemTemplate:    "prompts/description_extraction.system.md",
		UserTemplate:      "prompts/description_extraction.user.md",
		SchemaName:        "extract_apartment_description_signals",
		SchemaDescription: "Extract structured apartment offer signals from Finnish real-estate description text",
		MaxOutputTokens:   4000,
	},
	"valuation_input_extraction": {
		Version:           "valuation_inputs.v1",
		SystemTemplate:    "prompts/valuation_inputs.system.md",
		UserTemplate:      "prompts/valuation_inputs.user.md",
		SchemaName:        "extract_apartment_valuation_inputs",
		SchemaDescription: "Extract structured apartment valuation inputs from layout and description fields",
		MaxOutputTokens:   5000,
	},
	"manager_certificate_extraction": {
		Version:           "manager_certificate_pdf_v2",
		SystemTemplate:    "prompts/manager_certificate_extraction.system.md",
		UserTemplate:      "prompts/manager_certificate_extraction.user.md",
		SchemaName:        "extract_manager_certificate",
		SchemaDescription: "Extract normalized apartment, building, housing company, finance, renovation, and risk facts from a Finnish isännöitsijäntodistus PDF",
		MaxOutputTokens:   20000,
	},
	"house_overview_generation": {
		Version:           "house_overview.v1",
		SystemTemplate:    "prompts/house_overview.system.md",
		UserTemplate:      "prompts/house_overview.user.md",
		SchemaName:        "generate_house_overview",
		SchemaDescription: "Generate a concise housing company overview from preprocessed structured facts",
		MaxOutputTokens:   3500,
	},
}

func propertyLLMOperationConfig(operation string) propertyLLMOperation {
	return propertyLLMOperations[operation]
}

func propertyLLMPrompt(operation string, replacements map[string]string, files ...fantasy.FilePart) (fantasy.Prompt, error) {
	system, user, err := propertyLLMPromptText(operation, replacements)
	if err != nil {
		return nil, err
	}
	return fantasy.Prompt{fantasy.NewSystemMessage(system), fantasy.NewUserMessage(user, files...)}, nil
}

func propertyLLMPromptText(operation string, replacements map[string]string) (string, string, error) {
	config, ok := propertyLLMOperations[operation]
	if !ok {
		return "", "", fmt.Errorf("unknown property llm operation %q", operation)
	}
	system, err := renderPropertyLLMTemplate(config.SystemTemplate, replacements)
	if err != nil {
		return "", "", err
	}
	user, err := renderPropertyLLMTemplate(config.UserTemplate, replacements)
	if err != nil {
		return "", "", err
	}
	return system, user, nil
}

func renderPropertyLLMTemplate(path string, replacements map[string]string) (string, error) {
	raw, err := propertyLLMPromptFiles.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read property llm prompt template %s: %w", path, err)
	}
	pairs := make([]string, 0, len(replacements)*2)
	for key, value := range replacements {
		pairs = append(pairs, "{{"+key+"}}", value)
	}
	return strings.NewReplacer(pairs...).Replace(string(raw)), nil
}
