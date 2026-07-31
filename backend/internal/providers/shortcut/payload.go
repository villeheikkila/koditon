package shortcut

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const ShortcutAdPayloadSchemaVersion int16 = 1

var ErrInvalidShortcutAdPayload = errors.New("invalid shortcut ad payload")

type AdType string

const (
	AdTypeListing AdType = "listing"
	AdTypeRental  AdType = "rental"
)

type ShortcutAdPayloadV1 struct {
	AdID               int64
	AdType             AdType
	Address            ShortcutAddressPayloadV1
	BuildingExternalID *int64
	Price              ShortcutPricePayloadV1
	Raw                json.RawMessage
	SchemaVersion      int16
}

// ShortcutPricePayloadV1 contains normalized source price fields.
type ShortcutPricePayloadV1 struct {
	AskingPrice     *float64
	DebtFreePrice   *float64
	DebtShareAmount *float64
	PricePerM2      *float64
}

// ShortcutAddressPayloadV1 contains structured address fields used for matching.
type ShortcutAddressPayloadV1 struct {
	StreetName       *string
	StreetNumber     *string
	BuildingLetter   *string
	City             *string
	Postal           *string
	FormattedAddress *string
}

type RawAd map[string]any

func DecodeStoredAd(raw json.RawMessage) (*ShortcutAdPayloadV1, RawAd, error) {
	payload, err := ValidateShortcutAdPayloadV1(raw, 0)
	if err != nil {
		return nil, nil, err
	}
	rawAd, err := DecodeAdRaw(raw)
	if err != nil {
		return nil, nil, err
	}
	return payload, rawAd, nil
}

func DecodeAdRaw(data []byte) (RawAd, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, payloadError("empty payload")
	}
	var ad RawAd
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&ad); err != nil {
		return nil, payloadError(fmt.Sprintf("decode raw payload: %v", err))
	}
	if err := ensureObjectRoot(data); err != nil {
		return nil, err
	}
	return ad, nil
}

type shortcutAdPayloadV1Wire struct {
	CardID       IntLike            `json:"cardId"`
	ID           IntLike            `json:"id"`
	AdID         IntLike            `json:"adId"`
	CardType     IntLike            `json:"cardType"`
	BuildingID   IntLike            `json:"buildingId"`
	Rooms        IntLike            `json:"rooms"`
	Address      addressPayloadV1   `json:"address"`
	PriceData    pricePayloadV1     `json:"priceData"`
	AdData       adDataPayloadV1    `json:"adData"`
	BuildingData *buildingPayloadV1 `json:"buildingData"`
	Building     *buildingPayloadV1 `json:"building"`
	Property     propertyPayloadV1  `json:"property"`
	Description  StringLike         `json:"description"`
	Text         StringLike         `json:"text"`
	Raw          json.RawMessage    `json:"-"`
}

type addressPayloadV1 struct {
	Street           addressEntryPayloadV1 `json:"street"`
	City             addressEntryPayloadV1 `json:"city"`
	ZipCode          addressEntryPayloadV1 `json:"zipCode"`
	StreetNumber     StringLike            `json:"streetNumber"`
	BuildingLetter   StringLike            `json:"buildingLetter"`
	FormattedAddress StringLike            `json:"formattedAddress"`
}

type addressEntryPayloadV1 struct {
	Name  StringLike `json:"name"`
	Value StringLike `json:"value"`
	Raw   StringLike `json:"-"`
}

type pricePayloadV1 struct {
	Price             FloatLike  `json:"price"`
	PriceSell         FloatLike  `json:"priceSell"`
	PriceDebtFree     FloatLike  `json:"priceDebtFree"`
	RentPerMonth      FloatLike  `json:"rentPerMonth"`
	RentPerDay        FloatLike  `json:"rentPerDay"`
	RentPerWeek       FloatLike  `json:"rentPerWeek"`
	RentPerWeekend    FloatLike  `json:"rentPerWeekend"`
	RentPerYear       FloatLike  `json:"rentPerYear"`
	MaintenanceCharge FloatLike  `json:"maintenanceCharge"`
	MonthlyFee        FloatLike  `json:"monthlyFee"`
	TotalCharge       FloatLike  `json:"totalCharge"`
	WaterFee          FloatLike  `json:"waterFee"`
	DebtShare         FloatLike  `json:"debtShare"`
	PricePerSqm       FloatLike  `json:"pricePerSqm"`
	PricePerSquareM   FloatLike  `json:"pricePerSquareMeter"`
	ChargesText       StringLike `json:"chargesText"`
	AdditionalInfo    StringLike `json:"additionalInfo"`
}

type adDataPayloadV1 struct {
	Size                          FloatLike  `json:"size"`
	SizeTotal                     FloatLike  `json:"sizeTotal"`
	SizeLiving                    FloatLike  `json:"sizeLiving"`
	SizeMin                       FloatLike  `json:"sizeMin"`
	SizeMax                       FloatLike  `json:"sizeMax"`
	BuildingOverrideTotalSize     FloatLike  `json:"buildingOverrideTotalSize"`
	BuildingOverrideSizeMin       FloatLike  `json:"buildingOverrideSizeMin"`
	BuildingOverrideSizeMax       FloatLike  `json:"buildingOverrideSizeMax"`
	Floor                         IntLike    `json:"floor"`
	TotalFloors                   IntLike    `json:"totalFloors"`
	ConstructionYear              IntLike    `json:"constructionYear"`
	Rooms                         IntLike    `json:"rooms"`
	Elevator                      BoolLike   `json:"elevator"`
	HasElevator                   BoolLike   `json:"hasElevator"`
	Sauna                         BoolLike   `json:"sauna"`
	HasSauna                      BoolLike   `json:"hasSauna"`
	Description                   StringLike `json:"description"`
	AvailabilityDescription       StringLike `json:"availabilityDescription"`
	AvailableFrom                 StringLike `json:"availableFrom"`
	RenovationsDoneDescription    StringLike `json:"renovationsDoneDescription"`
	RenovationsPlannedDescription StringLike `json:"renovationsPlannedDescription"`
	AdditionalInfo                StringLike `json:"additionalInfo"`
	Condition                     StringLike `json:"condition"`
	EnergyClass                   StringLike `json:"energyClass"`
	PlotType                      StringLike `json:"plotType"`
}

type buildingPayloadV1 struct {
	BuildingID IntLike `json:"buildingId"`
	Floors     IntLike `json:"floors"`
	Year       IntLike `json:"year"`
}

type propertyPayloadV1 struct {
	Condition                       StringLike `json:"condition"`
	EnergyClass                     StringLike `json:"energyClass"`
	PlotType                        StringLike `json:"plotType"`
	RenovationsDoneDescription      StringLike `json:"renovationsDoneDescription"`
	RenovationsPlannedDescription   StringLike `json:"renovationsPlannedDescription"`
	OtherInfo                       StringLike `json:"otherInfo"`
	PeriodicChargesAdditionalInfo   StringLike `json:"periodicChargesAdditionalInfo"`
	ManagementChargesAdditionalInfo StringLike `json:"managementChargesAdditionalInfo"`
}

func ValidateShortcutAdPayloadV1(raw json.RawMessage, expectedAdID int64) (*ShortcutAdPayloadV1, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, payloadError("empty payload")
	}
	var wire shortcutAdPayloadV1Wire
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&wire); err != nil {
		return nil, payloadError(fmt.Sprintf("decode payload: %v", err))
	}
	if err := ensureObjectRoot(raw); err != nil {
		return nil, err
	}
	wire.Raw = raw
	adID, err := wire.validate(expectedAdID)
	if err != nil {
		return nil, err
	}
	cardType, _ := wire.CardType.Int64()
	adType, err := adTypeFromCardType(cardType)
	if err != nil {
		return nil, err
	}
	return &ShortcutAdPayloadV1{AdID: adID, AdType: adType, Address: wire.Address.Payload(), BuildingExternalID: wire.BuildingExternalID(), Price: wire.PriceData.Payload(adType), Raw: raw, SchemaVersion: ShortcutAdPayloadSchemaVersion}, nil
}

func (p shortcutAdPayloadV1Wire) validate(expectedAdID int64) (int64, error) {
	adID, ok := firstIntLike(p.CardID, p.ID, p.AdID)
	if !ok {
		return 0, payloadError("missing numeric id")
	}
	if expectedAdID > 0 && adID != expectedAdID {
		return 0, payloadError("ad id mismatch")
	}
	if _, ok := p.CardType.Int64(); !ok {
		return 0, payloadError("missing cardType")
	}
	if err := p.Address.Validate(); err != nil {
		return 0, err
	}
	if !p.HasBuildingData() {
		return 0, payloadError("missing building data")
	}
	return adID, nil
}

func (p shortcutAdPayloadV1Wire) HasBuildingData() bool {
	if p.BuildingData != nil {
		return true
	}
	if p.Building != nil {
		return true
	}
	return p.BuildingID.Valid()
}

func (p shortcutAdPayloadV1Wire) BuildingExternalID() *int64 {
	if p.BuildingData != nil {
		if value, ok := p.BuildingData.BuildingID.Int64(); ok {
			return &value
		}
	}
	if p.Building != nil {
		if value, ok := p.Building.BuildingID.Int64(); ok {
			return &value
		}
	}
	if value, ok := p.BuildingID.Int64(); ok {
		return &value
	}
	return nil
}

func (p addressPayloadV1) Validate() error {
	if !p.HasAddressSignal() {
		return payloadError("address missing usable text")
	}
	return nil
}

func (p addressPayloadV1) HasAddressSignal() bool {
	return p.Street.HasText() || p.City.HasText() || p.ZipCode.HasText() || p.StreetNumber.Valid() || p.BuildingLetter.Valid() || p.FormattedAddress.Valid()
}

func (p addressPayloadV1) Payload() ShortcutAddressPayloadV1 {
	return ShortcutAddressPayloadV1{
		StreetName:       p.Street.TextPtr(),
		StreetNumber:     p.StreetNumber.Ptr(),
		BuildingLetter:   p.BuildingLetter.Ptr(),
		City:             p.City.TextPtr(),
		Postal:           p.ZipCode.TextPtr(),
		FormattedAddress: p.FormattedAddress.Ptr(),
	}
}

func (p pricePayloadV1) HasUsablePrice() bool {
	return anyFloatLike(p.PriceSell, p.Price, p.PriceDebtFree, p.RentPerMonth, p.RentPerDay, p.RentPerWeek, p.RentPerWeekend, p.RentPerYear)
}

func (p pricePayloadV1) Payload(adType AdType) ShortcutPricePayloadV1 {
	askingPrice := firstFloatLikePtr(p.PriceSell, p.Price)
	if adType == AdTypeRental {
		askingPrice = firstFloatLikePtr(p.RentPerMonth, p.RentPerWeek, p.RentPerDay, p.RentPerWeekend, p.RentPerYear, p.Price)
	}
	return ShortcutPricePayloadV1{
		AskingPrice:     askingPrice,
		DebtFreePrice:   firstFloatLikePtr(p.PriceDebtFree),
		DebtShareAmount: firstFloatLikePtr(p.DebtShare),
		PricePerM2:      firstFloatLikePtr(p.PricePerSqm, p.PricePerSquareM),
	}
}

func firstFloatLikePtr(values ...FloatLike) *float64 {
	for _, value := range values {
		if parsed, ok := value.Float64(); ok {
			return &parsed
		}
	}
	return nil
}

func (p adDataPayloadV1) HasUsableSize() bool {
	return anyFloatLike(p.Size, p.SizeTotal, p.SizeLiving, p.SizeMin, p.SizeMax, p.BuildingOverrideTotalSize, p.BuildingOverrideSizeMin, p.BuildingOverrideSizeMax)
}

func adTypeFromCardType(cardType int64) (AdType, error) {
	switch cardType {
	case 100:
		return AdTypeListing, nil
	case 101:
		return AdTypeRental, nil
	default:
		return "", payloadError(fmt.Sprintf("unsupported card type %d", cardType))
	}
}

type IntLike struct {
	value int64
	valid bool
}

func (n *IntLike) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		n.valid = false
		n.value = 0
		return nil
	}
	raw, err := scalarString(data)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		n.valid = false
		n.value = 0
		return nil
	}
	value, ok := parseIntString(raw)
	if !ok {
		return fmt.Errorf("expected integer-like value")
	}
	n.value = value
	n.valid = true
	return nil
}

func (n IntLike) Int64() (int64, bool) {
	return n.value, n.valid
}

func (n IntLike) Valid() bool {
	return n.valid
}

type FloatLike struct {
	value float64
	valid bool
}

func (n *FloatLike) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		n.valid = false
		n.value = 0
		return nil
	}
	raw, err := scalarString(data)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		n.valid = false
		n.value = 0
		return nil
	}
	value, ok := parseFloatString(raw)
	if !ok {
		return fmt.Errorf("expected number-like value")
	}
	n.value = value
	n.valid = true
	return nil
}

func (n FloatLike) Float64() (float64, bool) {
	return n.value, n.valid
}

func (n FloatLike) Valid() bool {
	return n.valid
}

type BoolLike struct {
	value bool
	valid bool
}

func (n *BoolLike) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		n.valid = false
		n.value = false
		return nil
	}
	raw, err := scalarString(data)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		n.valid = false
		n.value = false
		return nil
	}
	value, ok := parseBoolString(raw)
	if !ok {
		return fmt.Errorf("expected boolean-like value")
	}
	n.value = value
	n.valid = true
	return nil
}

func (n BoolLike) Bool() (bool, bool) {
	return n.value, n.valid
}

func (n BoolLike) Valid() bool {
	return n.valid
}

type StringLike struct {
	value string
	valid bool
}

func (s *StringLike) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		s.value = ""
		s.valid = false
		return nil
	}
	raw, err := scalarString(data)
	if err != nil {
		return err
	}
	s.value = strings.TrimSpace(raw)
	s.valid = s.value != ""
	return nil
}

func (s StringLike) String() (string, bool) {
	return s.value, s.valid
}

func (s StringLike) Valid() bool {
	return s.valid
}

func (s StringLike) Ptr() *string {
	if !s.valid {
		return nil
	}
	return &s.value
}

func (a *addressEntryPayloadV1) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		*a = addressEntryPayloadV1{}
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		return a.Raw.UnmarshalJSON(data)
	}
	type alias addressEntryPayloadV1
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*a = addressEntryPayloadV1(decoded)
	return nil
}

func (a addressEntryPayloadV1) HasText() bool {
	return a.Raw.Valid() || a.Name.Valid() || a.Value.Valid()
}

func (a addressEntryPayloadV1) TextPtr() *string {
	if value, ok := a.Name.String(); ok {
		return &value
	}
	if value, ok := a.Value.String(); ok {
		return &value
	}
	if value, ok := a.Raw.String(); ok {
		return &value
	}
	return nil
}

func firstIntLike(values ...IntLike) (int64, bool) {
	for _, value := range values {
		if parsed, ok := value.Int64(); ok {
			return parsed, true
		}
	}
	return 0, false
}

func anyFloatLike(values ...FloatLike) bool {
	for _, value := range values {
		if value.Valid() {
			return true
		}
	}
	return false
}

func ensureObjectRoot(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return payloadError("empty payload")
	}
	if trimmed[0] != '{' {
		return payloadError("payload root must be object")
	}
	return nil
}

func scalarString(data []byte) (string, error) {
	var s string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return "", err
		}
		return s, nil
	}
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return "", err
	}
	switch value := raw.(type) {
	case json.Number:
		return value.String(), nil
	case bool:
		if value {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("expected scalar value")
	}
}

func parseIntString(value string) (int64, bool) {
	cleaned := cleanNumberString(value)
	parsed, err := strconv.ParseInt(cleaned, 10, 64)
	if err == nil {
		return parsed, true
	}
	asFloat, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false
	}
	return int64(asFloat), true
}

func parseFloatString(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(cleanNumberString(value), 64)
	return parsed, err == nil
}

func parseBoolString(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "kylla", "kyllä":
		return true, true
	case "0", "false", "no", "off", "ei":
		return false, true
	default:
		return false, false
	}
}

func cleanNumberString(value string) string {
	cleaned := strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return cleaned
}

func isJSONNull(data []byte) bool {
	return bytes.Equal(bytes.TrimSpace(data), []byte("null"))
}

func payloadError(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidShortcutAdPayload, message)
}
