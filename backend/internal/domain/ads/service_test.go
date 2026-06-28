package ads

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeSearchParamsDefaults(t *testing.T) {
	normalized := normalizeSearchParams(SearchParams{})
	if normalized.Source != "all" {
		t.Fatalf("expected source all, got %s", normalized.Source)
	}
	if normalized.Kind != "all" {
		t.Fatalf("expected kind all, got %s", normalized.Kind)
	}
	if normalized.Sort != "seen_desc" {
		t.Fatalf("expected sort seen_desc, got %s", normalized.Sort)
	}
	if normalized.Page != 1 {
		t.Fatalf("expected page 1, got %d", normalized.Page)
	}
	if normalized.PageSize != 50 {
		t.Fatalf("expected page size 50, got %d", normalized.PageSize)
	}
}

func TestNormalizePageSizeAllowedValues(t *testing.T) {
	cases := []int32{25, 50, 100}
	for _, c := range cases {
		got := normalizePageSize(c)
		if got != c {
			t.Fatalf("expected %d, got %d", c, got)
		}
	}
}

func TestNormalizeSortFallback(t *testing.T) {
	got := normalizeSort("unknown")
	if got != "seen_desc" {
		t.Fatalf("expected seen_desc, got %s", got)
	}
}

func TestNormalizeAddressLookupInputParsesPastedAddress(t *testing.T) {
	address, city, postal := normalizeAddressLookupInput("Askvagen 4, 22100 Maarianhamina", "", "")
	if address != "Askvagen 4" {
		t.Fatalf("expected street address, got %s", address)
	}
	if city != "Maarianhamina" {
		t.Fatalf("expected city Maarianhamina, got %s", city)
	}
	if postal != "22100" {
		t.Fatalf("expected postal 22100, got %s", postal)
	}
}

func TestNormalizeAddressLookupInputParsesCommaSeparatedPostalCity(t *testing.T) {
	address, city, postal := normalizeAddressLookupInput("Askvagen 4, Maarianhamina, 22100", "", "")
	if address != "Askvagen 4" {
		t.Fatalf("expected street address, got %s", address)
	}
	if city != "Maarianhamina" {
		t.Fatalf("expected city Maarianhamina, got %s", city)
	}
	if postal != "22100" {
		t.Fatalf("expected postal 22100, got %s", postal)
	}
}

func TestNormalizeAddressLookupInputParsesAddressWithPostalOnly(t *testing.T) {
	address, city, postal := normalizeAddressLookupInput("Askvagen 4, 22100", "", "")
	if address != "Askvagen 4" {
		t.Fatalf("expected street address, got %s", address)
	}
	if city != "" {
		t.Fatalf("expected empty city, got %s", city)
	}
	if postal != "22100" {
		t.Fatalf("expected postal 22100, got %s", postal)
	}
	address, city, postal = normalizeAddressLookupInput("Askvagen 4 22100", "", "")
	if address != "Askvagen 4" {
		t.Fatalf("expected street address, got %s", address)
	}
	if city != "" {
		t.Fatalf("expected empty city, got %s", city)
	}
	if postal != "22100" {
		t.Fatalf("expected postal 22100, got %s", postal)
	}
}

func TestDecodeRawTransactionMatches(t *testing.T) {
	score := int32(128)
	matches, err := decodeRawTransactionMatches(json.RawMessage(`[{"type":"listing","id":"11111111-1111-1111-1111-111111111111","canonical_id":"frontdoor:ad:1","source":"frontdoor","native_id":"1","headline":"Askvägen 4","status":"auto_linked","score":128},{"type":"offering_source","id":"22222222-2222-2222-2222-222222222222:33333333-3333-3333-3333-333333333333","canonical_id":"shortcut:ad:2","source":"shortcut","native_id":"2","headline":"Askvägen 4 B","status":"auto_linked","method":"offering_source_listing","score":91}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	match := matches[0]
	if match.Type != "listing" || match.CanonicalID != "frontdoor:ad:1" || match.Score == nil || *match.Score != score {
		t.Fatalf("unexpected match: %+v", match)
	}
	offering := matches[1]
	if offering.Type != "offering_source" || offering.ID != "22222222-2222-2222-2222-222222222222:33333333-3333-3333-3333-333333333333" || offering.CanonicalID != "shortcut:ad:2" || offering.Method != "offering_source_listing" || offering.Score == nil || *offering.Score != 91 {
		t.Fatalf("unexpected offering match: %+v", offering)
	}
	empty, err := decodeRawTransactionMatches(json.RawMessage(`null`))
	if err != nil {
		t.Fatalf("unexpected null error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty matches, got %d", len(empty))
	}
}

func TestAddressMatchReasonSummaryFormatsCommonReasons(t *testing.T) {
	summary := addressMatchReasonSummary(json.RawMessage(`{"postal":"22100","area":{"listing":54.5,"transaction":55},"layout":{"code":"exact","listing":"2h+k","transaction":"2h+k"},"score":{"total":128},"layout_prefix":true}`))
	expected := []string{"Postal 22100", "Area 54.5 / 55", "Layout exact 2h+k / 2h+k", "Layout prefix matched", "Score total 128"}
	if len(summary) != len(expected) {
		t.Fatalf("expected %d summary items, got %d: %+v", len(expected), len(summary), summary)
	}
	for i, item := range expected {
		if summary[i] != item {
			t.Fatalf("expected summary[%d] %q, got %q", i, item, summary[i])
		}
	}
}

func TestAddressMatchReasonSummaryFormatsSourceReasons(t *testing.T) {
	summary := addressMatchReasonSummary(json.RawMessage(`{"source_provider":"shortcut","target_provider":"frontdoor","postal":{"source":"22100","target":"22100"},"address":{"source":"askvagen 4","target":"askvagen 4"},"unit_match_key":{"source":"4:g","target":"4:g"},"area":{"source":54.5,"target":55},"layout":{"source":"2h+k","target":"2h+k"},"score":{"postal":10,"address":25,"unit":35,"area":8,"layout":6,"price":0}}`))
	expected := []string{"Sources shortcut / frontdoor", "Postal 22100 / 22100", "Address askvagen 4 / askvagen 4", "Unit key 4:g / 4:g", "Area 54.5 / 55", "Layout 2h+k / 2h+k", "Score address 25, unit 35, area 8, layout 6"}
	if len(summary) != len(expected) {
		t.Fatalf("expected %d summary items, got %d: %+v", len(expected), len(summary), summary)
	}
	for i, item := range expected {
		if summary[i] != item {
			t.Fatalf("expected summary[%d] %q, got %q", i, item, summary[i])
		}
	}
}

func TestNormalizeAddressLookupInputPreservesExplicitFilters(t *testing.T) {
	_, city, postal := normalizeAddressLookupInput("Askvagen 4, 22100 Maarianhamina", "Mariehamn", "22101")
	if city != "Mariehamn" {
		t.Fatalf("expected explicit city Mariehamn, got %s", city)
	}
	if postal != "22101" {
		t.Fatalf("expected explicit postal 22101, got %s", postal)
	}
}

func TestBuildAddressLookupResultGroupsListingsAndDeduplicatesTransactions(t *testing.T) {
	listingID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	offeringID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	transactionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	price := int64(132000)
	previousPrice := int64(145000)
	score := int32(128)
	rows := []addressLookupRow{
		{ListingID: listingID, CanonicalID: "frontdoor:ad:21531967", Source: "frontdoor", Kind: "ad", NativeID: "21531967", Headline: "Askvägen 4", Address: "Askvägen 4", City: "Maarianhamina", Postal: "22100", PreviousAskingPrice: &previousPrice, PriceMatchStatus: "auto_linked", SourceMatchStatus: "auto_linked", OfferingID: &offeringID, RenovationsDoneText: "Roof 2020", TransactionID: &transactionID, LinkType: "direct", LinkStatus: "auto_linked", LinkMethod: "source_listing", Score: &score, Confidence: "high", PriceDeltaPercent: nil, Reasons: []byte(`{"postal":"22100"}`), TransactionDescription: "3 r, k", TransactionPrice: &price},
		{ListingID: listingID, CanonicalID: "frontdoor:ad:21531967", Source: "frontdoor", Kind: "ad", NativeID: "21531967", Headline: "Askvägen 4", Address: "Askvägen 4", City: "Maarianhamina", Postal: "22100", PreviousAskingPrice: &previousPrice, PriceMatchStatus: "auto_linked", SourceMatchStatus: "auto_linked", OfferingID: &offeringID, RenovationsDoneText: "Roof 2020", TransactionID: &transactionID, LinkType: "direct", LinkStatus: "auto_linked", LinkMethod: "source_listing", Score: &score, Confidence: "high", PriceDeltaPercent: nil, Reasons: []byte(`{"postal":"22100"}`), TransactionDescription: "3 r, k", TransactionPrice: &price},
	}
	result := buildAddressLookupResult("Askvägen 4", "Maarianhamina", "22100", "all", rows)
	if len(result.Listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(result.Listings))
	}
	listing := result.Listings[0]
	if listing.OfferingID != offeringID.String() {
		t.Fatalf("expected offering id %s, got %s", offeringID, listing.OfferingID)
	}
	if listing.PreviousAskingPrice == nil || *listing.PreviousAskingPrice != previousPrice {
		t.Fatalf("expected previous asking price %d, got %v", previousPrice, listing.PreviousAskingPrice)
	}
	if listing.PriceMatchStatus != "auto_linked" || listing.SourceMatchStatus != "auto_linked" {
		t.Fatalf("unexpected match statuses: %+v", listing)
	}
	if len(listing.SourceRecords) != 1 {
		t.Fatalf("expected 1 source record, got %d", len(listing.SourceRecords))
	}
	if listing.Texts == nil || listing.Texts.RenovationsDone != "Roof 2020" {
		t.Fatalf("expected listing text evidence, got %+v", listing.Texts)
	}
	if len(listing.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(listing.Transactions))
	}
	transaction := listing.Transactions[0]
	if transaction.TransactionID != transactionID.String() {
		t.Fatalf("expected transaction id %s, got %s", transactionID, transaction.TransactionID)
	}
	if transaction.LinkType != "direct" || transaction.LinkStatus != "auto_linked" || transaction.Confidence != "high" {
		t.Fatalf("unexpected transaction link: %+v", transaction)
	}
	if transaction.ReasonsSummary == nil {
		t.Fatal("expected transaction reasons summary")
	}
}

func TestBuildAddressLookupResultKeepsUnmatchedListings(t *testing.T) {
	listingID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	rows := []addressLookupRow{
		{ListingID: listingID, CanonicalID: "shortcut:ad:24063710", Source: "shortcut", Kind: "ad", NativeID: "24063710", Headline: "Askvägen 4 G", Address: "Askvägen 4 G", City: "Maarianhamina", Postal: "22100"},
	}
	result := buildAddressLookupResult("Askvägen 4", "Maarianhamina", "22100", "all", rows)
	if len(result.Listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(result.Listings))
	}
	listing := result.Listings[0]
	if listing.CanonicalID != "shortcut:ad:24063710" {
		t.Fatalf("expected shortcut listing, got %s", listing.CanonicalID)
	}
	if listing.Transactions == nil {
		t.Fatal("expected empty transactions slice, got nil")
	}
	if listing.SourceRecords == nil {
		t.Fatal("expected empty source records slice, got nil")
	}
	if listing.SourceCandidates == nil {
		t.Fatal("expected empty source candidates slice, got nil")
	}
	if len(listing.Transactions) != 0 {
		t.Fatalf("expected no transactions, got %d", len(listing.Transactions))
	}
}

func TestBuildAddressLookupResultUsesOfferingSourceRecords(t *testing.T) {
	listingID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	sourceRecordID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	offeringID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	askingPrice := int64(198000)
	previousPrice := int64(205000)
	area := 54.5
	linkScore := int32(91)
	rows := []addressLookupRow{
		{ListingID: listingID, CanonicalID: "frontdoor:ad:21531967", Source: "frontdoor", Kind: "ad", NativeID: "21531967", Headline: "Askvägen 4", Address: "Askvägen 4", City: "Maarianhamina", Postal: "22100", OfferingID: &offeringID, SourceRecordListingID: &sourceRecordID, SourceRecordCanonicalID: "shortcut:ad:24063710", SourceRecordSource: "shortcut", SourceRecordKind: "ad", SourceRecordNativeID: "24063710", SourceRecordHeadline: "Askvägen 4 G", SourceRecordAddress: "Askvägen 4 G", SourceRecordCity: "Maarianhamina", SourceRecordPostal: "22100", SourceRecordAskingPrice: &askingPrice, SourceRecordArea: &area, SourceRecordRoomLayout: "2h+k", SourceRecordURL: "https://example.test/listing", SourceRecordPreviousAsk: &previousPrice, SourceRecordLinkStatus: "auto_linked", SourceRecordLinkMethod: "source_match_auto", SourceRecordLinkScore: &linkScore, SourceRecordRenovationsPlan: "Facade planned"},
	}
	result := buildAddressLookupResult("Askvägen 4", "Maarianhamina", "22100", "all", rows)
	if len(result.Listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(result.Listings))
	}
	sourceRecords := result.Listings[0].SourceRecords
	if len(sourceRecords) != 1 {
		t.Fatalf("expected 1 source record, got %d", len(sourceRecords))
	}
	record := sourceRecords[0]
	if record.ListingID != sourceRecordID.String() {
		t.Fatalf("expected explicit source record id %s, got %s", sourceRecordID, record.ListingID)
	}
	if record.CanonicalID != "shortcut:ad:24063710" || record.Source != "shortcut" || record.NativeID != "24063710" {
		t.Fatalf("unexpected source record: %+v", record)
	}
	if record.Address != "Askvägen 4 G" || record.City != "Maarianhamina" || record.Postal != "22100" {
		t.Fatalf("unexpected source record address: %+v", record)
	}
	if record.AskingPrice == nil || *record.AskingPrice != askingPrice {
		t.Fatalf("expected source asking price %d, got %v", askingPrice, record.AskingPrice)
	}
	if record.Area == nil || *record.Area != area {
		t.Fatalf("expected source area %v, got %v", area, record.Area)
	}
	if record.PreviousAskingPrice == nil || *record.PreviousAskingPrice != previousPrice {
		t.Fatalf("expected previous source asking price %d, got %v", previousPrice, record.PreviousAskingPrice)
	}
	if record.Texts == nil || record.Texts.RenovationsPlanned != "Facade planned" {
		t.Fatalf("expected source record text evidence, got %+v", record.Texts)
	}
	if record.LinkStatus != "auto_linked" || record.LinkMethod != "source_match_auto" || record.LinkScore == nil || *record.LinkScore != linkScore {
		t.Fatalf("unexpected source record link provenance: %+v", record)
	}
}

func TestBuildAddressLookupResultKeepsSourceRecordTransactionProvenance(t *testing.T) {
	listingID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	offeringID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	transactionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	rows := []addressLookupRow{
		{ListingID: listingID, CanonicalID: "shortcut:ad:23442274", Source: "shortcut", Kind: "ad", NativeID: "23442274", Headline: "Askvägen 4 C", Address: "Askvägen 4 C", City: "Maarianhamina", Postal: "22100", OfferingID: &offeringID, TransactionID: &transactionID, LinkType: "source_record", LinkStatus: "auto_linked", LinkMethod: "offering_source_listing", TransactionDescription: "3 r, k"},
	}
	result := buildAddressLookupResult("Askvägen 4", "Maarianhamina", "22100", "all", rows)
	if len(result.Listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(result.Listings))
	}
	transactions := result.Listings[0].Transactions
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}
	transaction := transactions[0]
	if transaction.LinkType != "source_record" || transaction.LinkMethod != "offering_source_listing" {
		t.Fatalf("unexpected transaction provenance: %+v", transaction)
	}
}

func TestAppendAddressSourceCandidateRowsDeduplicatesCandidates(t *testing.T) {
	selectedID := uuid.MustParse("12121212-1212-1212-1212-121212121212")
	candidateID := uuid.MustParse("34343434-3434-3434-3434-343434343434")
	selectedOfferingID := uuid.MustParse("56565656-5656-5656-5656-565656565656")
	candidateOfferingID := uuid.MustParse("78787878-7878-7878-7878-787878787878")
	priceDelta := 0.05
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	result := AddressLookupResult{
		Listings: []AddressListing{
			{ListingID: selectedID.String(), SourceCandidates: []AddressSourceCandidate{}},
		},
	}
	rows := []addressSourceCandidateRow{
		{SelectedListingID: selectedID, CandidateListingID: candidateID, CanonicalID: "shortcut:ad:24063710", Source: "shortcut", Kind: "ad", NativeID: "24063710", Headline: "Askvägen 4 G", Address: "Askvägen 4 G", City: "Maarianhamina", Postal: "22100", SelectedOfferingID: selectedOfferingID, CandidateOfferingID: candidateOfferingID, Direction: "source_to_target", Status: "candidate", Score: 118, Confidence: "high", PriceDeltaPercent: &priceDelta, Reasons: []byte(`{"postal":"22100"}`), CreatedAt: &createdAt},
		{SelectedListingID: selectedID, CandidateListingID: candidateID, CanonicalID: "shortcut:ad:24063710", Source: "shortcut", Kind: "ad", NativeID: "24063710", Headline: "Askvägen 4 G", SelectedOfferingID: selectedOfferingID, CandidateOfferingID: candidateOfferingID, Direction: "source_to_target", Status: "candidate", Score: 118, Confidence: "high"},
	}
	appendAddressSourceCandidateRows(&result, map[uuid.UUID]int{selectedID: 0}, rows)
	candidates := result.Listings[0].SourceCandidates
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	candidate := candidates[0]
	if candidate.ListingID != candidateID.String() || candidate.CanonicalID != "shortcut:ad:24063710" {
		t.Fatalf("unexpected candidate identity: %+v", candidate)
	}
	if candidate.SelectedOfferingID != selectedOfferingID.String() || candidate.CandidateOfferingID != candidateOfferingID.String() {
		t.Fatalf("unexpected offering provenance: %+v", candidate)
	}
	if candidate.Status != "candidate" || candidate.Score != 118 || candidate.Confidence != "high" {
		t.Fatalf("unexpected candidate match data: %+v", candidate)
	}
	if candidate.PriceDeltaPercent == nil || *candidate.PriceDeltaPercent != priceDelta {
		t.Fatalf("expected price delta %v, got %v", priceDelta, candidate.PriceDeltaPercent)
	}
	if len(candidate.ReasonsSummary) != 1 || candidate.ReasonsSummary[0] != "Postal 22100" {
		t.Fatalf("unexpected candidate reason summary: %+v", candidate.ReasonsSummary)
	}
	if candidate.CreatedAt == nil || !candidate.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created at %v, got %v", createdAt, candidate.CreatedAt)
	}
}

func TestRawTransactionLocationFallsBackToListingLocation(t *testing.T) {
	result := AddressLookupResult{
		Listings: []AddressListing{
			{City: "Maarianhamina", Postal: "22100"},
		},
	}
	city, postal := rawTransactionLocation(result)
	if city != "Maarianhamina" {
		t.Fatalf("expected fallback city Maarianhamina, got %s", city)
	}
	if postal != "22100" {
		t.Fatalf("expected fallback postal 22100, got %s", postal)
	}
}

func TestLinkedTransactionIDsDeduplicatesLookupLinks(t *testing.T) {
	transactionID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	result := AddressLookupResult{
		Listings: []AddressListing{
			{Transactions: []AddressTransactionLink{{TransactionID: transactionID, LinkType: "direct"}, {TransactionID: transactionID, LinkType: "direct"}}},
			{Transactions: []AddressTransactionLink{{TransactionID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", LinkType: "candidate", LinkStatus: "candidate"}}},
			{Transactions: []AddressTransactionLink{{TransactionID: "not-a-uuid", LinkType: "offering"}}},
		},
	}
	ids := linkedTransactionIDs(result)
	if len(ids) != 1 {
		t.Fatalf("expected 1 linked transaction id, got %d", len(ids))
	}
	if ids[0].String() != transactionID {
		t.Fatalf("expected linked transaction id %s, got %s", transactionID, ids[0])
	}
}

func TestCandidateTransactionIDsDeduplicatesLookupCandidates(t *testing.T) {
	transactionID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	result := AddressLookupResult{
		Listings: []AddressListing{
			{Transactions: []AddressTransactionLink{{TransactionID: transactionID, LinkType: "candidate"}, {TransactionID: transactionID, LinkType: "candidate"}}},
			{Transactions: []AddressTransactionLink{{TransactionID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", LinkType: "direct"}}},
			{Transactions: []AddressTransactionLink{{TransactionID: "not-a-uuid", LinkType: "candidate"}}},
		},
	}
	ids := candidateTransactionIDs(result)
	if len(ids) != 1 {
		t.Fatalf("expected 1 candidate transaction id, got %d", len(ids))
	}
	if ids[0].String() != transactionID {
		t.Fatalf("expected candidate transaction id %s, got %s", transactionID, ids[0])
	}
}

func TestRawTransactionScope(t *testing.T) {
	cases := []struct {
		name              string
		linkedToLookup    bool
		candidateToLookup bool
		isMatched         bool
		want              string
	}{
		{name: "linked", linkedToLookup: true, candidateToLookup: true, isMatched: true, want: "linked_here"},
		{name: "candidate", candidateToLookup: true, isMatched: true, want: "candidate_here"},
		{name: "matched elsewhere", isMatched: true, want: "matched_elsewhere"},
		{name: "postal history", want: "postal_history"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := rawTransactionScope(tt.linkedToLookup, tt.candidateToLookup, tt.isMatched)
			if got != tt.want {
				t.Fatalf("expected scope %s, got %s", tt.want, got)
			}
		})
	}
}

func TestAddressRawTransactionsExposeOfferingSourceMatches(t *testing.T) {
	for _, want := range []string{
		"'offering_source'::text AS match_type",
		"pot.property_offering_transaction_id::text || ':' || sl.sale_listing_id::text AS id",
		"JOIN public.property_offering_sources pos ON pos.property_offering_id = pot.property_offering_id",
		"JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id",
	} {
		if !strings.Contains(addressRawTransactionsSQL, want) {
			t.Fatalf("expected raw transaction SQL to include %q", want)
		}
	}
	if strings.Contains(addressRawTransactionsSQL, "primary_listing.sale_listing_canonical_id") {
		t.Fatal("expected raw transaction matches to use offering source rows instead of only the primary listing")
	}
}
