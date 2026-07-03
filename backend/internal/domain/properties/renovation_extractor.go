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
	fantasyopenrouter "charm.land/fantasy/providers/openrouter"
)

type RenovationExtractionResult struct {
	SaleListingID string                           `json:"sale_listing_id"`
	Model         string                           `json:"model"`
	Items         []RenovationExtractionResultItem `json:"items"`
}

type RenovationExtractionResultItem struct {
	Category        string `json:"category"`
	Component       string `json:"component,omitempty"`
	Status          string `json:"status"`
	Year            *int32 `json:"year,omitempty"`
	Scope           string `json:"scope,omitempty"`
	Stage           string `json:"stage,omitempty"`
	Responsibility  string `json:"responsibility,omitempty"`
	CostEstimateEUR *int64 `json:"cost_estimate_eur,omitempty"`
	Text            string `json:"text,omitempty"`
	Confidence      int32  `json:"confidence"`
	SourceField     string `json:"source_field"`
}

type renovationExtractionObject struct {
	Items []renovationExtractionItem `json:"items" description:"Structured renovation items mentioned in the source texts. Return an empty array when no renovation is mentioned."`
}

type renovationExtractionItem struct {
	Category        string `json:"category" description:"Normalized renovation category, for example pipe, sewer, facade, roof, window, balcony, electricity, elevator, heating, ventilation, drainage, yard, bathroom, kitchen, common_area, water_supply, other."`
	Component       string `json:"component,omitempty" description:"More specific component when available, for example sewer, water_supply, riser, bathroom, heat_exchanger, district_heating, geothermal, flat_roof, pitched_roof, courtyard, foundation."`
	Status          string `json:"status" enum:"done,planned,unknown" description:"done for completed work, planned for future or upcoming work, unknown when the text mentions renovation without timing."`
	Year            *int32 `json:"year,omitempty" description:"Four digit year if explicitly stated or clearly implied."`
	Scope           string `json:"scope,omitempty" enum:"full,partial,survey,maintenance,planning,unknown" description:"Scope of the item. Use survey for kuntotutkimus/kartoitus/kuvaus, planning for hankesuunnittelu/tarveselvitys/kilpailutus, partial for huolto/osittainen/sukitus/maalaus/lakkaus, full for uusinta/saneeraus/peruskorjaus."`
	Stage           string `json:"stage,omitempty" enum:"need_assessment,condition_survey,project_planning,tendering,decision,execution,completed,maintenance,unknown" description:"Project stage when stated or implied."`
	Responsibility  string `json:"responsibility,omitempty" enum:"housing_company,shareholder,mixed,unknown" description:"Who usually pays or is responsible according to the text. Housing company for taloyhtiö/kiinteistö work, shareholder for huoneisto/märkätila/keittiö only when clearly apartment-level."`
	CostEstimateEUR *int64 `json:"cost_estimate_eur,omitempty" description:"EUR cost estimate if explicitly stated. Leave empty if not stated."`
	Text            string `json:"text,omitempty" description:"Short source quote or paraphrase supporting the extraction."`
	Confidence      int32  `json:"confidence" description:"Confidence from 0 to 100."`
	SourceField     string `json:"source_field" enum:"renovations_done_text,renovations_planned_text" description:"The source text field where the item came from."`
}

func (s *Service) ExtractSaleListingRenovations(ctx context.Context, input string, modelName string) (RenovationExtractionResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return RenovationExtractionResult{}, ErrNotFound
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return RenovationExtractionResult{}, err
	}
	return s.extractSourceListingRenovations(ctx, saleListingID, modelName)
}

func (s *Service) extractSourceListingRenovations(ctx context.Context, saleListingID uuid.UUID, modelName string) (RenovationExtractionResult, error) {
	if strings.TrimSpace(s.renovationExtractorAPIKey) == "" {
		return RenovationExtractionResult{}, ErrRenovationExtractorNotConfigured
	}
	row, err := s.queries.GetSaleListingRenovationExtractionTexts(ctx, saleListingID)
	if err != nil {
		return RenovationExtractionResult{}, mapNotFound(err)
	}
	doneText := cleanDisplayString(valueOrEmpty(row.DoneText))
	plannedText := cleanDisplayString(valueOrEmpty(row.PlannedText))
	modelName = firstNonEmpty(modelName, s.renovationExtractorModelName, "~google/gemini-flash-latest")
	result := RenovationExtractionResult{SaleListingID: saleListingID.String(), Model: modelName}
	if doneText == "" && plannedText == "" {
		if err := s.replaceLLMRenovationRows(ctx, saleListingID, nil); err != nil {
			return RenovationExtractionResult{}, err
		}
		return result, nil
	}
	model, err := s.renovationExtractionModel(ctx, modelName)
	if err != nil {
		return RenovationExtractionResult{}, err
	}
	operation := propertyLLMOperationConfig("renovation_extraction")
	prompt, err := propertyLLMPrompt("renovation_extraction", map[string]string{"renovations_done_text": doneText, "renovations_planned_text": plannedText})
	if err != nil {
		return RenovationExtractionResult{}, err
	}
	objectResult, err := fantasyobject.Generate[renovationExtractionObject](ctx, model, fantasy.ObjectCall{Prompt: prompt, SchemaName: operation.SchemaName, SchemaDescription: operation.SchemaDescription, Temperature: ptrFloat64(0), MaxOutputTokens: new(operation.MaxOutputTokens)})
	if err != nil {
		return RenovationExtractionResult{}, fmt.Errorf("extract renovations with fantasy: %w", err)
	}
	items := normalizeRenovationExtractionItems(objectResult.Object.Items)
	if err := s.replaceLLMRenovationRows(ctx, saleListingID, items); err != nil {
		return RenovationExtractionResult{}, err
	}
	result.Items = renovationExtractionResultItems(items)
	return result, nil
}

func (s *Service) renovationExtractionModel(ctx context.Context, modelName string) (fantasy.LanguageModel, error) {
	provider, err := fantasyopenrouter.New(fantasyopenrouter.WithAPIKey(s.renovationExtractorAPIKey), fantasyopenrouter.WithObjectMode(fantasy.ObjectModeText))
	if err != nil {
		return nil, fmt.Errorf("create fantasy openrouter provider: %w", err)
	}
	model, err := provider.LanguageModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("create fantasy language model %q: %w", modelName, err)
	}
	return model, nil
}

func (s *Service) replaceLLMRenovationRows(ctx context.Context, saleListingID uuid.UUID, items []renovationExtractionItem) error {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("database handle does not support transactions")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin renovation extraction transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	if err := queries.DeleteLLMPropertySourceOfferingRenovations(ctx, &saleListingID); err != nil {
		return fmt.Errorf("delete previous llm renovation rows: %w", err)
	}
	for _, item := range items {
		sourceField := llmRenovationSourceField(item.SourceField)
		if err := queries.InsertLLMPropertySourceOfferingRenovation(ctx, db.InsertLLMPropertySourceOfferingRenovationParams{SaleListingID: &saleListingID, SourceField: &sourceField, Category: &item.Category, Status: &item.Status, Year: item.Year, Component: &item.Component, Scope: &item.Scope, Stage: &item.Stage, Responsibility: &item.Responsibility, CostEstimateEur: item.CostEstimateEUR, Summary: &item.Text, Confidence: &item.Confidence}); err != nil {
			return fmt.Errorf("insert llm renovation row: %w", err)
		}
	}
	if err := ProjectListingRenovationEvents(ctx, tx, saleListingID); err != nil {
		return err
	}
	if _, err := queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "renovation_events_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty after extraction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit renovation extraction transaction: %w", err)
	}
	return nil
}

func normalizeRenovationExtractionItems(items []renovationExtractionItem) []renovationExtractionItem {
	out := make([]renovationExtractionItem, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item.Category = normalizeRenovationCategory(item.Category)
		item.Component = normalizeRenovationComponent(item.Component)
		item.Status = normalizeRenovationStatus(item.Status)
		item.Scope = normalizeRenovationScope(firstNonEmpty(item.Scope, inferRenovationScope(item.Text)))
		item.Stage = normalizeRenovationStage(firstNonEmpty(item.Stage, inferRenovationStage(item.Text)))
		item.Responsibility = normalizeRenovationResponsibility(firstNonEmpty(item.Responsibility, inferRenovationResponsibility(item.Text)))
		item.SourceField = normalizeRenovationSourceField(item.SourceField, item.Status)
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
		if item.Year != nil && (*item.Year < 1800 || *item.Year > 2200) {
			item.Year = nil
		}
		if item.Category == "" {
			continue
		}
		key := fmt.Sprintf("%s:%s:%s:%s:%s:%d", item.SourceField, item.Category, item.Component, item.Status, item.Stage, ptrInt32Value(item.Year))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func renovationExtractionResultItems(items []renovationExtractionItem) []RenovationExtractionResultItem {
	out := make([]RenovationExtractionResultItem, 0, len(items))
	for _, item := range items {
		out = append(out, RenovationExtractionResultItem{Category: item.Category, Component: item.Component, Status: item.Status, Year: item.Year, Scope: item.Scope, Stage: item.Stage, Responsibility: item.Responsibility, CostEstimateEUR: item.CostEstimateEUR, Text: item.Text, Confidence: item.Confidence, SourceField: llmRenovationSourceField(item.SourceField)})
	}
	return out
}

func normalizeRenovationCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "pipes", "pipe renovation", "plumbing", "putki", "putkiremontti", "linjasaneeraus":
		return "pipe"
	case "sewer", "sewers", "viemari", "viemäri", "viemarit", "sukitus":
		return "sewer"
	case "facades", "facade renovation", "julkisivu":
		return "facade"
	case "windows", "ikkuna", "ikkunat":
		return "window"
	case "balconies", "parveke", "parvekkeet":
		return "balcony"
	case "electric", "electrical", "sahko", "sähkö":
		return "electricity"
	case "ventilation", "air ventilation", "iv", "ilmanvaihto":
		return "ventilation"
	case "drainage", "salaoja", "salaojat":
		return "drainage"
	case "heating", "lämmitys", "lammitys", "lämmönsiirrin", "lammonsiirrin":
		return "heating"
	case "common areas", "common area", "sauna", "pesutupa", "kuivaushuone":
		return "common_area"
	case "water", "water supply", "vesijohto", "tonttivesijohto":
		return "water_supply"
	}
	return value
}

func normalizeRenovationComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "unknown":
		return ""
	case "sewers", "viemari", "viemäri", "viemarit", "sukitus", "tontti- ja pohjaviemari", "tontti- ja pohjaviemäri":
		return "sewer"
	case "water", "water pipes", "vesijohto", "kayttovesi", "käyttövesi", "tonttivesijohto":
		return "water_supply"
	case "riser", "risers", "sähkönousut", "sahkonousut":
		return "riser"
	case "bathrooms", "märkätila", "markatila", "märkätilat", "markatilat":
		return "bathroom"
	case "heat exchanger", "lämmönsiirrin", "lammonsiirrin":
		return "heat_exchanger"
	case "district heating", "kaukolämpö", "kaukolampo":
		return "district_heating"
	case "geothermal", "maalämpö", "maalampo":
		return "geothermal"
	case "flat roof", "tasakatto":
		return "flat_roof"
	case "pitched roof", "harjakatto", "peltikate":
		return "pitched_roof"
	case "courtyard", "piha", "sisäpiha", "sisapiha":
		return "courtyard"
	case "foundation", "perustus", "sokkelit", "sokkeli":
		return "foundation"
	default:
		return value
	}
}

func normalizeRenovationStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "done", "completed", "complete", "finished":
		return "done"
	case "planned", "upcoming", "future", "decided", "proposed":
		return "planned"
	default:
		return "unknown"
	}
}

func normalizeRenovationScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "full", "complete", "renewal", "peruskorjaus", "saneeraus", "uusinta":
		return "full"
	case "partial", "maintenance", "huolto", "osittainen", "sukitus", "maalaus", "lakkaus":
		return "partial"
	case "survey", "inspection", "condition_survey", "kuntotutkimus", "kartoitus", "kuvaus":
		return "survey"
	case "planning", "plan", "need_assessment", "project_planning", "tarveselvitys", "hankesuunnittelu", "suunnittelu", "kilpailutus":
		return "planning"
	default:
		return "unknown"
	}
}

func normalizeRenovationStage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "need_assessment", "tarveselvitys", "kunnossapitotarveselvitys":
		return "need_assessment"
	case "condition_survey", "survey", "inspection", "kuntotutkimus", "kartoitus", "kuvaus":
		return "condition_survey"
	case "project_planning", "planning", "hankesuunnittelu", "suunnittelu":
		return "project_planning"
	case "tendering", "kilpailutus":
		return "tendering"
	case "decision", "paatos", "päätös":
		return "decision"
	case "execution", "urakka", "toteutus":
		return "execution"
	case "completed", "done", "valmis", "toteutettu":
		return "completed"
	case "maintenance", "huolto":
		return "maintenance"
	default:
		return "unknown"
	}
}

func normalizeRenovationResponsibility(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "housing_company", "company", "taloyhtio", "taloyhtiö", "kiinteistö", "kiinteisto":
		return "housing_company"
	case "shareholder", "osakas", "huoneisto":
		return "shareholder"
	case "mixed", "both":
		return "mixed"
	default:
		return "unknown"
	}
}

func normalizeRenovationSourceField(value string, status string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "renovations_done_text", "done":
		return "renovations_done_text"
	case "renovations_planned_text", "planned":
		return "renovations_planned_text"
	default:
		if status == "planned" {
			return "renovations_planned_text"
		}
		return "renovations_done_text"
	}
}

func llmRenovationSourceField(value string) string {
	if normalizeRenovationSourceField(value, "") == "renovations_planned_text" {
		return "llm_renovations_planned_text"
	}
	return "llm_renovations_done_text"
}
