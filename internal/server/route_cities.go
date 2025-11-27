package server

import (
	"context"
	"koditon-go/internal/db"

	"github.com/danielgtaylor/huma/v2"
)

type listCitiesOutput struct {
	Body []City
}

func (s *Server) listCitiesHandler(ctx context.Context, _ *struct{}) (*listCitiesOutput, error) {
	cities, err := s.db.ListCitiesWithNeighborhoods(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "list cities with neighborhoods", "err", err)
		return nil, huma.Error500InternalServerError("Internal server error")
	}
	return &listCitiesOutput{Body: mapCitiesWithNeighborhoods(cities)}, nil
}

func mapCitiesWithNeighborhoods(rows []db.ListCitiesWithNeighborhoodsRow) []City {
	cityMap := make(map[string]*City)
	var cityOrder []string
	for _, row := range rows {
		cityUUID, ok := uuidFromPg(row.PricesCitiesID)
		if !ok {
			continue
		}
		cityID := cityUUID.String()
		if _, exists := cityMap[cityID]; !exists {
			cityMap[cityID] = &City{
				Id:            cityUUID,
				Name:          row.PricesCitiesName,
				Neighborhoods: []Neighborhood{},
			}
			cityOrder = append(cityOrder, cityID)
		}
		if row.PricesNeighborhoodsID.Valid {
			neighborhoodID, ok := uuidFromPg(row.PricesNeighborhoodsID)
			if !ok {
				continue
			}
			cityMap[cityID].Neighborhoods = append(cityMap[cityID].Neighborhoods, Neighborhood{
				Id:         neighborhoodID,
				Name:       row.PricesNeighborhoodsName.String,
				PostalCode: stringPtrFromPg(row.PricesPostalCodesCode),
			})
		}
	}
	result := make([]City, 0, len(cityOrder))
	for _, cityID := range cityOrder {
		result = append(result, *cityMap[cityID])
	}
	return result
}
