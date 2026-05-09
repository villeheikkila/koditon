package properties

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/valuation"

	"charm.land/fantasy"
	fantasyobject "charm.land/fantasy/object"
)

type ValuationInputExtractionResult struct {
	SaleListingID string                    `json:"sale_listing_id"`
	Model         string                    `json:"model"`
	Facts         []valuation.ValuationFact `json:"facts"`
}

type valuationInputExtractionObject struct {
	Facts []valuationInputExtractionFact `json:"facts" description:"Canonical apartment valuation input facts extracted from listing layout and text fields."`
}

type valuationInputExtractionFact struct {
	Section     string   `json:"section" description:"Canonical valuation input section."`
	Key         string   `json:"key" description:"Stable snake_case key within the section."`
	ValueKind   string   `json:"value_kind" enum:"text,number,bool" description:"Type of extracted value."`
	ValueText   string   `json:"value_text,omitempty" description:"Text or categorical value when value_kind is text."`
	ValueNumber *float64 `json:"value_number,omitempty" description:"Numeric value when value_kind is number."`
	ValueBool   *bool    `json:"value_bool,omitempty" description:"Boolean value when value_kind is bool."`
	Confidence  int32    `json:"confidence" description:"Confidence from 0 to 100."`
	SourceField string   `json:"source_field" description:"Exact input field name supporting the fact."`
	Evidence    string   `json:"evidence,omitempty" description:"Short supporting phrase from the source field."`
}

const valuationClaimTargetSaleListing = "sale_listing"

func (s *Service) ExtractSaleListingValuationInputs(ctx context.Context, input string, modelName string) (ValuationInputExtractionResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return ValuationInputExtractionResult{}, ErrNotFound
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return ValuationInputExtractionResult{}, err
	}
	return s.extractSourceListingValuationInputs(ctx, saleListingID, modelName)
}

func (s *Service) extractSourceListingValuationInputs(ctx context.Context, saleListingID uuid.UUID, modelName string) (ValuationInputExtractionResult, error) {
	if strings.TrimSpace(s.renovationExtractorAPIKey) == "" {
		return ValuationInputExtractionResult{}, ErrRenovationExtractorNotConfigured
	}
	row, err := s.queries.GetPropertySourceOfferingValuationExtractionTexts(ctx, saleListingID)
	if err != nil {
		return ValuationInputExtractionResult{}, mapNotFound(err)
	}
	modelName = firstNonEmpty(modelName, s.renovationExtractorModelName, "~google/gemini-flash-latest")
	result := ValuationInputExtractionResult{SaleListingID: saleListingID.String(), Model: modelName}
	prompt, err := valuationInputExtractionPrompt(row)
	if err != nil {
		return ValuationInputExtractionResult{}, err
	}
	operation := propertyLLMOperationConfig("valuation_input_extraction")
	if !valuationInputPromptHasContent(row) {
		if err := s.replaceLLMPropertyClaims(ctx, saleListingID, modelName, operation.Version, nil); err != nil {
			return ValuationInputExtractionResult{}, err
		}
		return result, nil
	}
	model, err := s.renovationExtractionModel(ctx, modelName)
	if err != nil {
		return ValuationInputExtractionResult{}, err
	}
	objectResult, err := fantasyobject.Generate[valuationInputExtractionObject](ctx, model, fantasy.ObjectCall{Prompt: prompt, SchemaName: operation.SchemaName, SchemaDescription: operation.SchemaDescription, Temperature: ptrFloat64(0), MaxOutputTokens: ptrInt64(operation.MaxOutputTokens)})
	if err != nil {
		return ValuationInputExtractionResult{}, fmt.Errorf("extract valuation inputs with fantasy: %w", err)
	}
	facts := normalizeValuationInputFacts(objectResult.Object.Facts, modelName, operation.Version)
	if err := s.replaceLLMPropertyClaims(ctx, saleListingID, modelName, operation.Version, facts); err != nil {
		return ValuationInputExtractionResult{}, err
	}
	result.Facts = facts
	return result, nil
}

func (s *Service) replaceLLMPropertyClaims(ctx context.Context, saleListingID uuid.UUID, modelName string, promptVersion string, facts []valuation.ValuationFact) error {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("database handle does not support transactions")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin valuation input extraction transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	entity := db.DeleteLLMPropertyClaimsForEntityParams{EntityType: valuationClaimTargetSaleListing, EntityID: saleListingID}
	if err := queries.DeleteLLMPropertyClaimsForEntity(ctx, entity); err != nil {
		return fmt.Errorf("delete previous llm valuation facts: %w", err)
	}
	for _, fact := range facts {
		if err := queries.InsertPropertyClaim(ctx, db.InsertPropertyClaimParams{EntityType: valuationClaimTargetSaleListing, EntityID: saleListingID, SourceField: llmValuationFactSourceField(fact.Source), Section: fact.Section, Key: fact.Key, ValueKind: fact.ValueKind, ValueText: fact.ValueText, ValueNumber: fact.ValueNumber, ValueBool: fact.ValueBool, Confidence: fact.Confidence * 100, EvidenceText: fact.Evidence, Model: modelName, PromptVersion: promptVersion}); err != nil {
			return fmt.Errorf("insert llm valuation fact: %w", err)
		}
	}
	if _, err := queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "valuation_claims_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty after valuation extraction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit valuation input extraction transaction: %w", err)
	}
	return nil
}

func valuationInputExtractionPrompt(row db.GetPropertySourceOfferingValuationExtractionTextsRow) (fantasy.Prompt, error) {
	return propertyLLMPrompt("valuation_input_extraction", map[string]string{
		"room_layout":                      row.RoomLayout,
		"rooms_count":                      int32PromptValue(row.SaleListingRoomsCount),
		"bedrooms_count":                   int32PromptValue(row.SaleListingBedroomsCount),
		"area_m2":                          float64PromptValue(row.SaleListingAreaValue),
		"living_area_m2":                   float64PromptValue(row.SaleListingLivingAreaValue),
		"total_area_m2":                    float64PromptValue(row.SaleListingTotalAreaValue),
		"other_area_m2":                    float64PromptValue(row.SaleListingOtherAreaValue),
		"floor_level":                      int32PromptValue(row.SaleListingFloorLevel),
		"total_floors":                     int32PromptValue(row.SaleListingTotalFloors),
		"floor_text":                       row.FloorText,
		"condition":                        row.Condition,
		"sauna":                            boolPromptValue(row.SaleListingSauna),
		"balcony":                          boolPromptValue(row.SaleListingBalcony),
		"parking_text":                     row.ParkingText,
		"description_text":                 row.DescriptionText,
		"additional_info_text":             row.AdditionalInfoText,
		"kitchen_description_text":         row.KitchenDescriptionText,
		"bathroom_description_text":        row.BathroomDescriptionText,
		"storage_description_text":         row.StorageDescriptionText,
		"floor_materials_description_text": row.FloorMaterialsDescriptionText,
		"wall_materials_description_text":  row.WallMaterialsDescriptionText,
		"balcony_description_text":         row.BalconyDescriptionText,
		"sauna_description_text":           row.SaunaDescriptionText,
		"views_description_text":           row.ViewsDescriptionText,
		"building_material":                row.BuildingMaterial,
		"heating_system":                   row.HeatingSystem,
		"roof_type":                        row.RoofType,
		"roof_material":                    row.RoofMaterial,
		"car_storage_text":                 row.CarStorageText,
		"building_description_text":        row.BuildingDescriptionText,
		"building_other_info_text":         row.BuildingOtherInfoText,
		"charges_text":                     row.ChargesText,
	})
}

func valuationInputPromptHasContent(row db.GetPropertySourceOfferingValuationExtractionTextsRow) bool {
	return firstNonEmpty(row.RoomLayout, row.FloorText, row.Condition, row.ParkingText, row.DescriptionText, row.AdditionalInfoText, row.KitchenDescriptionText, row.BathroomDescriptionText, row.StorageDescriptionText, row.FloorMaterialsDescriptionText, row.WallMaterialsDescriptionText, row.BalconyDescriptionText, row.SaunaDescriptionText, row.ViewsDescriptionText, row.BuildingMaterial, row.HeatingSystem, row.RoofType, row.RoofMaterial, row.CarStorageText, row.BuildingDescriptionText, row.BuildingOtherInfoText, row.ChargesText) != "" || row.SaleListingRoomsCount != nil || row.SaleListingAreaValue != nil || row.SaleListingFloorLevel != nil || row.SaleListingSauna != nil || row.SaleListingBalcony != nil
}

func normalizeValuationInputFacts(items []valuationInputExtractionFact, modelName string, promptVersion string) []valuation.ValuationFact {
	out := make([]valuation.ValuationFact, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		section := normalizeDescriptionInsightKey(item.Section)
		key := normalizeDescriptionInsightKey(item.Key)
		valueKind := normalizeValuationFactValueKind(item.ValueKind, item)
		fact := valuation.ValuationFact{Section: section, Key: key, ValueKind: valueKind, ValueText: cleanDisplayString(item.ValueText), ValueNumber: item.ValueNumber, ValueBool: item.ValueBool, Confidence: float64(normalizeConfidence(item.Confidence)) / 100, Source: normalizeValuationFactSourceField(item.SourceField), Evidence: cleanDisplayString(item.Evidence), Model: modelName, Prompt: promptVersion}
		if !valuationFactHasValue(fact) || section == "" || key == "" {
			continue
		}
		dedupeKey := fact.Source + ":" + section + ":" + key
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}
		out = append(out, fact)
	}
	return out
}

func normalizeValuationFactValueKind(value string, item valuationInputExtractionFact) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "number":
		if item.ValueNumber != nil {
			return "number"
		}
	case "bool":
		if item.ValueBool != nil {
			return "bool"
		}
	}
	return "text"
}

func valuationFactHasValue(fact valuation.ValuationFact) bool {
	switch fact.ValueKind {
	case "number":
		return fact.ValueNumber != nil
	case "bool":
		return fact.ValueBool != nil
	default:
		return fact.ValueText != ""
	}
}

func normalizeValuationFactSourceField(value string) string {
	normalized := normalizeDescriptionInsightKey(value)
	if normalized == "" {
		return "extraction"
	}
	return normalized
}

func llmValuationFactSourceField(value string) string {
	return "llm_" + normalizeValuationFactSourceField(value)
}

func normalizeConfidence(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	if value == 0 {
		return 50
	}
	return value
}

func int32PromptValue(value *int32) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(*value)
}

func float64PromptValue(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(*value)
}

func boolPromptValue(value *bool) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(*value)
}
