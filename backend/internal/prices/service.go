package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	openrouter "github.com/revrost/go-openrouter"

	"koditon-go/internal/db"
	"koditon-go/internal/logging"
	"koditon-go/internal/prices/client"
	"koditon-go/internal/util"
)

type Service struct {
	client        *client.Client
	queries       *db.Queries
	openRouterCli *openrouter.Client
	nowFunc       func() time.Time
}

func NewService(
	dbtx db.DBTX,
	baseURL string,
	openRouterCli *openrouter.Client,
) (*Service, error) {
	pricesClient, err := client.NewClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("create prices client: %w", err)
	}
	return &Service{
		client:        pricesClient,
		queries:       db.New(dbtx),
		openRouterCli: openRouterCli,
		nowFunc:       time.Now,
	}, nil
}

func (s *Service) FetchCities(ctx context.Context) ([]string, error) {
	cities, err := s.client.FetchCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cities: %w", err)
	}
	if len(cities) == 0 {
		return []string{}, nil
	}
	return cities, nil
}

type SyncAllConfig struct {
	Concurrency int
	Logger      *slog.Logger
}

func DefaultSyncAllConfig() SyncAllConfig {
	return SyncAllConfig{
		Concurrency: 1,
		Logger:      slog.Default(),
	}
}

type SyncAllResult struct {
	CitiesProcessed       int
	PostalCodesProcessed  int
	NeighborhoodsUpdated  int
	TransactionsProcessed int
	Errors                []error
}

type SearchTransactionsRow struct {
	City             string
	Municipality     string
	PostalCode       string
	PostalArea       string
	Neighborhood     string
	Description      string
	Type             string
	Category         string
	Area             float64
	Price            int32
	PricePerSqm      int32
	BuildYear        int32
	Floor            string
	Elevator         bool
	Condition        string
	Plot             string
	EnergyClass      string
	PeriodIdentifier string
	CreatedAt        time.Time
}

func (s *Service) SearchTransactionsByCityAndAddress(ctx context.Context, cityName, searchTerm string, limit int32) ([]SearchTransactionsRow, error) {
	cityName = strings.TrimSpace(cityName)
	searchTerm = strings.TrimSpace(searchTerm)
	if cityName == "" {
		return nil, fmt.Errorf("city name is required")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.queries.SearchTransactionsByCityAndAddress(ctx, db.SearchTransactionsByCityAndAddressParams{
		CityName:   cityName,
		SearchTerm: searchTerm,
		LimitCount: &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search transactions by city and address: %w", err)
	}
	result := make([]SearchTransactionsRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, SearchTransactionsRow{
			City:             row.PricesCityName,
			Municipality:     row.MunicipalityNameFi,
			PostalCode:       row.PostalCode,
			PostalArea:       row.PostalAreaNameFi,
			Neighborhood:     row.PricesNeighborhoodName,
			Description:      row.PricesTransactionDescription,
			Type:             row.PricesTransactionType,
			Category:         row.PricesTransactionCategory,
			Area:             row.PricesTransactionArea,
			Price:            row.PricesTransactionPrice,
			PricePerSqm:      row.PricesTransactionPricePerSquareMeter,
			BuildYear:        row.PricesTransactionBuildYear,
			Floor:            ptrString(row.PricesTransactionFloor),
			Elevator:         row.PricesTransactionElevator,
			Condition:        ptrString(row.PricesTransactionCondition),
			Plot:             ptrString(row.PricesTransactionPlot),
			EnergyClass:      ptrString(row.PricesTransactionEnergyClass),
			PeriodIdentifier: row.PricesTransactionPeriodIdentifier,
			CreatedAt:        row.PricesTransactionCreatedAt,
		})
	}
	return result, nil
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) SyncAll(ctx context.Context, cfg SyncAllConfig) (*SyncAllResult, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logging.With(logger, logging.Op("prices.sync_all"))
	result := &SyncAllResult{}
	cities, err := s.client.FetchCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cities: %w", err)
	}
	logger.InfoContext(ctx, "fetched cities", "count", len(cities))
	for _, cityName := range cities {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		cityResult, err := s.syncCityWithPostalCodes(ctx, cityName, logger)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("city %q: %w", cityName, err))
			logger.WarnContext(ctx, "prices city sync failed", "city", cityName, "error", err, "outcome", logging.OutcomeError)
			continue
		}
		result.CitiesProcessed++
		result.PostalCodesProcessed += cityResult.postalCodes
		result.NeighborhoodsUpdated += cityResult.neighborhoods
		result.TransactionsProcessed += cityResult.transactions
		logger.InfoContext(ctx, "prices city synced",
			"city", cityName,
			"postal_codes", cityResult.postalCodes,
			"neighborhoods", cityResult.neighborhoods,
			"transactions", cityResult.transactions,
			"outcome", logging.OutcomeSuccess,
		)
	}
	logger.InfoContext(ctx, "prices sync all completed",
		"cities", result.CitiesProcessed,
		"postal_codes", result.PostalCodesProcessed,
		"neighborhoods", result.NeighborhoodsUpdated,
		"transactions", result.TransactionsProcessed,
		"errors", len(result.Errors),
		"outcome", logging.OutcomeSuccess,
	)
	return result, nil
}

type cityResult struct {
	postalCodes   int
	neighborhoods int
	transactions  int
}

func (s *Service) syncCityWithPostalCodes(ctx context.Context, cityName string, logger *slog.Logger) (*cityResult, error) {
	logger = logging.With(logger, slog.String("city", cityName), logging.Op("prices.sync_city"))
	result := &cityResult{}
	cityRow, err := s.queries.UpsertPricesCity(ctx, mapUpsertCityParams(cityName))
	if err != nil {
		return nil, fmt.Errorf("upsert city: %w", err)
	}
	cityID := cityRow.PricesCityID
	postalCodes, err := s.client.FetchPostalCodes(ctx, cityName)
	if err != nil {
		return nil, fmt.Errorf("fetch postal codes: %w", err)
	}
	postalCodes = util.UniqueStrings(postalCodes)
	result.postalCodes = len(postalCodes)
	postalCodeMap := make(map[string]uuid.UUID, len(postalCodes))
	if len(postalCodes) > 0 {
		pcParams := mapUpsertPostalCodesBulkParams(postalCodes, cityID)
		rows, err := s.queries.UpsertPricesPostalCodesBulk(ctx, pcParams)
		if err != nil {
			return nil, fmt.Errorf("bulk upsert postal codes: %w", err)
		}
		for _, row := range rows {
			postalCodeMap[row.PricesPostalCodeCode] = row.PricesPostalCodeID
		}
	}
	allNeighborhoods := make(map[string]string)
	var allTransactions []*client.TransactionEntity
	for i, pc := range postalCodes {
		if i > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		transactions, err := s.client.GetAllTransactionsForPostalCodeFast(ctx, cityName, pc)
		if err != nil {
			logger.WarnContext(ctx, "prices postal code transaction fetch failed",
				"postal_code", pc,
				"error", err,
				"outcome", logging.OutcomeError,
			)
			continue
		}
		for _, tx := range transactions {
			name := util.TrimUnicodeSpace(tx.Neighborhood)
			if name != "" {
				allNeighborhoods[name] = pc
			}
		}
		allTransactions = append(allTransactions, transactions...)
	}
	neighborhoodNames := make([]string, 0, len(allNeighborhoods))
	for name := range allNeighborhoods {
		neighborhoodNames = append(neighborhoodNames, name)
	}
	result.neighborhoods = len(neighborhoodNames)
	neighborhoodIDs := make(map[string]uuid.UUID, len(neighborhoodNames))
	if len(neighborhoodNames) > 0 {
		nbParams := mapUpsertNeighborhoodsBulkParams(neighborhoodNames, cityID)
		rows, err := s.queries.UpsertPricesNeighborhoodsBulk(ctx, nbParams)
		if err != nil {
			return nil, fmt.Errorf("bulk upsert neighborhoods: %w", err)
		}
		for _, row := range rows {
			key := util.NormalizeString(row.PricesNeighborhoodName)
			neighborhoodIDs[key] = row.PricesNeighborhoodID
		}
	}
	for neighborhoodName, postalCode := range allNeighborhoods {
		postalCodeID, ok := postalCodeMap[postalCode]
		if !ok {
			continue
		}
		pcIDCopy := postalCodeID
		_ = s.queries.UpdateNeighborhoodPostalCode(ctx, db.UpdateNeighborhoodPostalCodeParams{
			PostalCodeID: &pcIDCopy,
			Name:         neighborhoodName,
			CityID:       cityID,
		})
	}
	result.transactions = len(allTransactions)
	if len(allTransactions) > 0 {
		periodIdentifier := s.nowFunc().Format("2006-01")
		params, err := mapUpsertTransactionsBulkParams(allTransactions, neighborhoodIDs, periodIdentifier)
		if err != nil {
			return nil, fmt.Errorf("build transaction params: %w", err)
		}
		if _, err := s.queries.UpsertPricesTransactionsBulk(ctx, *params); err != nil {
			return nil, fmt.Errorf("bulk upsert transactions: %w", err)
		}
	}
	return result, nil
}

type SyncNeighborhoodPostalCodesProgress struct {
	City       string
	PostalCode string
	Page       int
	Updated    int
}

type SyncCityProgress struct {
	City    string
	Step    string
	Page    int
	Count   int
	Details string
}

func (s *Service) SyncNeighborhoodPostalCodes(ctx context.Context, progressFn func(SyncNeighborhoodPostalCodesProgress)) error {
	cities, err := s.queries.ListPricesCities(ctx)
	if err != nil {
		return fmt.Errorf("list cities: %w", err)
	}
	for _, city := range cities {
		postalCodes, err := s.queries.ListPricesPostalCodesByCity(ctx, city.PricesCityID)
		if err != nil {
			return fmt.Errorf("list postal codes for city %q: %w", city.PricesCityName, err)
		}
		for _, pc := range postalCodes {
			updated, err := s.syncNeighborhoodsForPostalCode(ctx, city, pc, progressFn)
			if err != nil {
				return fmt.Errorf("sync neighborhoods for postal code %q in %q: %w", pc.PricesPostalCodeCode, city.PricesCityName, err)
			}
			if progressFn != nil {
				progressFn(SyncNeighborhoodPostalCodesProgress{
					City:       city.PricesCityName,
					PostalCode: pc.PricesPostalCodeCode,
					Updated:    updated,
				})
			}
		}
	}
	return nil
}

func (s *Service) syncNeighborhoodsForPostalCode(
	ctx context.Context,
	city db.PricesCity,
	pc db.PricesPostalCode,
	progressFn func(SyncNeighborhoodPostalCodesProgress),
) (int, error) {
	neighborhoodNames := make(map[string]struct{})
	nextPage := new(int)
	*nextPage = 0
	page := 0
	for nextPage != nil {
		page = *nextPage
		if progressFn != nil {
			progressFn(SyncNeighborhoodPostalCodesProgress{
				City:       city.PricesCityName,
				PostalCode: pc.PricesPostalCodeCode,
				Page:       page,
			})
		}
		response, err := s.client.GetTransactionsForPage(ctx, &client.ApartmentSearchParams{
			City:        city.PricesCityName,
			PostalCodes: []string{pc.PricesPostalCodeCode},
			RenderType:  "renderTypeTable",
		}, page)
		if err != nil {
			return 0, fmt.Errorf("fetch page %d: %w", page, err)
		}
		for _, tx := range response.Apartments {
			name := util.TrimUnicodeSpace(tx.Neighborhood)
			if name != "" {
				neighborhoodNames[name] = struct{}{}
			}
		}
		nextPage = response.NextPage
	}
	updated := 0
	for name := range neighborhoodNames {
		pcID := pc.PricesPostalCodeID
		err := s.queries.UpdateNeighborhoodPostalCode(ctx, db.UpdateNeighborhoodPostalCodeParams{
			PostalCodeID: &pcID,
			Name:         name,
			CityID:       city.PricesCityID,
		})
		if err != nil {
			return updated, fmt.Errorf("update neighborhood %q: %w", name, err)
		}
		updated++
	}
	return updated, nil
}

func (s *Service) SyncCity(ctx context.Context, cityName string) error {
	return s.SyncCityWithProgress(ctx, cityName, nil)
}

func (s *Service) SyncCityWithProgress(ctx context.Context, cityName string, progressFn func(SyncCityProgress)) error {
	report := func(p SyncCityProgress) {
		if progressFn != nil {
			progressFn(p)
		}
	}
	report(SyncCityProgress{City: cityName, Step: "city_upsert_start"})
	cityRow, err := s.queries.UpsertPricesCity(ctx, mapUpsertCityParams(cityName))
	if err != nil {
		return fmt.Errorf("upsert city %q: %w", cityName, err)
	}
	report(SyncCityProgress{City: cityName, Step: "city_upsert_done"})
	cityID := cityRow.PricesCityID
	report(SyncCityProgress{City: cityName, Step: "postal_codes_fetch_start"})
	postalCodes, err := s.client.FetchPostalCodes(ctx, cityName)
	if err != nil {
		return fmt.Errorf("fetch postal codes for %q: %w", cityName, err)
	}
	postalCodes = util.UniqueStrings(postalCodes)
	report(SyncCityProgress{City: cityName, Step: "postal_codes_fetch_done", Count: len(postalCodes)})
	postalCodeIDs := make(map[string]uuid.UUID, len(postalCodes))
	if len(postalCodes) > 0 {
		report(SyncCityProgress{City: cityName, Step: "postal_codes_upsert_start", Count: len(postalCodes)})
		pcParams := mapUpsertPostalCodesBulkParams(postalCodes, cityID)
		rows, err := s.queries.UpsertPricesPostalCodesBulk(ctx, pcParams)
		if err != nil {
			return fmt.Errorf("bulk upsert postal codes for %q: %w", cityName, err)
		}
		for _, row := range rows {
			postalCodeIDs[row.PricesPostalCodeCode] = row.PricesPostalCodeID
		}
		report(SyncCityProgress{City: cityName, Step: "postal_codes_upsert_done", Count: len(rows)})
	}
	report(SyncCityProgress{City: cityName, Step: "neighborhoods_fetch_start"})
	neighborhoods, err := s.client.FetchNeighborhoods(ctx, cityName)
	if err != nil {
		return fmt.Errorf("fetch neighborhoods for %q: %w", cityName, err)
	}
	neighborhoods = util.UniqueStrings(neighborhoods)
	report(SyncCityProgress{City: cityName, Step: "neighborhoods_fetch_done", Count: len(neighborhoods)})
	report(SyncCityProgress{City: cityName, Step: "transactions_fetch_start"})
	transactions, err := s.client.GetAllTransactionsWithProgress(ctx, cityName, func(p client.TransactionsProgress) {
		report(SyncCityProgress{
			City:    cityName,
			Step:    "transactions_page",
			Page:    p.Page,
			Count:   p.Apartments,
			Details: fmt.Sprintf("total=%d", p.TotalApartments),
		})
	})
	if err != nil {
		return fmt.Errorf("fetch transactions for %q: %w", cityName, err)
	}
	report(SyncCityProgress{City: cityName, Step: "transactions_fetch_done", Count: len(transactions)})
	transactionNeighborhoods := make(map[string]bool)
	for _, tx := range transactions {
		normalized := util.TrimUnicodeSpace(tx.Neighborhood)
		if normalized != "" {
			transactionNeighborhoods[normalized] = true
		}
	}
	for name := range transactionNeighborhoods {
		neighborhoods = append(neighborhoods, name)
	}
	neighborhoods = util.UniqueStrings(neighborhoods)
	report(SyncCityProgress{City: cityName, Step: "neighborhoods_merge_done", Count: len(neighborhoods)})
	neighborhoodIDs := make(map[string]uuid.UUID, len(neighborhoods))
	if len(neighborhoods) > 0 {
		report(SyncCityProgress{City: cityName, Step: "neighborhoods_upsert_start", Count: len(neighborhoods)})
		nbParams := mapUpsertNeighborhoodsBulkParams(neighborhoods, cityID)
		rows, err := s.queries.UpsertPricesNeighborhoodsBulk(ctx, nbParams)
		if err != nil {
			return fmt.Errorf("bulk upsert neighborhoods for %q: %w", cityName, err)
		}
		for _, row := range rows {
			key := util.NormalizeString(row.PricesNeighborhoodName)
			neighborhoodIDs[key] = row.PricesNeighborhoodID
		}
		report(SyncCityProgress{City: cityName, Step: "neighborhoods_upsert_done", Count: len(rows)})
	}
	if len(transactions) > 0 {
		periodIdentifier := s.nowFunc().Format("2006-01")
		report(SyncCityProgress{City: cityName, Step: "transactions_upsert_start", Count: len(transactions), Details: periodIdentifier})
		params, err := mapUpsertTransactionsBulkParams(transactions, neighborhoodIDs, periodIdentifier)
		if err != nil {
			return fmt.Errorf("build transaction params for %q: %w", cityName, err)
		}
		if _, err := s.queries.UpsertPricesTransactionsBulk(ctx, *params); err != nil {
			return fmt.Errorf("bulk upsert transactions for %q: %w", cityName, err)
		}
		report(SyncCityProgress{City: cityName, Step: "transactions_upsert_done", Count: len(transactions), Details: periodIdentifier})
	}
	report(SyncCityProgress{City: cityName, Step: "sync_city_done"})
	return nil
}

func parseElevator(val string) (bool, error) {
	val = util.TrimUnicodeSpace(val)
	val = strings.ToLower(val)
	switch val {
	case "on":
		return true, nil
	case "ei":
		return false, nil
	default:
		return false, fmt.Errorf("invalid elevator value: %q", val)
	}
}

type LLMMatchResult struct {
	PostalCodeID   *uuid.UUID
	PostalCodeName string
	Confidence     string
	Reasoning      string
}

type MatchNeighborhoodsProgress struct {
	Processed       int
	Matched         int
	Unmatched       int
	TotalRemaining  int
	CurrentCity     string
	CurrentNeighbor string
}

func (s *Service) MatchNeighborhoodsWithLLM(
	ctx context.Context,
	model string,
	batchSize int,
	progressFn func(MatchNeighborhoodsProgress),
) error {
	if s.openRouterCli == nil {
		return fmt.Errorf("OpenRouter client not configured")
	}
	if model == "" {
		model = "google/gemini-2.0-flash-exp"
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	countRow, err := s.queries.CountUnmatchedNeighborhoods(ctx)
	if err != nil {
		return fmt.Errorf("count unmatched neighborhoods: %w", err)
	}
	totalRemaining := int(countRow)
	processed := 0
	matched := 0
	offset := int32(0)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		neighborhoods, err := s.queries.ListUnmatchedNeighborhoodsBatch(ctx, offset)
		if err != nil {
			return fmt.Errorf("list unmatched neighborhoods: %w", err)
		}
		if len(neighborhoods) == 0 {
			break
		}
		for _, neighborhood := range neighborhoods {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			cityName := ""
			if neighborhood.PricesCityName != nil {
				cityName = *neighborhood.PricesCityName
			}
			if progressFn != nil {
				progressFn(MatchNeighborhoodsProgress{
					Processed:       processed,
					Matched:         matched,
					Unmatched:       processed - matched,
					TotalRemaining:  totalRemaining,
					CurrentCity:     cityName,
					CurrentNeighbor: neighborhood.PricesNeighborhoodName,
				})
			}
			result, err := s.matchNeighborhoodWithLLM(ctx, model, neighborhood, cityName)
			if err != nil {
				return fmt.Errorf("match neighborhood %q in %q: %w", neighborhood.PricesNeighborhoodName, cityName, err)
			}
			if result.PostalCodeID != nil {
				if err := s.queries.UpdateNeighborhoodPostiPostalCode(ctx, db.UpdateNeighborhoodPostiPostalCodeParams{
					PostalCodeID:   result.PostalCodeID,
					NeighborhoodID: neighborhood.PricesNeighborhoodID,
				}); err != nil {
					return fmt.Errorf("update neighborhood postal code: %w", err)
				}
				matched++
			}
			processed++
			time.Sleep(500 * time.Millisecond)
		}
		if len(neighborhoods) < batchSize {
			break
		}
		offset += int32(batchSize)
	}
	if progressFn != nil {
		progressFn(MatchNeighborhoodsProgress{
			Processed:      processed,
			Matched:        matched,
			Unmatched:      processed - matched,
			TotalRemaining: 0,
		})
	}
	return nil
}

func (s *Service) matchNeighborhoodWithLLM(
	ctx context.Context,
	model string,
	neighborhood db.ListUnmatchedNeighborhoodsBatchRow,
	cityName string,
) (*LLMMatchResult, error) {
	postalCodes, err := s.queries.GetAvailablePostalCodesForMunicipality(ctx, cityName)
	if err != nil {
		return nil, fmt.Errorf("get available postal codes: %w", err)
	}
	if len(postalCodes) == 0 {
		return &LLMMatchResult{
			PostalCodeID: nil,
			Confidence:   "none",
			Reasoning:    "No postal codes available for municipality",
		}, nil
	}
	postalCodesJSON, err := json.Marshal(postalCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal postal codes: %w", err)
	}
	prompt := fmt.Sprintf(`You are a Finnish postal code matching assistant. Your task is to match a neighborhood name to the most appropriate postal code area.

Neighborhood: %s
Municipality: %s

Available postal codes in this municipality:
%s

Instructions:
1. Analyze the neighborhood name and find the best matching postal code area
2. Consider that neighborhood names might be informal, compound (comma-separated), or partial matches
3. If no reasonable match exists, return null for postal_code_id
4. Respond ONLY with valid JSON in this exact format:

{
  "postal_code_id": "uuid-here-or-null",
  "postal_code_name": "name-here-or-empty",
  "confidence": "high|medium|low|none",
  "reasoning": "brief explanation"
}

Do not include any text outside the JSON structure.`, neighborhood.PricesNeighborhoodName, cityName, string(postalCodesJSON))

	temperature := float32(0.1)
	resp, err := s.openRouterCli.CreateChatCompletion(ctx, openrouter.ChatCompletionRequest{
		Model: model,
		Messages: []openrouter.ChatCompletionMessage{
			openrouter.UserMessage(prompt),
		},
		Temperature: temperature,
		MaxTokens:   500,
	})
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	content := resp.Choices[0].Message.Content.Text
	content = strings.TrimSpace(content)
	if after, ok := strings.CutPrefix(content, "```json"); ok {
		content = after
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if after, ok := strings.CutPrefix(content, "```"); ok {
		content = after
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var result struct {
		PostalCodeID   *string `json:"postal_code_id"`
		PostalCodeName string  `json:"postal_code_name"`
		Confidence     string  `json:"confidence"`
		Reasoning      string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal LLM response: %w (content: %s)", err, content)
	}
	llmResult := &LLMMatchResult{
		PostalCodeName: result.PostalCodeName,
		Confidence:     result.Confidence,
		Reasoning:      result.Reasoning,
	}
	if result.PostalCodeID != nil && *result.PostalCodeID != "" && *result.PostalCodeID != "null" {
		parsed, parseErr := uuid.Parse(*result.PostalCodeID)
		if parseErr != nil {
			return nil, fmt.Errorf("parse postal code UUID: %w", parseErr)
		}
		llmResult.PostalCodeID = &parsed
	}
	return llmResult, nil
}
