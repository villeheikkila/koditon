package frontdoor

import "testing"

func TestAdResponseUnmarshalLiveSchemaFields(t *testing.T) {
	raw := []byte(`{"id":1234567890123,"friendlyId":"ad-1","status":"PUBLISHED","publishingTime":1710000000000,"unpublishingTime":1719999999999,"openBiddingLatestOffer":250000,"announcementContactInfo":{"isPrivateSeller":false,"isPremiumRealtor":true,"officeId":9876543210,"customerGroupId":42},"property":{"id":1001,"specificType":"APARTMENT","propertyType":"APARTMENT","plotPropertyType":"OWN","leisurePropertyType":"COTTAGE","periodicCharges":[{"id":1,"periodicCharge":"MAINTENANCE_CHARGE","price":123.45,"includedInOverallCost":true,"basedOnConsumption":false}],"shore":{"id":2,"description":"shore","watercourseName":"lake"},"housingCompany":{"id":3,"builder":"builder"},"images":{"10":{"id":10,"ordinal":1,"propertyImageType":"MAIN","image":{"id":11,"uri":"image","description":"main"}}}},"productEffects":{"backgroundTexture":{"imageUrl":"texture"},"officePageLink":{"shortUrlName":"office"},"largerImages":{"largerImagesEnabled":true,"videoEnabled":true,"virtualPresentationEnabled":false},"rakennuttajanStudio":{"id":"studio"},"realtorExtensions":{"contactCardEnabled":true},"itemPagePremiumFeatures":{"enabled":true},"listPagePremiumCard":{"enabled":false}}}`)
	ad, err := DecodeAd(raw)
	if err != nil {
		t.Fatalf("unmarshal ad response: %v", err)
	}
	if ad.ResidenceDetails != nil {
		t.Fatal("expected missing residenceDetailsDTO to remain nil")
	}
	if ad.UnpublishingTime == nil || *ad.UnpublishingTime != 1719999999999 {
		t.Fatalf("unexpected unpublishingTime: %#v", ad.UnpublishingTime)
	}
	if ad.OpenBiddingLatestOffer == nil || *ad.OpenBiddingLatestOffer != 250000 {
		t.Fatalf("unexpected openBiddingLatestOffer: %#v", ad.OpenBiddingLatestOffer)
	}
	if ad.AnnouncementContactInfo == nil || !ad.AnnouncementContactInfo.IsPremiumRealtor {
		t.Fatal("expected premium realtor flag")
	}
	if ad.ProductEffects == nil || ad.ProductEffects.LargerImages == nil || !ad.ProductEffects.LargerImages.VideoEnabled {
		t.Fatal("expected larger image product effects")
	}
	if len(ad.Property.PeriodicCharges) != 1 || ad.Property.PeriodicCharges[0].IncludedInOverallCost == nil || !*ad.Property.PeriodicCharges[0].IncludedInOverallCost {
		t.Fatalf("unexpected periodic charges: %#v", ad.Property.PeriodicCharges)
	}
}

func TestDecodeAdRaw(t *testing.T) {
	payload, err := DecodeAdRaw([]byte(`{"friendlyId":"ad-1","property":{"geoCode":{"latitude":60.1}}}`))
	if err != nil {
		t.Fatalf("decode raw ad: %v", err)
	}
	if payload["friendlyId"] != "ad-1" {
		t.Fatalf("unexpected friendlyId: %#v", payload["friendlyId"])
	}
}

func TestDecodeStoredAd(t *testing.T) {
	ad, raw, err := DecodeStoredAd([]byte(`{"id":1,"friendlyId":"ad-1","status":"PUBLISHED","publishingTime":1710000000000,"property":{"id":2,"specificType":"APARTMENT","propertyType":"APARTMENT","country":{"code":"fi","defaultName":"Finland"}},"preparsed":{"title":"title"}}`))
	if err != nil {
		t.Fatalf("decode stored ad: %v", err)
	}
	if ad.FriendlyID != "ad-1" {
		t.Fatalf("unexpected typed friendlyId: %q", ad.FriendlyID)
	}
	if raw["friendlyId"] != "ad-1" {
		t.Fatalf("unexpected raw friendlyId: %#v", raw["friendlyId"])
	}
}
