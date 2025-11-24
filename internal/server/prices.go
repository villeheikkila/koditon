package server

import (
	"koditon-go/internal/db"
	"koditon-go/internal/util"
	"time"
)

type PricesResponse struct {
	ID                  string                      `json:"id"`
	Neighborhood        TransactionNeighborhoodInfo `json:"neighborhood"`
	Description         string                      `json:"description"`
	Type                string                      `json:"type"`
	Area                float64                     `json:"area"`
	Price               int32                       `json:"price"`
	PricePerSquareMeter int32                       `json:"price_per_square_meter"`
	BuildYear           int32                       `json:"build_year"`
	Floor               string                      `json:"floor"`
	Elevator            bool                        `json:"elevator"`
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

func mapTransactionResponse(row db.ListTransactionsByNeighborhoodsRow) PricesResponse {
	var neighborhoodName string
	if row.PricesNeighborhoodsName.Valid {
		neighborhoodName = row.PricesNeighborhoodsName.String
	}
	return PricesResponse{
		ID: row.PricesTransactionsID.String(),
		Neighborhood: TransactionNeighborhoodInfo{
			Code:       row.PricesNeighborhoodsID.String(),
			PostalCode: util.FromPgText(row.PricesPostalCodesCode),
			Name:       neighborhoodName,
		},
		Description:         row.PricesTransactionsDescription,
		Type:                row.PricesTransactionsType,
		Area:                row.PricesTransactionsArea,
		Price:               row.PricesTransactionsPrice,
		PricePerSquareMeter: row.PricesTransactionsPricePerSquareMeter,
		BuildYear:           row.PricesTransactionsBuildYear,
		Floor:               row.PricesTransactionsFloor.String,
		Elevator:            row.PricesTransactionsElevator,
		Condition:           row.PricesTransactionsCondition.String,
		Plot:                row.PricesTransactionsPlot.String,
		EnergyClass:         util.FromPgText(row.PricesTransactionsEnergyClass),
		FirstSeenAt:         row.CreatedAt.Time,
		LastSeenAt:          row.UpdatedAt.Time,
		Category:            row.PricesTransactionsCategory,
	}
}

func mapCitiesWithNeighborhoods(rows []db.ListCitiesWithNeighborhoodsRow) []CityResponse {
	cityMap := make(map[string]*CityResponse)
	var cityOrder []string
	for _, row := range rows {
		cityID := row.PricesCitiesID.String()

		if _, exists := cityMap[cityID]; !exists {
			cityMap[cityID] = &CityResponse{
				ID:            cityID,
				Name:          row.PricesCitiesName,
				Neighborhoods: []NeighborhoodResponse{},
			}
			cityOrder = append(cityOrder, cityID)
		}
		if row.PricesNeighborhoodsID.Valid {
			cityMap[cityID].Neighborhoods = append(cityMap[cityID].Neighborhoods, NeighborhoodResponse{
				ID:         row.PricesNeighborhoodsID.String(),
				Name:       row.PricesNeighborhoodsName.String,
				PostalCode: util.FromPgText(row.PricesPostalCodesCode),
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
