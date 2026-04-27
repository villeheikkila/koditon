package client

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type LocationResponse struct {
	Card struct {
		Name        string       `json:"name"`
		CardID      int          `json:"cardId"`
		CardType    int          `json:"cardType"`
		Coordinates *Coordinates `json:"coordinates,omitempty"`
	} `json:"card"`
	Parent struct {
		Name        string       `json:"name"`
		CardID      int          `json:"cardId"`
		CardType    int          `json:"cardType"`
		Coordinates *Coordinates `json:"coordinates,omitempty"`
	} `json:"parent"`
}

func (l LocationResponse) LocationString() string {
	return fmt.Sprintf("[[%d, %d, %s]]", l.Card.CardID, l.Card.CardType, strconv.Quote(l.Card.Name))
}

type Coordinates struct {
	Latitude  float64 `json:"latitude,string,omitempty"`
	Longitude float64 `json:"longitude,string,omitempty"`
}

type BuildingData struct {
	Address      *string `json:"address"`
	District     *string `json:"district"`
	City         *string `json:"city"`
	Country      *string `json:"country"`
	Year         *int    `json:"year"`
	BuildingType *int    `json:"buildingType"`
}

type Card struct {
	ID                int             `json:"id"`
	URL               string          `json:"url"`
	Description       *string         `json:"description"`
	Rooms             *int            `json:"rooms"`
	RoomConfiguration *string         `json:"roomConfiguration"`
	Price             *NumberOrString `json:"price"`
	NewDevelopment    *bool           `json:"newDevelopment"`
	Published         *string         `json:"published"`
	Size              *float64        `json:"size"`
	Coordinates       *Coordinates    `json:"coordinates"`
	BuildingData      *BuildingData   `json:"buildingData"`
}

type SearchResult struct {
	Cards []Card `json:"cards"`
	Found int    `json:"found"`
	Start int    `json:"start"`
}

type BuildingResponse struct {
	BuildingID       int          `json:"buildingId"`
	BuildingClass    *int         `json:"buildingClass"`
	PrimaryAddressID *int         `json:"primaryAddressId"`
	BuildYear        *int         `json:"buildYear"`
	BuildingType     *int         `json:"buildingType"`
	LocationType     *int         `json:"locationType"`
	Floors           *int         `json:"floors"`
	Lift             *bool        `json:"lift"`
	Sauna            *bool        `json:"sauna"`
	Apartments       *int         `json:"apartments"`
	SizeMin          *int         `json:"sizeMin"`
	SizeMax          *int         `json:"sizeMax"`
	Status           *int         `json:"status"`
	VrkID            *string      `json:"vrkId"`
	FormattedAddress *string      `json:"formattedAddress"`
	URL              *string      `json:"url"`
	Address          *AddressInfo `json:"address"`
	Media            []Media      `json:"media"`
}

type AddressInfo struct {
	CardID           *int           `json:"cardId"`
	CardType         *int           `json:"cardType"`
	Name             *string        `json:"name"`
	Coordinates      *Coordinates   `json:"coordinates"`
	ZipCode          *LocationInfo  `json:"zipCode"`
	Street           *LocationInfo  `json:"street"`
	City             *LocationInfo  `json:"city"`
	County           *LocationInfo  `json:"county"`
	Country          *LocationInfo  `json:"country"`
	StreetNumber     *string        `json:"streetNumber"`
	BuildingLetter   *string        `json:"buildingLetter"`
	Districts        []LocationInfo `json:"districts"`
	FormattedAddress *string        `json:"formattedAddress"`
}

type LocationInfo struct {
	Name        *string      `json:"name"`
	CardID      *int         `json:"cardId"`
	CardType    *int         `json:"cardType"`
	Coordinates *Coordinates `json:"coordinates"`
}

type Media struct {
	Order       *int     `json:"ordernr"`
	CardID      *int     `json:"card_id"`
	MediaID     int      `json:"media_id"`
	Description *string  `json:"description"`
	URLFull     string   `json:"url_full"`
	URLLarge    string   `json:"url_large"`
	URLThumb    string   `json:"url_thumb"`
	URLContact  string   `json:"url_contact"`
	URLLogo     string   `json:"url_logo"`
	URLLogoFull string   `json:"url_logo_full"`
	URLSearch   string   `json:"url_search_logo"`
	Tags        []string `json:"tags"`
}

type NumberOrString struct {
	raw string
}

func (n *NumberOrString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		n.raw = value
		return nil
	}
	n.raw = string(data)
	return nil
}

func (n NumberOrString) String() string {
	return n.raw
}
