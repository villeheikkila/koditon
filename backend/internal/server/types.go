package server

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type TransactionNeighborhood struct {
	Id         openapi_types.UUID `json:"id"`
	Name       string             `json:"name"`
	PostalCode *string            `json:"postal_code"`
}

type HintatiedotTransaction struct {
	Area                float64                 `json:"area"`
	BuildYear           int32                   `json:"build_year"`
	Category            string                  `json:"category"`
	Condition           string                  `json:"condition"`
	Description         string                  `json:"description"`
	Elevator            bool                    `json:"elevator"`
	EnergyClass         *string                 `json:"energy_class"`
	FirstSeenAt         time.Time               `json:"first_seen_at"`
	Floor               string                  `json:"floor"`
	Id                  openapi_types.UUID      `json:"id"`
	LastSeenAt          time.Time               `json:"last_seen_at"`
	Neighborhood        TransactionNeighborhood `json:"neighborhood"`
	Plot                string                  `json:"plot"`
	Price               int32                   `json:"price"`
	PricePerSquareMeter int32                   `json:"price_per_square_meter"`
	Type                string                  `json:"type"`
}

type Neighborhood struct {
	Id         openapi_types.UUID `json:"id"`
	Name       string             `json:"name"`
	PostalCode *string            `json:"postal_code"`
}

type City struct {
	Id            openapi_types.UUID `json:"id"`
	Name          string             `json:"name"`
	Neighborhoods []Neighborhood     `json:"neighborhoods"`
}

type SyncHintatiedotRequest struct {
	City string `json:"city"`
}

type SyncHintatiedotResponse struct {
	Status string `json:"status"`
	City   string `json:"city"`
}
