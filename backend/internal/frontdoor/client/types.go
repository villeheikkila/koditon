package client

import (
	"encoding/json"
	"fmt"
)

type AdResponse struct {
	ID                               int                      `json:"id"`
	FriendlyID                       string                   `json:"friendlyId"`
	Status                           string                   `json:"status"`
	Text                             *string                  `json:"text"`
	CreationTime                     *int                     `json:"creationTime"`
	ModificationTime                 *int                     `json:"modificationTime"`
	PublishingTime                   int                      `json:"publishingTime"`
	AvailabilityDescription          *string                  `json:"availabilityDescription"`
	MapVisible                       *bool                    `json:"mapVisible"`
	SellingPrice                     *float64                 `json:"sellingPrice"`
	DebfFreePrice                    *float64                 `json:"debfFreePrice"`
	DebtShareAmount                  *float64                 `json:"debtShareAmount"`
	PricePerSquareMeter              *float64                 `json:"pricePerSquareMeter"`
	MoreInformationAvailableFrom     *string                  `json:"moreInformationAvailableFrom"`
	Links                            []Link                   `json:"links"`
	HasPdfBrochure                   *bool                    `json:"hasPdfBrochure"`
	AnnouncementContactInfo          *ContactInfo             `json:"announcementContactInfo"`
	Property                         Property                 `json:"property"`
	ResidenceDetails                 ResidenceDetails         `json:"residenceDetailsDTO"`
	CanReceiveLeads                  *bool                    `json:"canReceiveLeads"`
	LeadOptions                      *LeadOptions             `json:"leadOptions"`
	AnnouncementOriginDetails        *AnnouncementOrigin      `json:"announcementOriginDetails"`
	OpenBiddingInUse                 *bool                    `json:"openBiddingInUse"`
	Preparsed                        PreparsedInfo            `json:"preparsed"`
	ImageIDs                         *ImageIDs                `json:"imageIds"`
	ProductEffects                   *ProductEffects          `json:"productEffects"`
	Showings                         []Showing                `json:"showings"`
	AdditionalAnnouncementLinkList   []AdditionalAnnouncement `json:"additionalAnnouncementLinkList"`
	AdditionalItemsIncludedInSale    *string                  `json:"additionalItemsIncludedInSale"`
	ApartmentsInHousingCompany       []ApartmentInCompany     `json:"apartmentsInHousingCompany"`
	BookingStatus                    *string                  `json:"bookingStatus"`
	DebtShareAdditionalInfo          *string                  `json:"debtShareAdditionalInfo"`
	HcaDevelopmentPhase              *string                  `json:"hcaDevelopmentPhase"`
	HousingCompanyAnnouncementID     *string                  `json:"housingCompanyAnnouncementFriendlyId"`
	OpenBiddingStartingDebtFreePrice *float64                 `json:"openBiddingStartingDebtFreePrice"`
	OpenBiddingStartingSellingPrice  *float64                 `json:"openBiddingStartingSellingPrice"`
	OpenBiddingTargetURL             *string                  `json:"openBiddingTargetUrl"`
	PreviousPrice                    *PreviousPrice           `json:"previousPrice"`
	PurchasingShareOfPlot            *float64                 `json:"purchasingShareOfPlot"`
}

type Link struct {
	ID     int     `json:"id"`
	LinkID *int    `json:"linkId"`
	Type   *string `json:"type"`
	Title  *string `json:"title"`
	URL    string  `json:"url"`
}

type AdditionalAnnouncement struct {
	ID     int     `json:"id"`
	LinkID int     `json:"linkId"`
	Title  *string `json:"title"`
	Type   string  `json:"type"`
	URL    string  `json:"url"`
}

type ApartmentInCompany struct {
	AvailabilityDescription *string  `json:"availabilityDescription"`
	DebfFreePrice           *float64 `json:"debfFreePrice"`
	FloorLevel              *int     `json:"floorLevel"`
	FriendlyID              *string  `json:"friendlyId"`
	LivingArea              *float64 `json:"livingArea"`
	PropertySubtype         *string  `json:"propertySubtype"`
	PropertyType            *string  `json:"propertyType"`
	RoomStructure           *string  `json:"roomStructure"`
	SellingPrice            *float64 `json:"sellingPrice"`
	StairwayAndApartment    *string  `json:"stairwayAndApartment"`
}

type PreviousPrice struct {
	DebtFreePrice       *float64 `json:"debtFreePrice"`
	PricePerSquareMeter *float64 `json:"pricePerSquareMeter"`
	SellingPrice        *float64 `json:"sellingPrice"`
}

type Showing struct {
	EndTime          *int    `json:"endTime"`
	ID               *int    `json:"id"`
	Info             *string `json:"info"`
	IntroducerName   *string `json:"introducerName"`
	IntroducerPhone  *string `json:"introducerPhone"`
	ModificationTime *int    `json:"modificationTime"`
	StartTime        *int    `json:"startTime"`
}

type ContactInfo struct {
	Name                       *string `json:"name"`
	Phone                      *string `json:"phone"`
	MobilePhone                *string `json:"mobilePhone"`
	Title                      *string `json:"title"`
	ImageURI                   *string `json:"imageUri"`
	IsPrivateSeller            bool    `json:"isPrivateSeller"`
	OfficeName                 *string `json:"officeName"`
	OfficeID                   *int    `json:"officeId"`
	OfficeNumber               *int    `json:"officeNumber"`
	OfficeLogoURI              *string `json:"officeLogoUri"`
	OfficeStreetAddressLineOne *string `json:"officeStreetAddressLineOne"`
	OfficePostCode             *string `json:"officePostCode"`
	OfficePostOffice           *string `json:"officePostOffice"`
	OfficeMunicipality         *string `json:"officeMunicipality"`
	OfficeCountry              *string `json:"officeCountry"`
	CustomerGroupID            *int    `json:"customerGroupId"`
	CustomerGroupName          *string `json:"customerGroupName"`
	OfficePhoneNumber          *string `json:"officePhoneNumber"`
	OfficeMobilePhoneNumber    *string `json:"officeMobilePhoneNumber"`
	OfficeStreetAddressLineTwo *string `json:"officeStreetAddressLineTwo"`
}

type Property struct {
	SpecificType                         string                   `json:"specificType"`
	ID                                   int                      `json:"id"`
	PropertyType                         string                   `json:"propertyType"`
	GeoCode                              *GeoCode                 `json:"geoCode"`
	Country                              NamedItem                `json:"country"`
	Region                               *NamedItem               `json:"region"`
	Municipality                         *NamedItem               `json:"municipality"`
	District                             *NamedItem               `json:"district"`
	PostCode                             *PostCode                `json:"postCode"`
	Street                               *NamedItem               `json:"street"`
	AreaNicknames                        []NamedItem              `json:"areaNicknames"`
	HouseNumber                          *string                  `json:"houseNumber"`
	StairwayAndApartment                 *string                  `json:"stairwayAndApartment"`
	RegionNameFreeForm                   *string                  `json:"regionNameFreeForm"`
	MunicipalityNameFreeForm             *string                  `json:"municipalityNameFreeForm"`
	DistrictNameFreeForm                 *string                  `json:"districtNameFreeForm"`
	PostCodeFreeForm                     *string                  `json:"postCodeFreeForm"`
	StreetAddressFreeForm                *string                  `json:"streetAddressFreeForm"`
	Description                          *string                  `json:"description"`
	PeriodicChargesAdditionalInfo        *string                  `json:"periodicChargesAdditionalInfo"`
	TransportationServicesDescription    *string                  `json:"transportationServicesDescription"`
	WaterSupplyDescription               *string                  `json:"waterSupplyDescription"`
	NearbyAmenitiesDescription           *string                  `json:"nearbyAmenitiesDescription"`
	Yard                                 *Yard                    `json:"yard"`
	Shore                                *Shore                   `json:"shore"`
	Images                               map[string]PropertyImage `json:"images"`
	PeriodicCharges                      []PeriodicCharge         `json:"periodicCharges"`
	Share                                *bool                    `json:"share"`
	AdditionalAreaMeasurementInformation *string                  `json:"additionalAreaMeasurementInformation"`
	CarParkingInformation                *string                  `json:"carParkingInformation"`
	ResidentialPropertyType              *string                  `json:"residentialPropertyType"`
	OwnershipType                        *string                  `json:"ownershipType"`
	NewProperty                          *bool                    `json:"newProperty"`
	HousingCompany                       *HousingCompany          `json:"housingCompany"`
	DrivingInstructions                  *string                  `json:"drivingInstructions"`
	FinancingFeeInterestOnlyEnddate      *string                  `json:"financingFeeInterestOnlyEnddate"`
	FinancingFeeInterestOnlyPeriod       *string                  `json:"financingFeeInterestOnlyPeriod"`
	FinancingFeeInterestOnlyStartdate    *string                  `json:"financingFeeInterestOnlyStartdate"`
	LotRedemptionInfo                    *string                  `json:"lotRedemptionInfo"`
	LotRentalAgreement                   *string                  `json:"lotRentalAgreement"`
	ManagementChargesAdditionalInfo      *string                  `json:"managementChargesAdditionalInfo"`
	Plot                                 *Plot                    `json:"plot"`
	PlotShareRedeemed                    *bool                    `json:"plotShareRedeemed"`
	RenovationsDoneDescription           *string                  `json:"renovationsDoneDescription"`
	RenovationsPlannedDescription        *string                  `json:"renovationsPlannedDescription"`
	RoadAvailableTillProperty            *bool                    `json:"roadAvailableTillProperty"`
	SewersDescription                    *string                  `json:"sewersDescription"`
	Subdistrict                          map[string]string        `json:"subdistrict"`
	SubdistrictNameFreeForm              *string                  `json:"subdistrictNameFreeForm"`
	SubscriptionsDescription             *string                  `json:"subscriptionsDescription"`
	WaterSupplyTypes                     []string                 `json:"waterSupplyTypes"`
}

type GeoCode struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Accuracy  string   `json:"accuracy"`
}

type NamedItem struct {
	Code        string `json:"code"`
	DefaultName string `json:"defaultName"`
}

type PostCode struct {
	PostCodeKey string `json:"postCodeKey"`
	PostCode    string `json:"postCode"`
	PostArea    string `json:"postArea"`
}

type Yard struct {
	ID               int     `json:"id"`
	ViewsDescription *string `json:"viewsDescription"`
}

type Shore struct {
	ID             int     `json:"id"`
	ShoreType      *string `json:"shoreType"`
	ShoreRightType *string `json:"shoreRightType"`
}

type PropertyImage struct {
	ID                int                  `json:"id"`
	PropertyImageType string               `json:"propertyImageType"`
	Ordinal           int                  `json:"ordinal"`
	Image             PropertyImageDetails `json:"image"`
}

type PropertyImageDetails struct {
	ID   int     `json:"id"`
	UUID *string `json:"uuid"`
	URI  string  `json:"uri"`
}

type PeriodicCharge struct {
	ID             int      `json:"id"`
	PeriodicCharge string   `json:"periodicCharge"`
	Price          *float64 `json:"price"`
	ChargePeriod   *string  `json:"chargePeriod"`
}

type Plot struct {
	AdditionalBuildingsDescription *string                   `json:"additionalBuildingsDescription"`
	ConstructionRightDTO           map[string]NumberOrString `json:"constructionRightDTO"`
	ElectricConnectionTransferred  *bool                     `json:"electricConnectionTransferred"`
	EncumberanceInfo               *string                   `json:"encumberanceInfo"`
	HoldingType                    *string                   `json:"holdingType"`
	PlotArea                       *float64                  `json:"plotArea"`
	PlotName                       *string                   `json:"plotName"`
	PlotNumber                     *string                   `json:"plotNumber"`
	PlotType                       *string                   `json:"plotType"`
	RentalAgreementDTO             *RentalAgreement          `json:"rentalAgreementDTO"`
	ZoningAspectTypes              []string                  `json:"zoningAspectTypes"`
	ZoningInfo                     *string                   `json:"zoningInfo"`
	Area                           *int                      `json:"area"`
	ENumber                        *float64                  `json:"eNumber"`
	RentalAgreementType            *string                   `json:"rentalAgreementType"`
	RentingType                    *string                   `json:"rentingType"`
}

type HousingCompany struct {
	ID                              *int                `json:"id"`
	Name                            *string             `json:"name"`
	Manager                         *string             `json:"manager"`
	Maintainer                      *string             `json:"maintainer"`
	RenovationsDoneDescription      *string             `json:"renovationsDoneDescription"`
	RenovationsPlannedDescription   *string             `json:"renovationsPlannedDescription"`
	OtherInfo                       *string             `json:"otherInfo"`
	ApartmentCount                  *int                `json:"apartmentCount"`
	FloorCount                      *float64            `json:"floorCount"`
	BusinessPremiseCount            *int                `json:"businessPremiseCount"`
	CarStorageDescription           *string             `json:"carStorageDescription"`
	ConnectivityFeaturesDescription *string             `json:"connectivityFeaturesDescription"`
	MaintenanceResponsibilityType   *string             `json:"maintenanceResponsibilityType"`
	Plot                            *HousingCompanyPlot `json:"plot"`
	EnergyCertificate               *EnergyCertificate  `json:"energyCertificate"`
	HousingCompanyFeatures          []string            `json:"housingCompanyFeatures"`
	GeoCode                         *GeoCode            `json:"geoCode"`
	BusinessID                      *string             `json:"businessId"`
	ParkingSpaces                   map[string]int      `json:"parkingSpaces"`
	UsageStartYear                  *int                `json:"usageStartYear"`
}

type HousingCompanyPlot struct {
	ID                            int              `json:"id"`
	PlotNumber                    *string          `json:"plotNumber"`
	PlotArea                      *float64         `json:"plotArea"`
	HoldingType                   string           `json:"holdingType"`
	RentalAgreementDTO            *RentalAgreement `json:"rentalAgreementDTO"`
	ZoningInfo                    *string          `json:"zoningInfo"`
	ZoningAspectTypes             []string         `json:"zoningAspectTypes"`
	PlotDescription               *string          `json:"plotDescription"`
	ElectricConnectionTransferred *bool            `json:"electricConnectionTransferred"`
}

type EnergyCertificate struct {
	ID                           int     `json:"id"`
	EnergyCertificateType        *string `json:"energyCertificateType"`
	EnergyCertificateDescription *string `json:"energyCertificateDescription"`
}

type RentalAgreement struct {
	ID                      int      `json:"id"`
	RentalAgreementType     string   `json:"rentalAgreementType"`
	RentingType             string   `json:"rentingType"`
	ChargePeriod            *string  `json:"chargePeriod"`
	EndDate                 *int     `json:"endDate"`
	OwnerDescription        *string  `json:"ownerDescription"`
	Rent                    *float64 `json:"rent"`
	RentalPeriodDescription *string  `json:"rentalPeriodDescription"`
}

type ResidenceDetails struct {
	TotalRoomCount                                    *int                      `json:"totalRoomCount"`
	BedroomCount                                      *int                      `json:"bedroomCount"`
	VentilationSystemType                             *string                   `json:"ventilationSystemType"`
	VentilationSystemDescription                      *string                   `json:"ventilationSystemDescription"`
	HeatingSystemsDescription                         *string                   `json:"heatingSystemsDescription"`
	BuildingConstructionAndSurfaceMaterialDescription *string                   `json:"buildingConstructionAndSurfaceMaterialDescription"`
	BuildingConstructionMaterial                      *string                   `json:"buildingConstructionMaterial"`
	OuterRoofType                                     *string                   `json:"outerRoofType"`
	OuterRoofMaterial                                 *string                   `json:"outerRoofMaterial"`
	OuterRoofMaterialDescription                      *string                   `json:"outerRoofMaterialDescription"`
	OuterRoofTypeDescription                          *string                   `json:"outerRoofTypeDescription"`
	RoomCount                                         string                    `json:"roomCount"`
	RoomStructure                                     *string                   `json:"roomStructure"`
	LivingArea                                        *float64                  `json:"livingArea"`
	TotalArea                                         *float64                  `json:"totalArea"`
	LivingRoomDescription                             *string                   `json:"livingRoomDescription"`
	BedroomDescription                                *string                   `json:"bedroomDescription"`
	ViewsDescription                                  *string                   `json:"viewsDescription"`
	BalconyDescription                                *string                   `json:"balconyDescription"`
	HousingCompanyApartmentInformation                *ApartmentInfo            `json:"housingCompanyApartmentInformationDTO"`
	ConnectivityFeaturesDescription                   *string                   `json:"connectivityFeaturesDescription"`
	KichenDescription                                 *string                   `json:"kitchenDescription"`
	BathroomDescription                               *string                   `json:"bathroomDescription"`
	StorageSpaceDescription                           *string                   `json:"storageSpaceDescription"`
	ConstructionFinishedYear                          *int                      `json:"constructionFinishedYear"`
	ToiletDescription                                 *string                   `json:"toiletDescription"`
	FloorMaterialsDescription                         *string                   `json:"floorMaterialsDescription"`
	WallMaterialsDescription                          *string                   `json:"wallMaterialsDescription"`
	Inspection                                        *Inspection               `json:"inspection"`
	GeneralDwellingFeatures                           []string                  `json:"generalDwellingFeatures"`
	HeatingSystems                                    []string                  `json:"heatingSystems"`
	CarParkingFeatures                                []string                  `json:"carParkingFeatures"`
	EnergyCertificate                                 map[string]NumberOrString `json:"energyCertificate"`
	ExtraApartmentFeaturesDescription                 *string                   `json:"extraApartmentFeaturesDescription"`
	ExtraApartmentInfo                                *string                   `json:"extraApartmentInfo"`
	FireplaceDescription                              *string                   `json:"fireplaceDescription"`
	FloorCount                                        *float64                  `json:"floorCount"`
	FloorsDescription                                 *string                   `json:"floorsDescription"`
	OtherSpaceArea                                    *float64                  `json:"otherSpaceArea"`
	OtherSpaceDescription                             *string                   `json:"otherSpaceDescription"`
	ResidentialFloorCountType                         *string                   `json:"residentialFloorCountType"`
	SaunaDescription                                  *string                   `json:"saunaDescription"`
	ToiletCount                                       *int                      `json:"toiletCount"`
	UsageStartedYear                                  *int                      `json:"usageStartedYear"`
	UtilityRoomDescription                            *string                   `json:"utilityRoomDescription"`
}

type ApartmentInfo struct {
	ID                      int      `json:"id"`
	FloorLevel              *float64 `json:"floorLevel"`
	FloorPositionInHighrise *string  `json:"floorPositionInHighrise"`
}

type Inspection struct {
	ID                          int     `json:"id"`
	OverallCondition            *string `json:"overallCondition"`
	OverallConditionDescription *string `json:"overallConditionDescription"`
	AsbestosMapping             *bool   `json:"asbestosMapping"`
	AsbestosMappingDescription  *string `json:"asbestosMappingDescription"`
	ConditionInspectionDate     *int    `json:"conditionInspectionDate"`
	ConditionInspectionDone     *bool   `json:"conditionInspectionDone"`
	HumidityInspectionDone      *bool   `json:"humidityInspectionDone"`
}

type LeadOptions struct {
	ShowWantToBeContactedOption      bool `json:"showWantToBeContactedOption"`
	ShowWantPropertyEstimationOption bool `json:"showWantPropertyEstimationOption"`
	ShowWantMoreDetailsOption        bool `json:"showWantMoreDetailsOption"`
	ShowWantToReserveBookingOption   bool `json:"showWantToReserveBookingOption"`
	ShowWantToOrderBrochureOption    bool `json:"showWantToOrderBrochureOption"`
}

type AnnouncementOrigin struct {
	CustomerItemCode       *string `json:"customerItemCode"`
	SupplierIdentifier     *string `json:"supplierIdentifier"`
	DTSIntegrationSourceID *int    `json:"dtsIntegrationSourceId"`
}

type PreparsedInfo struct {
	Title *string  `json:"title"`
	Area  *float64 `json:"area"`
	Price *float64 `json:"price"`
}

type ImageIDs struct {
	MainImageID       *int  `json:"mainImageId"`
	Sorted            []int `json:"sorted"`
	FloorPlanImageIDs []int `json:"floorPlanImageIds"`
}

type ProductEffects struct {
	OfficeLogo                   *OfficeLogo         `json:"officeLogo"`
	BackgroundTexture            map[string]string   `json:"backgroundTexture"`
	BrandColors                  *BrandColors        `json:"brandColors"`
	PromotionalImage             *PromotionalImage   `json:"promotionalImage"`
	Chat                         *Chat               `json:"chat"`
	AdSlots                      *AdSlots            `json:"adSlots"`
	ItemPageMobileAd             *ItemPageMobileAd   `json:"itemPageMobileAd"`
	ItemPageImageConfiguration   *ImageConfiguration `json:"itemPageImageConfiguration"`
	ItemPageAlsoForSale          *AlsoForSale        `json:"itemPageAlsoForSale"`
	ListPageImageConfiguration   *ImageConfiguration `json:"listPageImageConfiguration"`
	OfficePageLink               map[string]string   `json:"officePageLink"`
	OpenBidding                  *OpenBidding        `json:"openBidding"`
	VideoMainImage               *VideoMainImage     `json:"videoMainImage"`
	LoanCalculator               *FeatureEnabled     `json:"loanCalculator"`
	RenovationCalculator         *FeatureEnabled     `json:"renovationCalculator"`
	MovingCalculator             *FeatureEnabled     `json:"movingCalculator"`
	OtherServicesAd              *FeatureEnabled     `json:"otherServicesAd"`
	LargerImages                 map[string]bool     `json:"largerImages"`
	RakennuttajanStudio          map[string]string   `json:"rakennuttajanStudio"`
	RealtorExtensions            map[string]bool     `json:"realtorExtensions"`
	VirtualPresentationMainImage *FeatureEnabled     `json:"virtualPresentationMainImage"`
}

type OfficeLogo struct {
	FilePath  *string `json:"filePath"`
	TargetURL *string `json:"targetUrl"`
}

type BrandColors struct {
	BrandColor *string `json:"brandColor"`
	TextColor  *string `json:"textColor"`
}

type PromotionalImage struct {
	ImageURL                *string `json:"imageUrl"`
	TargetURL               *string `json:"targetUrl"`
	UseAlternativeTargetURL bool    `json:"useAlternativeTargetUrl"`
}

type Chat struct {
	ChatEnabled      bool   `json:"chatEnabled"`
	ChatRoomKey      string `json:"chatRoomKey"`
	ChatRealtorEmail string `json:"chatRealtorEmail"`
	ChatOfficeName   string `json:"chatOfficeName"`
}

type AdSlots struct {
	Banner1 *Banner `json:"banner1"`
	Banner2 *Banner `json:"banner2"`
	Banner3 *Banner `json:"banner3"`
}

type Banner struct {
	AdAlternative bool    `json:"adAlternative"`
	FilePath      *string `json:"filePath"`
	TargetURL     *string `json:"targetUrl"`
}

type ItemPageMobileAd struct {
	BannerMobile *Banner `json:"bannerMobile"`
}

type ImageConfiguration struct {
	MaxImageCount int `json:"maxImageCount"`
}

type AlsoForSale struct {
	Scope           *string `json:"scope"`
	CustomerGroupID int     `json:"customerGroupId"`
}

type OpenBidding struct {
	HeaderLogoFilePath        *string `json:"headerLogoFilePath"`
	BigLogoFilePath           *string `json:"bigLogoFilePath"`
	SmallLogoFilePath         *string `json:"smallLogoFilePath"`
	ItemPageTargetURLTemplate *string `json:"itemPageTargetUrlTemplate"`
	InformationPageURL        *string `json:"informationPageUrl"`
	InformationPageURLText    *string `json:"informationPageUrlText"`
	InformationPageTextColor  *string `json:"informationPageTextColor"`
}

type VideoMainImage struct {
	Enabled bool `json:"enabled"`
}

type FeatureEnabled struct {
	Enabled        bool    `json:"enabled"`
	BankName       *string `json:"bankName"`
	UsesSingleBank *bool   `json:"usesSingleBank"`
}

type NumberOrString struct {
	IntValue    *int
	DoubleValue *float64
	StringValue *string
}

func (n *NumberOrString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		n.IntValue = &intValue
		return nil
	}
	var floatValue float64
	if err := json.Unmarshal(data, &floatValue); err == nil {
		if floatValue == float64(int(floatValue)) {
			intValue := int(floatValue)
			n.IntValue = &intValue
		} else {
			n.DoubleValue = &floatValue
		}
		return nil
	}
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		n.StringValue = &stringValue
		return nil
	}

	return fmt.Errorf("value is neither int, float64, nor string")
}

func (n NumberOrString) MarshalJSON() ([]byte, error) {
	if n.IntValue != nil {
		return json.Marshal(*n.IntValue)
	}
	if n.DoubleValue != nil {
		return json.Marshal(*n.DoubleValue)
	}
	if n.StringValue != nil {
		return json.Marshal(*n.StringValue)
	}
	return json.Marshal(nil)
}

func (n NumberOrString) String() string {
	if n.IntValue != nil {
		return fmt.Sprintf("%d", *n.IntValue)
	}
	if n.DoubleValue != nil {
		return fmt.Sprintf("%f", *n.DoubleValue)
	}
	if n.StringValue != nil {
		return *n.StringValue
	}
	return ""
}

type HousingCompanyResponse struct {
	HousingCompanyPage    *HousingCompanyPage    `json:"housing-company-page"`
	KsaHousingCompanyPage *KsaHousingCompanyPage `json:"ksa-housing-company-page"`
}

type HousingCompanyPage struct {
	Response *HousingCompanyPageResponse `json:"response"`
}

type HousingCompanyPageResponse struct {
	HousingCompanyAnnouncement *HousingCompanyAnnouncement `json:"housingCompanyAnnouncement"`
	ApartmentsInHousingCompany []Listing                   `json:"apartmentsInHousingCompany"`
	ImageIDs                   *ImageIDsSimple             `json:"imageIds"`
}

type HousingCompanyAnnouncement struct {
	ID               *int                        `json:"id"`
	FriendlyID       *string                     `json:"friendlyId"`
	Status           *string                     `json:"status"`
	DevelopmentPhase *string                     `json:"developmentPhase"`
	Text             *string                     `json:"text"`
	ContactInfo      *HousingContact             `json:"contactInfo"`
	Links            []AnnouncementLink          `json:"links"`
	HousingCompany   *AnnouncementHousingCompany `json:"housingCompany"`
}

type AnnouncementLink struct {
	ID     *int    `json:"id"`
	LinkID *int    `json:"linkId"`
	Type   *string `json:"type"`
	Title  *string `json:"title"`
	URL    *string `json:"url"`
}

type HousingContact struct {
	Phone                      *string `json:"phone"`
	OfficeName                 *string `json:"officeName"`
	OfficeID                   *int    `json:"officeId"`
	OfficeStreetAddressLineOne *string `json:"officeStreetAddressLineOne"`
	OfficeStreetAddressLineTwo *string `json:"officeStreetAddressLineTwo"`
	OfficePostcode             *string `json:"officePostcode"`
	OfficePostOffice           *string `json:"officePostOffice"`
	CustomerGroupID            *int    `json:"customerGroupId"`
	CustomerGroupName          *string `json:"customerGroupName"`
}

type AnnouncementHousingCompany struct {
	ID                       *int                      `json:"id"`
	Name                     *string                   `json:"name"`
	OtherInfo                *string                   `json:"otherInfo"`
	ApartmentCount           *int                      `json:"apartmentCount"`
	FloorCount               *float64                  `json:"floorCount"`
	Plot                     *HousingCompanyPlotSimple `json:"plot"`
	GeoCode                  *GeoCode                  `json:"geoCode"`
	Country                  *Country                  `json:"country"`
	Region                   *Region                   `json:"region"`
	Municipality             *Municipality             `json:"municipality"`
	District                 *District                 `json:"district"`
	PostCode                 *PostCodeSimple           `json:"postCode"`
	Street                   *Street                   `json:"street"`
	HouseNumber              *string                   `json:"houseNumber"`
	MunicipalityNameFreeForm *string                   `json:"municipalityNameFreeForm"`
	DistrictNameFreeForm     *string                   `json:"districtNameFreeForm"`
	PostCodeFreeForm         *string                   `json:"postCodeFreeForm"`
	StreetAddressFreeForm    *string                   `json:"streetAddressFreeForm"`
	ConstructionEndYear      *int                      `json:"constructionEndYear"`
}

type HousingCompanyPlotSimple struct {
	ID          *int    `json:"id"`
	PlotArea    *int    `json:"plotArea"`
	HoldingType *string `json:"holdingType"`
}

type Country struct {
	Code        *string `json:"code"`
	DefaultName *string `json:"defaultName"`
}

type Region struct {
	Code        *string `json:"code"`
	DefaultName *string `json:"defaultName"`
}

type Municipality struct {
	Code        *string `json:"code"`
	DefaultName *string `json:"defaultName"`
}

type District struct {
	Code        *string `json:"code"`
	DefaultName *string `json:"defaultName"`
}

type PostCodeSimple struct {
	PostCodeKey *string `json:"postCodeKey"`
	PostCode    *string `json:"postCode"`
	PostArea    *string `json:"postArea"`
}

type Street struct {
	Code        *string `json:"code"`
	DefaultName *string `json:"defaultName"`
}

type Listing struct {
	FriendlyID              *string  `json:"friendlyId"`
	StairwayAndApartment    *string  `json:"stairwayAndApartment"`
	RoomStructure           *string  `json:"roomStructure"`
	LivingArea              *float64 `json:"livingArea"`
	FloorLevel              *float64 `json:"floorLevel"`
	SellingPrice            *float64 `json:"sellingPrice"`
	DebfFreePrice           *float64 `json:"debfFreePrice"`
	AvailabilityDescription *string  `json:"availabilityDescription"`
	PropertyType            *string  `json:"propertyType"`
	PropertySubtype         *string  `json:"propertySubtype"`
}

type ImageIDsSimple struct {
	MainImageID *int  `json:"mainImageId"`
	Sorted      []int `json:"sorted"`
}

type KsaHousingCompanyPage struct {
	Response *KsaHousingCompanyResponse `json:"response"`
}

type KsaHousingCompanyResponse struct {
	BusinessID                     *string                    `json:"businessId"`
	CompanyName                    *string                    `json:"companyName"`
	AdHousingCompanyInfo           *AdHousingCompanyInfo      `json:"adHousingCompanyInfo"`
	HouseAddresses                 []HouseAddress             `json:"houseAddresses"`
	BuildingsGroupedByPurpose      *BuildingsGroupedByPurpose `json:"buildingsGroupedByPurpose"`
	PublishedAnnouncements         []Announcement             `json:"publishedAnnouncements"`
	UnpublishedAnnouncements       []Announcement             `json:"unpublishedAnnouncements"`
	PublishedRentalAnnouncements   []Announcement             `json:"publishedRentalAnnouncements"`
	UnpublishedRentalAnnouncements []Announcement             `json:"unpublishedRentalAnnouncements"`
}

type Announcement struct {
	ID                       *int     `json:"id"`
	FriendlyID               *string  `json:"friendlyId"`
	UnpublishingTime         *float64 `json:"unpublishingTime"`
	AddressLine1             *string  `json:"addressLine1"`
	AddressLine2             *string  `json:"addressLine2"`
	Location                 *string  `json:"location"`
	SearchPrice              *float64 `json:"searchPrice"`
	NotifyPriceChanged       *bool    `json:"notifyPriceChanged"`
	PropertyType             *string  `json:"propertyType"`
	PropertySubtype          *string  `json:"propertySubtype"`
	ConstructionFinishedYear *int     `json:"constructionFinishedYear"`
	MainImageURI             *string  `json:"mainImageUri"`
	HasOpenBidding           *bool    `json:"hasOpenBidding"`
	RoomStructure            *string  `json:"roomStructure"`
	Area                     *float64 `json:"area"`
	TotalArea                *float64 `json:"totalArea"`
	PricePerSquare           *float64 `json:"pricePerSquare"`
	DaysOnMarket             *int     `json:"daysOnMarket"`
	NewBuilding              *bool    `json:"newBuilding"`
	Office                   *Office  `json:"office"`
	MainImageHidden          *bool    `json:"mainImageHidden"`
	IsCompanyAnnouncement    *bool    `json:"isCompanyAnnouncement"`
	ShowBiddingIndicators    *bool    `json:"showBiddingIndicators"`
	Published                *bool    `json:"published"`
	RentPeriod               *string  `json:"rentPeriod"`
	RentalUniqueNo           *int     `json:"rentalUniqueNo"`
}

type Office struct {
	ID              *int    `json:"id"`
	LogoURI         *string `json:"logoUri"`
	WebPageURL      *string `json:"webPageUrl"`
	Name            *string `json:"name"`
	CustomerGroupID *int    `json:"customerGroupId"`
	OfficeNumber    *int    `json:"officeNumber"`
}

type AdHousingCompanyInfo struct {
	FloorCount                        *float64                   `json:"floorCount"`
	HasElevator                       *bool                      `json:"hasElevator"`
	ApartmentCount                    *int                       `json:"apartmentCount"`
	BusinessPremiseCount              *int                       `json:"businessPremiseCount"`
	CarStorageDescription             *string                    `json:"carStorageDescription"`
	EnergyCertificateCode             *string                    `json:"energyCertificateCode"`
	PlotHoldingType                   *string                    `json:"plotHoldingType"`
	OuterRoofMaterial                 *string                    `json:"outerRoofMaterial"`
	OuterRoofType                     *string                    `json:"outerRoofType"`
	HasSauna                          *bool                      `json:"hasSauna"`
	ClassifiedPastRenovationsDetected *ClassifiedPastRenovations `json:"classifiedPastRenovationsDetected"`
}

type ClassifiedPastRenovations struct {
	ElevatorRenovatedYear    *int  `json:"elevatorRenovatedYear"`
	ElevatorRenovated        *bool `json:"elevatorRenovated"`
	FacadeRenovatedYear      *int  `json:"facadeRenovatedYear"`
	FacadeRenovated          *bool `json:"facadeRenovated"`
	WindowRenovatedYear      *int  `json:"windowRenovatedYear"`
	WindowRenovated          *bool `json:"windowRenovated"`
	RoofRenovatedYear        *int  `json:"roofRenovatedYear"`
	RoofRenovated            *bool `json:"roofRenovated"`
	PipeRenovatedYear        *int  `json:"pipeRenovatedYear"`
	PipeRenovated            *bool `json:"pipeRenovated"`
	BalconyRenovatedYear     *int  `json:"balconyRenovatedYear"`
	BalconyRenovated         *bool `json:"balconyRenovated"`
	ElectricityRenovatedYear *int  `json:"electricityRenovatedYear"`
	ElectricityRenovated     *bool `json:"electricityRenovated"`
}

type HouseAddress struct {
	StreetAddress *string  `json:"streetAddress"`
	Municipality  *string  `json:"municipality"`
	Postcode      *string  `json:"postcode"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	District      *string  `json:"district"`
}

type BuildingsGroupedByPurpose struct {
	ResidentialOrBusinessPremises []Building `json:"RESIDENTIAL_OR_BUSINESS_PREMISES"`
}

type Building struct {
	BuildingCode       *string  `json:"buildingCode"`
	PropertyCode       *string  `json:"propertyCode"`
	BuildingType       *string  `json:"buildingType"`
	BuildYear          *int     `json:"buildYear"`
	Heating            *string  `json:"heating"`
	HeatingFuel        []string `json:"heatingFuel"`
	CityApartmentCount *int     `json:"cityApartmentCount"`
	CityHasElevator    *bool    `json:"cityHasElevator"`
	FloorCount         *float64 `json:"floorCount"`
}
