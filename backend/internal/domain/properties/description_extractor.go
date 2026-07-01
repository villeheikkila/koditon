package properties

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"

	"charm.land/fantasy"
	fantasyobject "charm.land/fantasy/object"
)

type DescriptionExtractionResult struct {
	SaleListingID string                            `json:"sale_listing_id"`
	Model         string                            `json:"model"`
	Items         []DescriptionExtractionResultItem `json:"items"`
}

type DescriptionExtractionResultItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Direction   string `json:"direction"`
	Severity    string `json:"severity"`
	Confidence  int32  `json:"confidence"`
	Text        string `json:"text,omitempty"`
	SourceField string `json:"source_field"`
}

type descriptionExtractionObject struct {
	Items []descriptionExtractionItem `json:"items" description:"Structured apartment-quality, location, view, usability, and risk signals from the description text."`
}

type descriptionExtractionItem struct {
	Key         string `json:"key" description:"Signal key: floor_ground, floor_top, floor_high, elevator_missing, view_open, view_courtyard, quiet, light, condition_needs_surface, condition_renovated, layout_good, layout_awkward, storage_good, parking_good, balcony_positive, sauna_positive, accessibility_risk, premium_architecture, noise_risk, moisture_risk, unclear_claim."`
	Value       string `json:"value" description:"Concise normalized value or short label."`
	Direction   string `json:"direction" enum:"positive,negative,neutral" description:"Whether this supports or hurts offer attractiveness."`
	Severity    string `json:"severity" enum:"low,medium,high" description:"Importance for valuation or liquidity."`
	Confidence  int32  `json:"confidence" description:"Confidence from 0 to 100."`
	Text        string `json:"text,omitempty" description:"Short quote or paraphrase supporting the signal."`
	SourceField string `json:"source_field" enum:"description_text,building_text,additional_info_text" description:"Text field where the signal came from."`
}

func (s *Service) ExtractSaleListingDescriptionInsights(ctx context.Context, input string, modelName string) (DescriptionExtractionResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return DescriptionExtractionResult{}, ErrNotFound
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return DescriptionExtractionResult{}, err
	}
	return s.extractSourceListingDescriptionInsights(ctx, saleListingID, modelName)
}

func (s *Service) extractSourceListingDescriptionInsights(ctx context.Context, saleListingID uuid.UUID, modelName string) (DescriptionExtractionResult, error) {
	if strings.TrimSpace(s.renovationExtractorAPIKey) == "" {
		return DescriptionExtractionResult{}, ErrRenovationExtractorNotConfigured
	}
	row, err := s.queries.GetPropertySourceOfferingDescriptionTexts(ctx, &saleListingID)
	if err != nil {
		return DescriptionExtractionResult{}, mapNotFound(err)
	}
	descriptionText := cleanDisplayString(stringValue(row.DescriptionText))
	buildingText := cleanDisplayString(stringValue(row.BuildingText))
	additionalInfoText := cleanDisplayString(stringValue(row.AdditionalInfoText))
	modelName = firstNonEmpty(modelName, s.renovationExtractorModelName, "~google/gemini-flash-latest")
	result := DescriptionExtractionResult{SaleListingID: saleListingID.String(), Model: modelName}
	if descriptionText == "" && buildingText == "" && additionalInfoText == "" {
		if err := s.replaceLLMDescriptionInsights(ctx, saleListingID, nil); err != nil {
			return DescriptionExtractionResult{}, err
		}
		return result, nil
	}
	model, err := s.renovationExtractionModel(ctx, modelName)
	if err != nil {
		return DescriptionExtractionResult{}, err
	}
	operation := propertyLLMOperationConfig("description_extraction")
	prompt, err := propertyLLMPrompt("description_extraction", map[string]string{"description_text": descriptionText, "building_text": buildingText, "additional_info_text": additionalInfoText})
	if err != nil {
		return DescriptionExtractionResult{}, err
	}
	objectResult, err := fantasyobject.Generate[descriptionExtractionObject](ctx, model, fantasy.ObjectCall{Prompt: prompt, SchemaName: operation.SchemaName, SchemaDescription: operation.SchemaDescription, Temperature: ptrFloat64(0), MaxOutputTokens: ptrInt64(operation.MaxOutputTokens)})
	if err != nil {
		return DescriptionExtractionResult{}, fmt.Errorf("extract description insights with fantasy: %w", err)
	}
	items := normalizeDescriptionExtractionItems(objectResult.Object.Items)
	if err := s.replaceLLMDescriptionInsights(ctx, saleListingID, items); err != nil {
		return DescriptionExtractionResult{}, err
	}
	result.Items = descriptionExtractionResultItems(items)
	return result, nil
}

func (s *Service) replaceLLMDescriptionInsights(ctx context.Context, saleListingID uuid.UUID, items []descriptionExtractionItem) error {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("database handle does not support transactions")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin description extraction transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	if err := queries.DeleteLLMPropertySourceOfferingInsights(ctx, &saleListingID); err != nil {
		return fmt.Errorf("delete previous llm description insights: %w", err)
	}
	for _, item := range items {
		sourceField := llmDescriptionSourceField(item.SourceField)
		if err := queries.InsertPropertySourceOfferingInsight(ctx, db.InsertPropertySourceOfferingInsightParams{SaleListingID: &saleListingID, SourceField: &sourceField, Key: &item.Key, Value: &item.Value, Direction: &item.Direction, Severity: &item.Severity, Confidence: &item.Confidence, Text: &item.Text}); err != nil {
			return fmt.Errorf("insert llm description insight: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit description extraction transaction: %w", err)
	}
	return nil
}

func normalizeDescriptionExtractionItems(items []descriptionExtractionItem) []descriptionExtractionItem {
	out := make([]descriptionExtractionItem, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item.Key = normalizeDescriptionInsightKey(item.Key)
		item.Value = cleanDisplayString(item.Value)
		item.Direction = normalizeInsightDirection(item.Direction)
		item.Severity = normalizeInsightSeverity(item.Severity)
		item.SourceField = normalizeDescriptionSourceField(item.SourceField)
		item.Text = cleanDisplayString(item.Text)
		if item.Confidence < 0 {
			item.Confidence = 0
		}
		if item.Confidence > 100 {
			item.Confidence = 100
		}
		if item.Confidence == 0 {
			item.Confidence = 50
		}
		if item.Key == "" || item.Value == "" {
			continue
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func descriptionExtractionResultItems(items []descriptionExtractionItem) []DescriptionExtractionResultItem {
	out := make([]DescriptionExtractionResultItem, 0, len(items))
	for _, item := range items {
		out = append(out, DescriptionExtractionResultItem{Key: item.Key, Value: item.Value, Direction: item.Direction, Severity: item.Severity, Confidence: item.Confidence, Text: item.Text, SourceField: llmDescriptionSourceField(item.SourceField)})
	}
	return out
}

func normalizeDescriptionInsightKey(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "-", "_")))
}

func normalizeInsightDirection(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "positive", "supports", "good":
		return "positive"
	case "negative", "risk", "bad":
		return "negative"
	default:
		return "neutral"
	}
}

func normalizeInsightSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func normalizeDescriptionSourceField(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "building_text":
		return "building_text"
	case "additional_info_text":
		return "additional_info_text"
	default:
		return "description_text"
	}
}

func llmDescriptionSourceField(value string) string {
	return "llm_" + normalizeDescriptionSourceField(value)
}
