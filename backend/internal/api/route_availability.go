package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type availableMunicipality struct {
	ID     string  `json:"id"`
	Code   string  `json:"code"`
	NameFi string  `json:"name_fi"`
	NameSv *string `json:"name_sv,omitempty"`
}

type availablePostalCode struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	NameFi         string  `json:"name_fi"`
	NameSv         *string `json:"name_sv,omitempty"`
	MunicipalityID string  `json:"municipality_id"`
}

type availableLocationsOutput struct {
	Body struct {
		Municipalities []availableMunicipality `json:"municipalities"`
		PostalCodes    []availablePostalCode   `json:"postal_codes"`
	}
}

func (a *API) availableLocationsHandler(ctx context.Context, _ *struct{}) (*availableLocationsOutput, error) {
	municipalityRows, err := a.pricesQueries.ListAvailableMunicipalities(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("internal server error")
	}
	municipalities := make([]availableMunicipality, 0, len(municipalityRows))
	for _, row := range municipalityRows {
		municipalities = append(municipalities, availableMunicipality{
			ID:     formatUUID(row.PostalMunicipalityID),
			Code:   row.PostalMunicipalityCode,
			NameFi: row.PostalMunicipalityNameFi,
			NameSv: row.PostalMunicipalityNameSv,
		})
	}
	postalCodeRows, err := a.pricesQueries.ListAvailablePostalCodes(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("internal server error")
	}
	postalCodes := make([]availablePostalCode, 0, len(postalCodeRows))
	for _, row := range postalCodeRows {
		postalCodes = append(postalCodes, availablePostalCode{
			ID:             formatUUID(row.PostalPostalCodeID),
			Code:           row.PostalPostalCodeCode,
			NameFi:         row.PostalPostalCodeNameFi,
			NameSv:         row.PostalPostalCodeNameSv,
			MunicipalityID: formatUUID(row.PostalMunicipalityID),
		})
	}
	output := &availableLocationsOutput{}
	output.Body.Municipalities = municipalities
	output.Body.PostalCodes = postalCodes
	return output, nil
}

var categoryTranslations = map[string]string{
	"Yksiöt":                     "Studios",
	"Kaksiot":                    "2 Rooms",
	"Kolmiot":                    "3 Rooms",
	"Neljä huonetta tai enemmän": "4+ Rooms",
}

var typeTranslations = map[string]string{
	"kt": "Apartment Building",
	"rt": "Row House",
	"ok": "Detached House",
}

var plotTranslations = map[string]string{
	"oma":    "Owned",
	"vuokra": "Rental",
}

type translatedValue struct {
	Value       string `json:"value"`
	Translation string `json:"translation"`
}

type availableCategoriesOutput struct {
	Body struct {
		Categories []translatedValue `json:"categories"`
	}
}

func translateCategory(finnishValue string) string {
	if translation, ok := categoryTranslations[finnishValue]; ok {
		return translation
	}
	return finnishValue
}

func translateType(finnishValue string) string {
	if translation, ok := typeTranslations[finnishValue]; ok {
		return translation
	}
	return finnishValue
}

func translatePlot(finnishValue string) string {
	if translation, ok := plotTranslations[finnishValue]; ok {
		return translation
	}
	return finnishValue
}

func (a *API) availableCategoriesHandler(ctx context.Context, _ *struct{}) (*availableCategoriesOutput, error) {
	rows, err := a.pricesQueries.ListDistinctCategories(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("internal server error")
	}
	categories := make([]translatedValue, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, translatedValue{
			Value:       row,
			Translation: translateCategory(row),
		})
	}
	output := &availableCategoriesOutput{}
	output.Body.Categories = categories
	return output, nil
}

type availableTypesOutput struct {
	Body struct {
		Types []translatedValue `json:"types"`
	}
}

func (a *API) availableTypesHandler(ctx context.Context, _ *struct{}) (*availableTypesOutput, error) {
	rows, err := a.pricesQueries.ListDistinctTypes(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("internal server error")
	}
	types := make([]translatedValue, 0, len(rows))
	for _, row := range rows {
		types = append(types, translatedValue{
			Value:       row,
			Translation: translateType(row),
		})
	}
	output := &availableTypesOutput{}
	output.Body.Types = types
	return output, nil
}

type availablePlotsOutput struct {
	Body struct {
		Plots []translatedValue `json:"plots"`
	}
}

func (a *API) availablePlotsHandler(ctx context.Context, _ *struct{}) (*availablePlotsOutput, error) {
	rows, err := a.pricesQueries.ListDistinctPlots(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("internal server error")
	}
	plots := make([]translatedValue, 0, len(rows))
	for _, row := range rows {
		if row == nil || *row == "" {
			continue
		}
		plots = append(plots, translatedValue{
			Value:       *row,
			Translation: translatePlot(*row),
		})
	}
	output := &availablePlotsOutput{}
	output.Body.Plots = plots
	return output, nil
}
