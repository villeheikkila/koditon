package server

import (
	"context"

	"github.com/google/uuid"
)

type postalCity struct {
	ID          string       `json:"id"`
	Code        string       `json:"code"`
	NameFi      string       `json:"name_fi"`
	NameSv      *string      `json:"name_sv,omitempty"`
	PostalCodes []postalCode `json:"postal_codes"`
}

type postalCode struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	NameFi         string  `json:"name_fi"`
	NameSv         *string `json:"name_sv,omitempty"`
	NeighborhoodFi *string `json:"neighborhood_fi,omitempty"`
}

type postalCitiesOutput struct {
	Body struct {
		Cities []postalCity `json:"cities"`
	}
}

func (s *Server) postalCitiesHandler(ctx context.Context, _ *struct{}) (*postalCitiesOutput, error) {
	rows, err := s.postalQueries.ListMunicipalitiesWithPostalCodes(ctx)
	if err != nil {
		return nil, err
	}
	cityIndex := make(map[string]int)
	cities := make([]postalCity, 0, len(rows))
	for _, row := range rows {
		cityID := formatUUID(row.PostalMunicipalityID)
		index, exists := cityIndex[cityID]
		if !exists {
			cityIndex[cityID] = len(cities)
			cities = append(cities, postalCity{
				ID:          cityID,
				Code:        row.PostalMunicipalityCode,
				NameFi:      row.PostalMunicipalityNameFi,
				NameSv:      row.PostalMunicipalityNameSv,
				PostalCodes: []postalCode{},
			})
			index = cityIndex[cityID]
		}
		cities[index].PostalCodes = append(cities[index].PostalCodes, postalCode{
			ID:             formatUUID(row.PostalPostalCodeID),
			Code:           row.PostalPostalCodeCode,
			NameFi:         row.PostalPostalCodeNameFi,
			NameSv:         row.PostalPostalCodeNameSv,
			NeighborhoodFi: row.PostalPostalCodeNeighborhoodFi,
		})
	}
	output := &postalCitiesOutput{}
	output.Body.Cities = cities
	return output, nil
}

func formatUUID(id uuid.UUID) string {
	return id.String()
}
