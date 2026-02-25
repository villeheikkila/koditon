package postal

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/db"
	"koditon-go/internal/postal/client"
)

func parseDate(s string) *time.Time {
	if len(s) != 8 {
		return nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return nil
	}
	return &t
}

type adAreaKey struct {
	Code   string
	NameFi string
	NameSv string
}

type municipalityKey struct {
	Code              string
	NameFi            string
	NameSv            string
	LanguageRatioCode string
}

func extractAdAreas(records []*client.PostalCodeRecord) *db.UpsertPostalAdAreasBulkParams {
	seen := make(map[string]adAreaKey)
	for _, r := range records {
		if r.AdAreaCode == "" {
			continue
		}
		if _, exists := seen[r.AdAreaCode]; !exists {
			seen[r.AdAreaCode] = adAreaKey{
				Code:   r.AdAreaCode,
				NameFi: r.AdAreaFi,
				NameSv: r.AdAreaSv,
			}
		}
	}
	params := &db.UpsertPostalAdAreasBulkParams{
		Codes:   make([]string, 0, len(seen)),
		NamesFi: make([]string, 0, len(seen)),
		NamesSv: make([]string, 0, len(seen)),
	}
	for _, v := range seen {
		params.Codes = append(params.Codes, v.Code)
		params.NamesFi = append(params.NamesFi, v.NameFi)
		params.NamesSv = append(params.NamesSv, v.NameSv)
	}
	return params
}

func extractMunicipalities(records []*client.PostalCodeRecord) *db.UpsertPostalMunicipalitiesBulkParams {
	seen := make(map[string]municipalityKey)
	for _, r := range records {
		if r.MunicipalCode == "" {
			continue
		}
		if _, exists := seen[r.MunicipalCode]; !exists {
			seen[r.MunicipalCode] = municipalityKey{
				Code:              r.MunicipalCode,
				NameFi:            r.MunicipalNameFi,
				NameSv:            r.MunicipalNameSv,
				LanguageRatioCode: r.MunicipalLanguageRatioCode,
			}
		}
	}
	params := &db.UpsertPostalMunicipalitiesBulkParams{
		Codes:              make([]string, 0, len(seen)),
		NamesFi:            make([]string, 0, len(seen)),
		NamesSv:            make([]string, 0, len(seen)),
		LanguageRatioCodes: make([]string, 0, len(seen)),
	}
	for _, v := range seen {
		params.Codes = append(params.Codes, v.Code)
		params.NamesFi = append(params.NamesFi, v.NameFi)
		params.NamesSv = append(params.NamesSv, v.NameSv)
		params.LanguageRatioCodes = append(params.LanguageRatioCodes, v.LanguageRatioCode)
	}
	return params
}

func mapUpsertPostalCodesBulkParams(
	records []*client.PostalCodeRecord,
	adAreaIDs map[string]pgtype.UUID,
	municipalityIDs map[string]pgtype.UUID,
) *db.UpsertPostalPostalCodesBulkParams {
	params := &db.UpsertPostalPostalCodesBulkParams{
		Dates:           make([]time.Time, 0, len(records)),
		Codes:           make([]string, 0, len(records)),
		NamesFi:         make([]string, 0, len(records)),
		NamesSv:         make([]string, 0, len(records)),
		AbbrsFi:         make([]string, 0, len(records)),
		AbbrsSv:         make([]string, 0, len(records)),
		NeighborhoodsFi: make([]string, 0, len(records)),
		ValidsFrom:      make([]time.Time, 0, len(records)),
		TypeCodes:       make([]string, 0, len(records)),
		AdAreaIds:       make([]pgtype.UUID, 0, len(records)),
		MunicipalityIds: make([]pgtype.UUID, 0, len(records)),
	}
	for _, r := range records {
		date := parseDate(r.Date)
		if date == nil {
			continue
		}
		validFrom := parseDate(r.ValidFrom)
		if validFrom == nil {
			validFrom = date
		}
		var adAreaID pgtype.UUID
		if id, ok := adAreaIDs[r.AdAreaCode]; ok {
			adAreaID = id
		}
		var municipalityID pgtype.UUID
		if id, ok := municipalityIDs[r.MunicipalCode]; ok {
			municipalityID = id
		}
		neighborhood := GetNeighborhood(r.Postcode)
		params.Dates = append(params.Dates, *date)
		params.Codes = append(params.Codes, r.Postcode)
		params.NamesFi = append(params.NamesFi, r.PostcodeNameFi)
		params.NamesSv = append(params.NamesSv, r.PostcodeNameSv)
		params.AbbrsFi = append(params.AbbrsFi, r.PostcodeAbbrFi)
		params.AbbrsSv = append(params.AbbrsSv, r.PostcodeAbbrSv)
		params.NeighborhoodsFi = append(params.NeighborhoodsFi, neighborhood)
		params.ValidsFrom = append(params.ValidsFrom, *validFrom)
		params.TypeCodes = append(params.TypeCodes, r.TypeCode)
		params.AdAreaIds = append(params.AdAreaIds, adAreaID)
		params.MunicipalityIds = append(params.MunicipalityIds, municipalityID)
	}
	return params
}
