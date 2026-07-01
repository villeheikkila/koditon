# Canonical Property Model Plan

This document is a working plan for cleaning up the property data model. It is intentionally an iteration artifact: the goal is to make the source/canonical boundaries clear enough that the plan can later become implementation goals.

## Decisions From Model Review

- Raw provider evidence, price rows, uploaded PDFs, and extraction outputs are the only durable migration inputs.
- Existing computed source, canonical, profile, insight, and link tables are disposable. They may be used for parity checks, but they are not authoritative migration sources.
- The target model should use clean final table and API vocabulary immediately. Compatibility with old computed table names is not a goal unless an external API consumer explicitly requires it.
- All derived-data work triggered by raw data changes must run through Go Absurd workflows. Database triggers must not create, link, resolve, or mutate source-normalized or canonical data.
- Full rebuilds should truncate computed tables and rebuild them from preserved raw evidence. There is no need for shadow tables or computed-table history.
- Canonical IDs should be fresh UUIDs. Do not maintain legacy ID redirects unless a future external API requirement explicitly adds that constraint.
- `Listing` means the canonical sale offering, not the underlying apartment or house.
- A canonical listing may have multiple provider source listings.
- A source listing may back at most one non-rejected canonical listing.
- Provider housing-company pages are first-class sources and should be raw-synced and normalized independently.
- Listing-derived housing-company facts may create or link a housing company, but with lower reliability than provider housing-company pages, manager certificates, or business ID evidence.
- Business ID is strong housing-company evidence, but the model must support heuristic linking and manual delinking because business IDs are often missing.
- A housing company can contain one or more physical buildings, and a physical building can contain units.
- Address is useful matching evidence, especially for apartment buildings, but housing-company identity must not depend only on address.
- Detached manager certificate uploads should be supported by creating document-derived provisional targets that can later be linked or merged.
- Document-derived facts do not overwrite provider facts directly; they become higher-reliability claims that the resolver selects or conflicts.
- Price matches should move into a separate canonical table that links prices rows to listings and supports legacy building-announcement targets.
- Physical table renames are not the goal. The goal is a clean rebuilt computed schema that preserves raw evidence.

## Current Problem

The project has three main data sources:

- `hintatiedot.fi` prices data, currently stored in `prices_transactions`.
- `etuovi.fi` listing and housing-company data, currently under the Frontdoor naming.
- `oikotie.fi` listing and building data, currently under the Shortcut naming.

The raw sync layer is mostly healthy because provider payloads are preserved in sync tables. The model gets unclear after that. Provider-normalized rows, canonical listing rows, matching state, price links, documents, LLM-derived facts, and final read models are all partially implemented, but the names and read paths make it hard to know which layer owns which truth.

The migration should preserve raw evidence and rebuild the computed model. Existing computed tables are useful only as validation fixtures and parity references.

The desired model is provider-agnostic:

- Raw provider and document evidence is immutable.
- Provider-normalized source entities are derived from one raw source.
- Canonical entities have our own UUIDs.
- Resolved facts and insights are projected with provenance.

## Layered Model

### 1. Raw Evidence

Raw evidence stores the data as received. It should not be edited to fit our canonical model.

Existing raw evidence tables:

- `shortcut_ads`
- `shortcut_buildings`
- `frontdoor_ads`
- `frontdoor_building_announcements`
- `prices_transactions`
- `property_documents`
- `property_document_extractions`

Rules:

- Raw provider JSON and uploaded PDFs are retained as evidence.
- Provider payload hash/version fields decide when source normalization must rerun.
- Canonical reads should not parse provider JSON directly.
- Raw evidence is exposed through source links, not embedded into canonical objects by default.

### 2. Source-Normalized Entities

Source entities are normalized projections from one raw source. They preserve provider identity and extracted provider facts, but they are not the final domain model.

Existing table mapping:

- Current `property_source_offerings` is disposable computed state. The replacement is `source_listings`, rebuilt from raw provider evidence.
- Current `housing_company_sources` is disposable computed state. The replacement is split into source-normalized housing-company records and generic target links.
- Future provider-specific housing-company pages should have raw sync tables and then normalize into `source_housing_companies`.

Rules:

- One source listing comes from one provider listing or announcement.
- Source listings keep provider/native IDs, URLs, first/last seen timestamps, raw table references, and normalized source facts.
- Source listing IDs are UUIDs, but they are not public canonical listing IDs.
- Names like `sale_listing_*` are compatibility baggage. The target schema should physically rename them to `source_listing_*` or split them into cleaner source/fact tables.

### 3. Canonical Entities

Canonical entities are the public domain objects with stable UUIDs.

Existing table mapping:

- Current `property_offerings.property_offering_id` is disposable.
- Current `housing_companies.housing_company_id` is disposable.
- Current `property_units`, `physical_buildings`, and `property_houses` are computed state and should be rebuilt into `units`, `physical_buildings`, and `houses`.

Rules:

- Public listing IDs must be our UUIDs, never `frontdoor:...` or `shortcut:...`.
- Provider IDs are source references only.
- A canonical listing can have one or many source listings.
- A canonical housing company can have one or many provider/document/manual sources.
- Canonical objects should survive source re-sync and provider payload changes.
- Target type vocabulary should use `listing`, not `offering`, throughout the new model.

### 4. Derived And Resolved Knowledge

Derived data is produced from sources and documents.

Existing table mapping:

- `property_dimension_claims` stores source/document/manual facts with provenance.
- `property_dimension_values` stores selected resolved values.
- `property_dimension_profiles` stores read-model JSON grouped by target.
- `property_offering_transactions` currently stores canonical listing to prices transaction links.
- `property_source_offering_insights` currently stores source-level insights and should be replaced by target-scoped observations.
- Existing dimension, transaction-link, and insight tables are rebuildable computed state.

Rules:

- The dimension layer is the long-term normalized truth layer.
- Source listing columns can remain extraction inputs and fallback fields.
- Price matches should attach canonically through a dedicated price-link table that can target canonical listings and legacy provider announcements.
- LLM insights should be target-scoped observations with evidence references. A source listing can produce claims or observations about the listing, unit, building, housing company, or house.
- Document-derived facts should generally outrank listing-derived facts for housing-company and building details.

## Canonical Listing Model

`Listing` is the user-facing sale listing object.

Current backing tables:

- `property_offerings`
- `property_offering_sources`
- `property_source_offerings`
- `property_dimension_profiles`
- `property_offering_transactions`

Target tables:

- `listings`
- `source_listings`
- `target_sources`
- `dimension_profiles`
- `price_links`
- `target_observations`

Expected API/read shape:

- `id`: canonical listing UUID.
- `facts`: resolved normalized facts from dimension profiles.
- `sources`: linked source listings with provider, native ID, URL, status, score, and raw payload link.
- `price_matches`: linked prices transactions and legacy price evidence.
- `observations`: target-scoped LLM or rule-generated observations, grouped for listing detail.
- `documents`: attached documents and extraction status.
- `raw`: fetched through a source-specific endpoint, not embedded by default.

Source link behavior:

- `target_sources` is the authoritative target-to-source link table.
- Link statuses remain `confirmed`, `candidate`, and `rejected`.
- Link methods remain explicit, for example `sync_auto`, `source_match_auto`, `manual`, and `backfill_auto`.
- Automatic matching should produce candidate evidence before changing confirmed links.
- Manual links must not be overwritten by automatic matching.
- A partial unique index must ensure one source listing backs at most one non-rejected canonical listing.

Source link recommendation:

- `target_sources` should fully replace `listing_sources`, `housing_company_sources`, and `document_targets`.
- Use domain-specific query views only when they make reads clearer; do not create separate authoritative link tables.
- Enforce target-specific rules with partial indexes and application-level workflow checks.
- Keep source-normalized entities in typed source tables, for example `source_listings` and `source_housing_companies`.

## Canonical Housing Company Model

`HousingCompany` is the user-facing housing-company object.

Backed by:

- `housing_companies`
- `source_housing_companies`
- `target_sources`
- `housing_company_merge_decisions`
- `property_dimension_claims`
- `property_dimension_values`
- `property_dimension_profiles`

Housing-company source types:

- Etuovi housing-company page.
- Oikotie housing-company or building page.
- Listing-derived housing-company facts.
- Manager certificate.
- Manual input.

Matching priority:

1. Business ID.
2. Provider housing-company source page ID.
3. Exact normalized company name plus address and postal code.
4. Address/building match with provider page, listing, document, or geocode support.
5. Similar normalized company name plus nearby/same address evidence.
6. Manual link for ambiguous cases.

Rules:

- Provider housing-company pages should be first-class source evidence.
- Listing-derived housing-company data should be lower-reliability because listing pages are partial and stale more often.
- `source_housing_companies` should identify source provider, source kind, native ID, raw table, raw ID, URL, hash/version fields, and timestamps.
- `target_sources` should identify the housing-company link status, method, score, reasons, and source reference.
- `housing_company_merge_decisions` should preserve merge history and evidence.
- Manual delinking must be represented by rejected/superseded links or merge decisions so later automatic jobs do not relink the same pair without stronger evidence.

## Listing To Housing Company

The intended relationship is:

- `Listing -> Unit` for apartments.
- `Unit -> HousingCompany`.
- `Unit -> PhysicalBuilding -> HousingCompany` when building identity is known separately.
- `HousingCompany -> PhysicalBuilding[]`.
- `Listing -> House` for detached houses.
- `House` has no housing company by default.

Rules:

- Apartment listings should resolve to a unit and housing company whenever evidence supports it.
- If Etuovi or Oikotie has a housing-company page, use it as stronger evidence than listing-derived fields.
- If a provider only has listing data, still create claims for housing-company identity/facts with lower source reliability.
- Housing-company matching and listing matching should be separate but coordinated. A strong housing-company link can support listing-source matching, but should not alone prove two listings are the same unit.

## Price Data And Legacy Building Announcements

Prices data is no longer published, so matching mostly helps older listings and historical provider data.

Target behavior:

- Create a dedicated `price_links` table instead of treating price matches as source listing columns.
- A price link can target a canonical listing when the listing/unit is known.
- A price link can also target a legacy provider building announcement when older data has building-related announcement evidence but no actual ad.
- Source-level price matches remain migration evidence until backfilled.
- A prices transaction should link at most once to the same canonical listing.
- Ambiguous matches should remain reviewable and manually rejectable.

Target link shape:

- `price_link_id`
- `target_type`: `listing`, `source_listing`, `source_building_announcement`, `building`, or `housing_company`
- `target_id`
- `prices_transaction_id`
- `link_status`
- `link_method`
- `link_score`
- `link_reasons`
- `created_at`
- `updated_at`

## Manager Certificate As A Source

`isännöitsijäntodistus` should be treated as a document source, not just an attachment.

Existing flow:

- Original PDF is stored in `property_documents`.
- Extraction JSON is stored in `property_document_extractions`.
- Extraction/projector code creates or attaches targets.
- Projected facts are inserted into `property_dimension_claims` with `source_table = 'property_documents'`.

Desired behavior:

- A document can be attached to a listing, unit, building, housing company, or uploaded detached.
- If uploaded detached, create provisional canonical targets from document identity.
- Later matching can merge provisional targets into existing canonical listings or housing companies.
- Every extracted fact must preserve document ID, extraction model, prompt version, evidence text/page/section when available, and extraction timestamp.
- Document-derived claims should feed listing, unit, building, and housing-company profiles.
- Document-to-target links should use `target_sources` with `source_type = 'document'`, not a separate document-specific link table.

Manager certificate fact ownership:

- Unit facts: apartment number, shares, area, room layout, floor, charges, debt share, shareholder liability.
- Building facts: build year, floor count, apartment count, energy class, heating, material, roof, elevator.
- Housing-company facts: name, business ID, apartment count, plot ownership, property manager, finances, restrictions, risks.
- Renovation facts: canonical renovation events on building or housing company, depending on evidence.
- Listing facts: commercial and valuation-facing facts resolved from the unit/company context.

Document reliability:

- Manager certificate evidence should generally outrank listing-page evidence for housing-company, building, finance, charge, risk, and renovation facts.
- It should not automatically overwrite manually confirmed values.
- Conflicts should remain visible in `property_dimension_values.conflict_status` and supporting/rejected claim IDs.

## Claims, Observations, And Insights

Insights should not be listing-only. A listing page, housing-company page, manager certificate, or price match can create information about several targets:

- Listing: sale timing, pricing narrative, provider inconsistencies, unusually high/low asking price.
- Unit: condition, layout quality, charges, debt share, renovation need, valuation inputs.
- Building: elevator, roof, materials, upcoming renovations, technical risks.
- Housing company: financial risk, maintenance risk, legal restrictions, plot ownership, property manager.
- House: plot, condition, systems, detached-house-specific risks.

Use two related concepts:

- `dimension_claims` for structured facts that should resolve into canonical profiles.
- `target_observations` for narrative/risk/insight output that should be displayed, reviewed, or used by valuation but does not necessarily become a selected scalar fact.

`target_observations` target shape:

- `target_observation_id`
- `target_type`
- `target_id`
- `observation_key`
- `observation_kind`: `risk`, `opportunity`, `inconsistency`, `summary`, `valuation_note`
- `severity`
- `direction`
- `value`
- `text`
- `confidence`
- `evidence`
- `source_type`: `source_listing`, `source_housing_company`, `document`, `price_transaction`, `dimension_claim`, `manual`
- `source_id`
- `created_at`
- `superseded_at`

Listing detail should aggregate observations from the listing, unit/house, building, and housing company so users see the full context without losing provenance.

## Additional Provider Extension Contract

Adding a new listing provider should require these pieces:

1. Raw sync table or tables that retain provider payloads.
2. Source normalizer that writes source listings and source housing-company references.
3. Source hash/version tracking to know when normalization is stale.
4. Source claim projection into `property_dimension_claims`.
5. Source matcher that proposes or confirms `target_sources` links.
6. Housing-company matcher that proposes or confirms `target_sources` links from source housing-company evidence.
7. Raw payload API path through source records.
8. Fixture tests for parsing, matching, and projection.

Provider-specific parsing must stay isolated to the provider sync/normalization layer. Canonical reads should consume source rows, canonical links, and dimension profiles.

## Workflow Model

Raw sync writes raw evidence only. All derived work must be handled by Go Absurd workflows. Database constraints may enforce invariants, but database triggers must not perform source normalization, canonical linking, target rebuilding, dimension projection, dimension resolution, price linking, or observation generation.

Workflows must be idempotent, rerunnable, and versioned. Spawn idempotency keys should cover duplicate raw updates, retries, deploy overlap, and manual rebuilds.

Initial workflow contracts:

- `NormalizeRawProviderSource`: input is provider, raw table, raw ID, payload hash, and normalizer version. Output is a `source_listing` or `source_housing_company`.
- `BuildCanonicalTargets`: input is source entity IDs and builder version. Output is canonical housing companies, buildings, units, houses, and listings.
- `LinkSourcesToTargets`: input is source entity IDs, candidate target IDs, and matcher version. Output is `target_sources` links.
- `ProjectDimensionClaims`: input is source or document IDs and projection version. Output is `dimension_claims`.
- `ResolveDimensionProfiles`: input is dirty target IDs and resolver version. Output is `dimension_values` and `dimension_profiles`.
- `BuildPriceLinks`: input is price transaction IDs, source listing IDs, or rebuild scope plus matcher version. Output is `price_links`.
- `GenerateTargetObservations`: input is target IDs and observation version. Output is `target_observations`.
- `RebuildCanonicalPropertyModel`: parent workflow for full or scoped rebuilds.

Dirty scopes are the units of derived data that need recomputation after raw evidence changes. For example, if one `shortcut_ads` row changes, the dirty scope may be that raw row, the resulting source listing, the canonical listing, its unit, and related profiles. Dirty scopes should be explicit workflow inputs, not inferred by triggers.

Absurd task payloads are enough for ordinary incremental updates. Explicit dirty tables are only needed if operators need to inspect, batch, pause, or replay pending recomputation outside Absurd's normal task inspection tools.

Candidate dirty tables, if operator visibility becomes necessary:

- `dirty_sources`: source type, source ID, reason, version, queued at.
- `dirty_targets`: target type, target ID, reason, version, queued at.
- `dirty_price_transactions`: prices transaction ID, reason, version, queued at.

Open workflow-shape questions:

- Should source normalization enqueue canonical build/link workflows immediately per raw row, or should a batch parent workflow coalesce dirty source IDs first?
- Do Absurd task inspection tools provide enough operator visibility for dirty scopes, or should explicit dirty tables be added later?

## Implementation Phases

### Phase 1: Clean Schema

- Create final-name source, canonical, source-link, dimension, observation, and price-link tables.
- Preserve raw provider tables, prices data, PDFs, and extraction outputs.
- Do not preserve old computed table names unless external API compatibility requires temporary aliases.
- Add partial unique indexes for active source links and confirmed price links.

### Phase 2: Source Normalization Workflows

- Implement Go Absurd workflows that normalize raw provider rows into `source_listings` and `source_housing_companies`.
- Track payload hash, normalizer version, and normalized timestamp.
- Stop introducing new `sale_listing` terminology in Go/API code.

### Phase 3: Canonical Target Rebuild

- Build `housing_companies`, `physical_buildings`, `units`, `houses`, and `listings` from source entities and document evidence.
- Link sources, documents, and manual inputs to canonical targets through `target_sources`.
- Keep all rebuild steps idempotent and versioned.

### Phase 4: Claims, Profiles, Prices, And Observations

- Project source and document claims into `dimension_claims`.
- Resolve `dimension_values` and `dimension_profiles`.
- Build `price_links` from price transactions, source listings, and legacy building-announcement evidence.
- Replace listing-only insight thinking with target-scoped `target_observations`.
- Aggregate listing, unit/house, building, and housing-company observations for listing detail.

### Phase 5: Read Path Switch

- Make listing search/detail start from `listings`.
- Join source evidence through `target_sources`.
- Use dimension profiles as primary facts.
- Use source rows only as fallback when resolved dimensions are missing.
- Expose raw payloads through explicit source raw endpoints.

### Phase 6: Provider Contract Hardening

- Create fixture tests for Etuovi and Oikotie parsing and matching.
- Add a fake provider fixture to prove the extension contract does not require canonical read-path changes.
- Document required source fields and matching evidence for new providers.

### Phase 7: Drop Disposable Computed State

- Drop old computed tables, functions, and triggers after rebuild parity checks.
- Keep raw evidence tables, price transactions, documents, and extraction outputs.
- Do not keep legacy ID redirect tables.

## Disposable Computed Tables

The following tables and related functions/triggers are not authoritative migration inputs. They can be used for validation and parity checks, but the new model should be rebuilt from raw provider evidence, prices data, PDFs, and extraction outputs.

- `property_source_offerings`
- `property_offerings`
- `property_offering_sources`
- `property_offering_transactions`
- `housing_company_sources`
- `housing_company_facts`
- `property_units`
- `physical_buildings`
- `property_houses`
- `property_target_sources`
- `property_dimension_claims`
- `property_dimension_values`
- `property_dimension_profiles`
- `property_dimension_projection_runs`
- `property_dimension_dirty_targets`
- `property_source_offering_insights`
- `apartment_profiles`
- `building_profiles`
- `property_system_profiles`
- canonical sync, source matching, dimension projection, and geometry refresh triggers/functions

## Target Database Shape

The exact migration can happen incrementally, but the target schema should expose these domain names.

### Listings And Sources

```sql
CREATE TABLE listings (
    listing_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_type text NOT NULL,
    listing_status text,
    primary_source_listing_id uuid,
    unit_id uuid,
    house_id uuid,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (listing_type = ANY (ARRAY['sale'])),
    CHECK (((unit_id IS NOT NULL)::int + (house_id IS NOT NULL)::int) = 1)
);
```

```sql
CREATE TABLE source_listings (
    source_listing_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text NOT NULL,
    canonical_source_id text NOT NULL,
    raw_table text NOT NULL,
    raw_id text NOT NULL,
    url text,
    payload_hash text,
    normalized_version integer NOT NULL DEFAULT 0,
    normalized_at timestamptz,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, source_kind, native_id)
);
```

```sql
CREATE TABLE target_sources (
    target_source_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer NOT NULL,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['listing','unit','building','housing_company','house'])),
    CHECK (source_type = ANY (ARRAY['source_listing','source_housing_company','document','price_transaction','manual'])),
    CHECK (link_status = ANY (ARRAY['confirmed','candidate','rejected','superseded'])),
    CHECK (link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']))
);

CREATE UNIQUE INDEX target_sources_active_source_listing
ON target_sources (source_id)
WHERE target_type = 'listing'
  AND source_type = 'source_listing'
  AND link_status <> 'rejected';

CREATE INDEX target_sources_target
ON target_sources (target_type, target_id, link_status);

CREATE INDEX target_sources_source
ON target_sources (source_type, source_id, link_status);
```

### Canonical Property Targets

```sql
CREATE TABLE housing_companies (
    housing_company_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_key text NOT NULL UNIQUE,
    business_id text,
    name text,
    address_norm text,
    postal_norm text,
    city_norm text,
    geom postgis.geometry(Point, 4326),
    match_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

```sql
CREATE TABLE source_housing_companies (
    source_housing_company_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text,
    raw_table text,
    raw_id text,
    url text,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

```sql
CREATE TABLE physical_buildings (
    physical_building_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    housing_company_id uuid REFERENCES housing_companies(housing_company_id) ON DELETE SET NULL,
    identity_key text NOT NULL UNIQUE,
    address_norm text,
    postal_norm text,
    city_norm text,
    latitude double precision,
    longitude double precision,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

```sql
CREATE TABLE units (
    unit_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    housing_company_id uuid REFERENCES housing_companies(housing_company_id) ON DELETE SET NULL,
    physical_building_id uuid REFERENCES physical_buildings(physical_building_id) ON DELETE SET NULL,
    identity_key text NOT NULL UNIQUE,
    address_norm text,
    apartment text,
    floor_level integer,
    area_m2 double precision,
    room_layout text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

```sql
CREATE TABLE houses (
    house_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_key text NOT NULL UNIQUE,
    address_norm text,
    postal_norm text,
    city_norm text,
    latitude double precision,
    longitude double precision,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

### Documents, Claims, Observations, And Prices

```sql
CREATE TABLE target_observations (
    target_observation_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    observation_key text NOT NULL,
    observation_kind text NOT NULL,
    severity text NOT NULL,
    direction text NOT NULL,
    value jsonb,
    text text,
    confidence double precision NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    superseded_at timestamptz
);
```

```sql
CREATE TABLE price_links (
    price_link_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    prices_transaction_id uuid NOT NULL REFERENCES prices_transactions(prices_transaction_id) ON DELETE CASCADE,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer NOT NULL,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (target_type, target_id, prices_transaction_id)
);

CREATE UNIQUE INDEX price_links_one_confirmed_listing_per_transaction
ON price_links (prices_transaction_id)
WHERE target_type = 'listing'
  AND link_status = 'confirmed';
```

## Test Scenarios

- A listing with both Etuovi and Oikotie sources resolves to one canonical listing when unit evidence is strong.
- Two listings in the same housing company do not merge unless unit evidence matches.
- A provider housing-company page links to one canonical housing company and enriches all linked listings.
- A manager certificate uploaded to a listing enriches listing, unit, building, and housing-company profiles.
- A detached manager certificate creates provisional targets and later merges cleanly into an existing listing/company.
- A prices transaction links once to the canonical listing even when multiple source listings exist.
- A prices transaction can link to a legacy source building announcement when no canonical listing exists.
- A target observation created from a source listing can target the unit, building, housing company, or listing and still appear in listing detail.
- Re-syncing a provider payload updates source facts and claims without changing canonical listing UUIDs.
- Manual source links, manual delinks, merge rejections, and manual dimension overrides are not overwritten by automatic jobs.

## Open Questions

- Should canonical listing API fields be renamed from `offering_id` to `listing_id` in a breaking API version, or should both be returned during a transition?
- Should source-level insights be migrated into target observations immediately, or first bridged through a compatibility projection?
- What is the minimum evidence threshold for auto-linking housing companies without business ID?
- How should we handle one housing company with multiple physical buildings at the same address or adjacent addresses?
- Should detached manager certificate uploads be allowed to create canonical listings, or only unit/building/housing-company targets until a listing is explicitly linked?
- Which document facts are allowed to update valuation inputs automatically, and which require review?
- Do Absurd task inspection tools provide enough operator visibility for dirty scopes, or should explicit dirty tables be added later?

## Default Decisions For First Implementation Goal

- Preserve only raw provider evidence, prices data, PDFs, and extraction outputs as authoritative inputs.
- Treat existing computed tables as disposable validation data.
- Truncate computed tables for full rebuilds instead of writing shadow tables.
- Use fresh UUIDs for rebuilt canonical entities and do not maintain legacy redirects.
- Use `listing` terminology throughout the new target model.
- Use `target_sources` instead of separate `listing_sources`, `housing_company_sources`, and `document_targets`.
- Treat manager certificates as document sources with high-reliability claims.
- Make dimension profiles the preferred read source for normalized facts.
- Model LLM output as target-scoped observations plus structured dimension claims, not listing-only insights.
- Move price matching into dedicated `price_links` with canonical listing and legacy source-announcement targets.
- Preserve raw provider payloads and PDFs as evidence.
- Implement all derived-data updates as Go Absurd workflows. Do not use database triggers for computed model updates.
