package properties

import (
	"context"
	"os"
	"testing"

	"charm.land/fantasy"
	fantasyobject "charm.land/fantasy/object"
)

func TestLiveFantasyRenovationExtraction(t *testing.T) {
	if os.Getenv("KODITON_LIVE_LLM_TESTS") != "1" {
		t.Skip("set KODITON_LIVE_LLM_TESTS=1 to run live renovation extraction")
	}
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY is required for live renovation extraction")
	}
	service := NewService(nil, WithOpenRouterRenovationExtractor(apiKey, "~google/gemini-flash-latest"))
	model, err := service.renovationExtractionModel(context.Background(), "~google/gemini-flash-latest")
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	result, err := fantasyobject.Generate[renovationExtractionObject](context.Background(), model, fantasy.ObjectCall{Prompt: fantasy.Prompt{fantasy.NewUserMessage(renovationExtractionPrompt("Kylpyhuone remontoitu 2019. Ikkunat uusittu 2021.", "Putkiremontti suunnitteilla 2027. Julkisivun kuntotutkimus tulossa."))}, SchemaName: "extract_apartment_renovations", SchemaDescription: "Extract structured apartment and housing company renovations from Finnish real-estate listing text", Temperature: ptrFloat64(0), MaxOutputTokens: ptrInt64(1400)})
	if err != nil {
		t.Fatalf("generate object: %v", err)
	}
	items := normalizeRenovationExtractionItems(result.Object.Items)
	var hasPlannedPipe, hasDoneBathroom bool
	for _, item := range items {
		hasPlannedPipe = hasPlannedPipe || item.Category == "pipe" && item.Status == "planned" && item.Year != nil && *item.Year == 2027
		hasDoneBathroom = hasDoneBathroom || item.Category == "bathroom" && item.Status == "done" && item.Year != nil && *item.Year == 2019
	}
	if !hasPlannedPipe || !hasDoneBathroom {
		t.Fatalf("expected planned pipe and done bathroom renovations, got %#v", items)
	}
}
