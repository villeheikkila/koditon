# Typed Housing Company Data Model

Scope: Helsinki apartment listings in housing-company block buildings. Detached houses, semi-detached houses, row houses, and generic property modeling are explicitly out of scope for this pass.

## Asset Class Boundary

This model is the housing-company apartment module. Detached houses, semi-detached houses, and row houses should get separate typed modules later because their risk model is different: roof, foundation, drainage, plot, moisture, heating, and inspection evidence matter more than housing-company systems, debt, and charges.

Shared infrastructure can include raw provider storage, canonical source matching, transactions, LLM extraction plumbing, and lightweight `field_sources` provenance. Typed profile tables, renovation categories, lifecycle rules, and valuation/risk math should remain asset-class-specific.

## Goal

Build a typed building-centered model that can answer:

- What housing company/building is this listing part of?
- What do we currently believe about the building systems and major renovations?
- What apartment/listing facts matter for valuation?
- Which source produced each important value?
- Which facts are missing, stale, or conflicting?
- When is ownership likely to become expensive?

The main app surface should read typed profiles and typed risk outputs. Generic extracted facts are allowed as evidence, but not as the primary app model.

## Data Flow

```text
raw provider records
-> deterministic provider canonicalization
-> typed housing company and listing profile tables
-> manual LLM extraction for targeted gaps
-> typed profile updates with field_sources provenance
-> valuation and building risk snapshots
```

## Existing Tables To Keep

- `property_source_offerings`: sale listing canonical/source row.
- provider raw tables such as `shortcut_ads`, `frontdoor_ads`, `frontdoor_building_announcements`.
- transaction tables and listing transaction matching.
- current source/merge/linking tables.

Do not replace raw provider storage. It remains the audit trail and reprocessing source.

## New Typed Core

### Existing `housing_companies`

Durable housing-company identity.

The repository already has this table with identity keys, normalized address fields, business ID, basic building facts, geometry, source links, merge decisions, and generic `housing_company_facts`. The typed model should reuse that identity table instead of creating a parallel `housing_companies` table.

### Existing `property_units`

The repository already links canonical offerings to units and units to housing companies through `property_units`, `property_offerings`, and `property_offering_sources`. The typed apartment profile should use those links when available.

Do not add a new physical building table until the product needs multi-building company modeling. For now, company-level systems and renovations are enough for Helsinki block-building assessment.

### `housing_company_systems`

Current belief about major building systems.

```sql
CREATE TABLE public.housing_company_systems (
    housing_company_system_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_system_type text NOT NULL,
    housing_company_system_status text NOT NULL DEFAULT 'unknown',
    housing_company_system_last_renovated_year int,
    housing_company_system_next_expected_start_year int,
    housing_company_system_next_expected_end_year int,
    housing_company_system_confidence text NOT NULL DEFAULT 'low',
    housing_company_system_evidence_level text NOT NULL DEFAULT 'none',
    housing_company_system_primary_source_id uuid,
    housing_company_system_summary text,
    housing_company_system_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (housing_company_system_type = ANY (ARRAY['pipes','water_supply','sewer','roof','facade','windows','balconies','elevator','heating','ventilation','drainage','electrical','yard','common_areas']::text[])),
    CHECK (housing_company_system_status = ANY (ARRAY['unknown','original','maintained','partly_renewed','renewed','under_study','planned','under_construction','risk']::text[])),
    CHECK (housing_company_system_confidence = ANY (ARRAY['low','medium','high']::text[])),
    CHECK (housing_company_system_evidence_level = ANY (ARRAY['none','ad_only','multiple_ads','announcement','manager_certificate','financial_statement','manual']::text[]))
);
CREATE UNIQUE INDEX idx_housing_company_systems_unique ON public.housing_company_systems (housing_company_id, housing_company_system_type);
```

### `housing_company_renovations`

Typed renovation events and forecasted renovation needs.

```sql
CREATE TABLE public.housing_company_renovations (
    housing_company_renovation_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_renovation_category text NOT NULL,
    housing_company_renovation_status text NOT NULL,
    housing_company_renovation_stage text NOT NULL DEFAULT 'unknown',
    housing_company_renovation_scope text NOT NULL DEFAULT 'unknown',
    housing_company_renovation_responsibility text NOT NULL DEFAULT 'unknown',
    housing_company_renovation_year int,
    housing_company_renovation_window_start_year int,
    housing_company_renovation_window_end_year int,
    housing_company_renovation_cost_estimate_eur bigint,
    housing_company_renovation_confidence text NOT NULL DEFAULT 'low',
    housing_company_renovation_evidence_level text NOT NULL DEFAULT 'none',
    housing_company_renovation_primary_source_id uuid,
    housing_company_renovation_summary text,
    housing_company_renovation_created_at timestamptz DEFAULT now() NOT NULL,
    housing_company_renovation_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (housing_company_renovation_category = ANY (ARRAY['pipe','water_supply','sewer','roof','facade','window','balcony','elevator','heating','ventilation','drainage','electrical','yard','common_areas','other']::text[])),
    CHECK (housing_company_renovation_status = ANY (ARRAY['done','planned','suspected','forecast','cancelled','unknown']::text[])),
    CHECK (housing_company_renovation_stage = ANY (ARRAY['unknown','study','condition_assessment','planning','tendering','execution','completed']::text[])),
    CHECK (housing_company_renovation_scope = ANY (ARRAY['unknown','full','partial','maintenance']::text[])),
    CHECK (housing_company_renovation_responsibility = ANY (ARRAY['unknown','housing_company','shareholder','mixed']::text[])),
    CHECK (housing_company_renovation_confidence = ANY (ARRAY['low','medium','high']::text[])),
    CHECK (housing_company_renovation_evidence_level = ANY (ARRAY['none','ad_only','multiple_ads','announcement','manager_certificate','financial_statement','manual']::text[]))
);
CREATE INDEX idx_housing_company_renovations_company ON public.housing_company_renovations (housing_company_id);
CREATE INDEX idx_housing_company_renovations_timing ON public.housing_company_renovations (housing_company_renovation_status, housing_company_renovation_year, housing_company_renovation_window_start_year);
```

## Listing/Apartment Profile Tables

Listings are market events. Apartment profile data can be listing-derived until we have reliable unit identity.

### `sale_listing_apartment_profiles`

```sql
CREATE TABLE public.sale_listing_apartment_profiles (
    sale_listing_id uuid NOT NULL PRIMARY KEY REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    property_unit_id uuid REFERENCES public.property_units(property_unit_id) ON DELETE SET NULL,
    apartment_profile_area_m2 double precision,
    apartment_profile_living_area_m2 double precision,
    apartment_profile_room_layout text,
    apartment_profile_room_count int,
    apartment_profile_bedroom_count int,
    apartment_profile_floor_level int,
    apartment_profile_total_floors int,
    apartment_profile_kitchen_type text,
    apartment_profile_layout_quality text,
    apartment_profile_awkward_layout boolean,
    apartment_profile_condition text,
    apartment_profile_kitchen_condition text,
    apartment_profile_bathroom_condition text,
    apartment_profile_surface_renovation_need boolean,
    apartment_profile_modernization_need boolean,
    apartment_profile_sauna boolean,
    apartment_profile_balcony boolean,
    apartment_profile_balcony_glazing boolean,
    apartment_profile_parking_type text,
    apartment_profile_storage_quality text,
    apartment_profile_view_quality text,
    apartment_profile_noise_risk boolean,
    apartment_profile_accessibility text,
    apartment_profile_confidence text NOT NULL DEFAULT 'low',
    apartment_profile_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (apartment_profile_kitchen_type IS NULL OR apartment_profile_kitchen_type = ANY (ARRAY['separate','open','kitchenette','unknown']::text[])),
    CHECK (apartment_profile_layout_quality IS NULL OR apartment_profile_layout_quality = ANY (ARRAY['weak','average','good','excellent','unknown']::text[])),
    CHECK (apartment_profile_condition IS NULL OR apartment_profile_condition = ANY (ARRAY['poor','fair','good','excellent','new','unknown']::text[])),
    CHECK (apartment_profile_parking_type IS NULL OR apartment_profile_parking_type = ANY (ARRAY['none','street','yard','garage','carport','separate_share','unknown']::text[])),
    CHECK (apartment_profile_storage_quality IS NULL OR apartment_profile_storage_quality = ANY (ARRAY['weak','normal','good','unknown']::text[])),
    CHECK (apartment_profile_view_quality IS NULL OR apartment_profile_view_quality = ANY (ARRAY['weak','normal','good','excellent','unknown']::text[])),
    CHECK (apartment_profile_accessibility IS NULL OR apartment_profile_accessibility = ANY (ARRAY['poor','average','good','unknown']::text[])),
    CHECK (apartment_profile_confidence = ANY (ARRAY['low','medium','high']::text[]))
);
CREATE INDEX idx_sale_listing_apartment_profiles_company ON public.sale_listing_apartment_profiles (housing_company_id);
```

## Lightweight Provenance

Use provenance as sidecar metadata for typed fields. Do not make it the main domain model.

### `field_sources`

```sql
CREATE TABLE public.field_sources (
    field_source_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    field_source_target_table text NOT NULL,
    field_source_target_id uuid NOT NULL,
    field_source_target_field text NOT NULL,
    field_source_source_record_table text NOT NULL,
    field_source_source_record_id uuid NOT NULL,
    field_source_source_path text,
    field_source_evidence_text text,
    field_source_method text NOT NULL,
    field_source_confidence double precision NOT NULL DEFAULT 1,
    field_source_observed_at timestamptz DEFAULT now() NOT NULL,
    field_source_valid_from date,
    field_source_valid_until date,
    CHECK (field_source_method = ANY (ARRAY['provider_field','llm','manual','parser','forecast']::text[]))
);
CREATE INDEX idx_field_sources_target ON public.field_sources (field_source_target_table, field_source_target_id, field_source_target_field);
CREATE INDEX idx_field_sources_source ON public.field_sources (field_source_source_record_table, field_source_source_record_id);
```

For provider JSON, store:

```text
source_record_table=property_source_offerings
source_record_id=<sale_listing_id>
source_path=sale_listing_area_value
method=provider_field
```

For LLM values, store:

```text
source_record_table=property_source_offerings
source_record_id=<sale_listing_id>
source_path=sale_listing_balcony_description_text
evidence_text=lasitettu parveke
method=llm
```

Later document ingestion can add `document_sources`, `document_pages`, and `document_cells`; `field_sources` can then point to document rows/cells without changing profile tables.

## Service Model

Add a new backend package or submodule around the existing properties domain:

```text
internal/domain/properties/housingcompany
```

Core service responsibilities:

- Link sale listings to `housing_companies`.
- Project provider fields into `sale_listing_apartment_profiles`.
- Project building fields into existing `housing_companies` and typed `housing_company_systems`.
- Project renovations into `housing_company_systems` and `housing_company_renovations`.
- Write `field_sources` for values that appear in typed profile rows.
- Produce building dossier read models for API/UI.

Keep current `valuation` package as calculation/read-model code. Its inputs should eventually come from typed profiles instead of generic `ValuationInputs`.

## Projection Rules

### Provider fields

Provider structured values can write directly to typed profiles with `field_sources.method = provider_field`.

Examples:

- `sale_listing_area_value` -> `sale_listing_apartment_profiles.apartment_profile_area_m2`
- `sale_listing_floor_level` -> `apartment_profile_floor_level`
- `sale_listing_total_floors` -> `apartment_profile_total_floors`
- `sale_listing_sauna` -> `apartment_profile_sauna`
- `sale_listing_balcony` -> `apartment_profile_balcony`
- `sale_listing_build_year` -> existing `housing_companies.housing_company_build_year`
- `sale_listing_energy_class` -> existing `housing_companies.housing_company_energy_efficiency_label`

### LLM extracted values

LLM extraction should produce a typed patch, not only generic facts.

Initial patch shape:

```go
type ApartmentProfilePatch struct {
    BalconyGlazing *bool
    KitchenType *string
    LayoutQuality *string
    AwkwardLayout *bool
    KitchenCondition *string
    BathroomCondition *string
    SurfaceRenovationNeed *bool
    ModernizationNeed *bool
    StorageQuality *string
    ViewQuality *string
    NoiseRisk *bool
}
```

Each populated field must write a `field_sources` row with the source text field and evidence phrase.

### Renovations

Renovations should land in typed tables:

- Completed concrete work -> `housing_company_renovations.status = done`
- Planned work from listing text -> `status = planned`
- Lifecycle-derived future work -> `status = forecast`
- System status summary -> `housing_company_systems`

The existing declarative renovation rule catalog can remain in Go, but it should output typed `housing_company_renovations` rows and a building risk snapshot.

## Read Models

### `BuildingDossier`

API shape:

```go
type BuildingDossier struct {
    HousingCompany HousingCompanySummary
    Buildings []HousingCompanyBuilding
    Systems []HousingCompanySystem
    Renovations []HousingCompanyRenovation
    Listings []SaleListingSummary
    Transactions []TransactionSummary
    Risk BuildingRiskSummary
    Missing []string
    Conflicts []string
}
```

### `ListingAssessment`

Inputs:

- `sale_listing_apartment_profiles`
- linked `housing_company_dossier`
- transaction match
- current sale listing price/charges

Outputs:

- offer assessment
- ownership cost windows
- top risks/supports
- missing evidence

## Implementation Phases

### Phase 1: typed tables

- Add migrations for `housing_company_systems`, `housing_company_renovations`, `sale_listing_apartment_profiles`, and `field_sources`.
- Add sqlc queries for upserts and reads.
- Backfill from `property_source_offerings` and existing structured renovations.
- Keep existing valuation UI behavior unchanged.

### Phase 2: typed projection service

- Add `ProjectSaleListingProfile(ctx, saleListingID)`.
- Upsert apartment profile fields from provider canonical columns.
- Upsert housing company/building identity from listing/building fields.
- Write `field_sources` for projected provider fields.
- Add tests using one Shortcut-like and one Frontdoor-like listing row.

### Phase 3: LLM typed patches

- Change valuation input extraction to produce `ApartmentProfilePatch`.
- Store patch output into `sale_listing_apartment_profiles`.
- Keep generic `property_valuation_facts` only as optional raw extraction evidence until it can be removed or demoted.
- Write `field_sources.method = llm`.

### Phase 4: building renovations/profile

- Project current structured renovations into `housing_company_renovations`.
- Project lifecycle forecasts into typed forecast rows.
- Maintain `housing_company_systems` summary per major system.
- Add `BuildingDossier` API endpoint.

### Phase 5: listing valuation reads typed profiles

- Update valuation adapter to read apartment profile + building dossier.
- Keep current JSON response shape if possible.
- Make the listing UI show typed profile fields and source drill-down.

## Design Constraints

- Do not generalize for detached houses yet.
- Do not use generic facts as primary read model.
- Keep provider raw data untouched.
- Every non-trivial derived field must have a `field_sources` row.
- Prefer typed columns and controlled enum check constraints over arbitrary key/value rows.
- Keep document table/cell modeling out until actual document ingestion starts.

## Success Criteria

- A listing page can show a typed apartment profile.
- A housing-company page can show typed systems and renovations.
- A typed value can show where it came from.
- Provider fields, LLM fields, and forecast fields are distinguishable.
- Valuation reads typed profiles rather than parsing ad/source text directly.
