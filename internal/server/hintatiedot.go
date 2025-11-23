package server

import (
	"koditon-go/internal/db"
	"koditon-go/internal/util"
	"time"
)

type HintatiedotResponse struct {
	ID                  string                      `json:"id"`
	Neighborhood        TransactionNeighborhoodInfo `json:"neighborhood"`
	Description         string                      `json:"description"`
	Type                string                      `json:"type"`
	Area                float64                     `json:"area"`
	Price               int32                       `json:"price"`
	PricePerSquareMeter int32                       `json:"price_per_square_meter"`
	BuildYear           int32                       `json:"build_year"`
	Floor               string                      `json:"floor"`
	Elevator            string                      `json:"elevator"`
	Condition           string                      `json:"condition"`
	Plot                string                      `json:"plot"`
	EnergyClass         util.Nullable[string]       `json:"energy_class,omitzero"`
	FirstSeenAt         time.Time                   `json:"first_seen_at"`
	LastSeenAt          time.Time                   `json:"last_seen_at"`
	Category            string                      `json:"category"`
}

type TransactionNeighborhoodInfo struct {
	Code       string                `json:"code"`
	PostalCode util.Nullable[string] `json:"postal_code,omitzero"`
	Name       string                `json:"name"`
}

type CityResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Neighborhoods []NeighborhoodResponse `json:"neighborhoods"`
}

type NeighborhoodResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	PostalCode util.Nullable[string] `json:"postal_code,omitzero"`
}

func mapTransactionResponse(row db.ListTransactionsByNeighborhoodsRow) HintatiedotResponse {
	var neighborhoodName string
	if row.HintatiedotNeighborhoodsName.Valid {
		neighborhoodName = row.HintatiedotNeighborhoodsName.String
	}

	return HintatiedotResponse{
		ID: row.HintatiedotTransactionsID.String(),
		Neighborhood: TransactionNeighborhoodInfo{
			Code:       row.HintatiedotNeighborhoodsID.String(),
			PostalCode: util.FromPgText(row.HintatiedotPostalCodesCode),
			Name:       neighborhoodName,
		},
		Description:         row.HintatiedotTransactionsDescription,
		Type:                row.HintatiedotTransactionsType,
		Area:                row.HintatiedotTransactionsArea,
		Price:               row.HintatiedotTransactionsPrice,
		PricePerSquareMeter: row.HintatiedotTransactionsPricePerSquareMeter,
		BuildYear:           row.HintatiedotTransactionsBuildYear,
		Floor:               row.HintatiedotTransactionsFloor.String,
		Elevator:            formatBool(row.HintatiedotTransactionsElevator),
		Condition:           row.HintatiedotTransactionsCondition.String,
		Plot:                row.HintatiedotTransactionsPlot.String,
		EnergyClass:         util.FromPgText(row.HintatiedotTransactionsEnergyClass),
		FirstSeenAt:         row.CreatedAt.Time,
		LastSeenAt:          row.UpdatedAt.Time,
		Category:            row.HintatiedotTransactionsCategory,
	}
}

func mapCitiesWithNeighborhoods(rows []db.ListCitiesWithNeighborhoodsRow) []CityResponse {
	cityMap := make(map[string]*CityResponse)
	var cityOrder []string

	for _, row := range rows {
		cityID := row.HintatiedotCitiesID.String()

		if _, exists := cityMap[cityID]; !exists {
			cityMap[cityID] = &CityResponse{
				ID:            cityID,
				Name:          row.HintatiedotCitiesName,
				Neighborhoods: []NeighborhoodResponse{},
			}
			cityOrder = append(cityOrder, cityID)
		}

		if row.HintatiedotNeighborhoodsID.Valid {
			cityMap[cityID].Neighborhoods = append(cityMap[cityID].Neighborhoods, NeighborhoodResponse{
				ID:         row.HintatiedotNeighborhoodsID.String(),
				Name:       row.HintatiedotNeighborhoodsName.String,
				PostalCode: util.FromPgText(row.HintatiedotPostalCodesCode),
			})
		}
	}

	result := make([]CityResponse, 0, len(cityOrder))
	for _, cityID := range cityOrder {
		result = append(result, *cityMap[cityID])
	}
	return result
}

func formatBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
