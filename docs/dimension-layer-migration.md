# Dimension Layer Migration

This migration replaces the current derived claim/profile layer with a rebuildable dimension layer. Source data stays; derived data can be dropped and rebuilt.

The core rule is:

```text
source rows -> source-scoped claims/events -> entity links -> resolved values -> profiles
```

Claims are evidence. Links are identity decisions. Values are resolved facts.

## Keep And Drop Boundary

Keep only source data:

- `property_source_offerings`
- raw provider tables for Shortcut, Frontdoor, announcements, and price transactions
- `property_documents`
- `property_document_extraction_runs`

Drop or rebuild everything else:

- `property_offerings`
- `property_offering_sources`
- `property_units`
- `physical_buildings`
- `housing_companies`
- source matching/linking tables that decide offering, unit, building, company, or transaction identity
- `property_claims`
- `apartment_profiles`
- `building_profiles`
- `housing_company_profiles`
- `housing_company_systems`
- `housing_company_renovations`
- `property_quality_scores`
- valuation-specific projection tables based on old claim names

The migration does not need compatibility with old derived table shapes or old canonical/linking rows. Canonical identity tables may keep their current names if useful, but their rows should be treated as rebuildable outputs from source data.

## Architecture

### 1. Source Evidence Projection

Each projector reads one stable source row or extraction run and writes source-scoped claims/events. This step must not require canonical matching to be solved.

Examples:

- Shortcut listing -> `target_type = 'listing'`
- Frontdoor listing -> `target_type = 'listing'`
- listing text extraction -> `target_type = 'listing'`
- manager certificate PDF -> `target_type = 'document'` plus known `housing_company`, `building`, or `unit` targets when already linked
- price transaction -> `target_type = 'transaction'`
- manual override -> direct canonical target

Projectors are deterministic. Rebuilding a source and projection version deletes that source/version's claims/events and inserts a fresh result.

### 2. Entity Resolution And Linking

Matching processes link source targets to canonical identity rows.

This layer answers:

- which source listings belong to the same `property_offering`;
- which offering belongs to which `property_unit`;
- which unit belongs to which `physical_building`;
- which building belongs to which `housing_company`;
- which price transaction matches which offering, unit, building, or listing;
- which PDF document belongs to which company, building, or unit.

This layer writes canonical identity and relationship/link rows. It does not rewrite extracted evidence.

### 3. Link-Aware Value Resolution

After links exist, the resolver reads source-scoped claims through those links.

Example:

```text
listing A: unit.area_m2 = 52.0
listing B: unit.area_m2 = 52.5
listing A and B link to the same property_offering and property_unit
resolver sees both listing claims as evidence for the linked property_unit
resolver selects unit.area_m2 with provenance to the source claims
```

Resolution groups linked source claims by canonical `target_type`, `target_id`, and `dimension_key`, then applies a dimension-specific policy.

Manual overrides win. Otherwise strategies decide by source priority, source reliability, confidence, freshness, or numeric consensus.

### 4. Profile Projection

Profiles are JSONB read models for valuation, API responses, and UI. They are fully rebuildable from resolved dimension values and typed event projections.

### 5. Rebuild Scheduling

Semantic work runs in canonical jobs, not triggers.

Source projection and linking code may mark affected listing or canonical targets dirty. The canonical queue then fans dirty rows into small idempotent jobs:

- `canonical_rebuild_dimension_layer_listing` projects one listing's source claims, resolves linked targets, and projects profiles.
- `canonical_resolve_dimension_target` resolves one already-linked target and projects its profile.
- `canonical_rebuild_dimension_layer_backfill` only fans out listing jobs.
- `canonical_resolve_dirty_dimension_targets` only fans out dirty target jobs.

Database functions may enforce local invariants and record dirty rows, but they must not perform matching, semantic resolution, profile projection, or cross-source linking implicitly as trigger side effects.

## Canonical Names

Use one canonical key per real-world meaning.

Top-level namespaces:

- `unit.*`
- `layout.*`
- `condition.*`
- `features.*`
- `charges.*`
- `building.*`
- `site.*`
- `housing_company.*`
- `risk.*`
- `document.*`
- `score.*`
- `adjustment.*`

Rules:

- Use one owner namespace for each meaning: `unit.floor_level`, not `floor.floor_level`.
- Use `_m2` for areas.
- Use `_eur` for money.
- Use `_monthly_eur` for monthly charges.
- Prefer direct boolean names where the namespace is clear: `features.balcony`, not `balcony.has_balcony`.
- Store source/raw metadata under `document.*` or extraction-run JSON.
- Store derived outputs under `score.*` and `adjustment.*`.
- Do not keep old aliases in the new persisted layer; translate old extraction keys at projector boundaries.

Initial canonical replacements:

| Old key | New key |
| --- | --- |
| `unit.balcony` | `features.balcony` |
| `balcony.has_balcony` | `features.balcony` |
| `balcony.glazing` | `features.balcony_glazing` |
| `unit.sauna` | `features.sauna` |
| `sauna.has_sauna` | `features.sauna` |
| `sauna.private_sauna` | `features.private_sauna` |
| `parking.parking_text` | `features.parking_type` |
| `storage.storage_quality` | `features.storage_quality` |
| `views.view_quality` | `features.view_quality` |
| `views.noise_risk` | `features.noise_risk` |
| `condition.condition` | `condition.unit_condition` |
| `layout.layout_quality` | `layout.quality` |
| `layout.awkward_layout` | `layout.awkward` |
| `heating.heating_method` | `building.heating_method` |
| `charges.maintenance_charge_monthly` | `charges.maintenance_monthly_eur` |
| `charges.capital_charge_monthly` | `charges.capital_monthly_eur` |
| `charges.total_charge_monthly` | `charges.total_monthly_eur` |

## Target Types

Allowed target types:

- `listing`
- `document`
- `transaction`
- `offering`
- `unit`
- `building`
- `housing_company`

Source projectors write `listing`, `document`, or `transaction` claims first. If a source is already explicitly attached to a canonical target, it may write a source-scoped claim directly on `offering`, `unit`, `building`, or `housing_company`.

## Core Schema

### Projection Runs

Projection runs make source projection, resolution, and profile projection disposable and auditable.

```sql
CREATE TABLE public.property_dimension_projection_runs (
    property_dimension_projection_run_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    projection_type text NOT NULL,
    projection_version text NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    status text NOT NULL,
    result jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_text text,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CHECK (projection_type = ANY (ARRAY['source_claims','renovation_events','resolved_values','profiles','system_profiles']::text[])),
    CHECK (status = ANY (ARRAY['running','succeeded','failed']::text[]))
);
CREATE INDEX idx_property_dimension_projection_runs_source ON public.property_dimension_projection_runs (projection_type, source_table, source_id, projection_version, started_at DESC);
```

### Dimension Claims

Claims are normalized factual evidence rows. They are the source of truth for facts; links are the source of truth for identity and attachment decisions.

```sql
CREATE TABLE public.property_dimension_claims (
    property_dimension_claim_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_dimension_projection_run_id uuid NOT NULL REFERENCES public.property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
    projection_version text NOT NULL,
    claim_scope text NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    source_field text,
    source_claim_id uuid REFERENCES public.property_dimension_claims(property_dimension_claim_id),
    source_observed_at timestamptz,
    valid_from date,
    valid_until date,
    confidence double precision NOT NULL DEFAULT 0.5,
    source_reliability double precision NOT NULL DEFAULT 0.5,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    extraction_model text,
    extraction_prompt_version text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (claim_scope = ANY (ARRAY['source','manual']::text[])),
    CHECK (target_type = ANY (ARRAY['listing','document','transaction','offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (source_reliability >= 0 AND source_reliability <= 1)
);
CREATE UNIQUE INDEX idx_property_dimension_claims_unique_source ON public.property_dimension_claims (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''), projection_version);
CREATE INDEX idx_property_dimension_claims_target ON public.property_dimension_claims (claim_scope, target_type, target_id, dimension_key);
CREATE INDEX idx_property_dimension_claims_source ON public.property_dimension_claims (source_table, source_id, projection_version);
CREATE INDEX idx_property_dimension_claims_source_claim ON public.property_dimension_claims (source_claim_id);
CREATE INDEX idx_property_dimension_claims_dimension ON public.property_dimension_claims (dimension_key);
CREATE INDEX idx_property_dimension_claims_value_gin ON public.property_dimension_claims USING gin (value jsonb_path_ops);
```

### Manual Overrides

Manual overrides are explicit source rows that produce `claim_scope = 'manual'` claims.

```sql
CREATE TABLE public.property_dimension_manual_overrides (
    property_dimension_manual_override_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    reason text NOT NULL,
    created_by text NOT NULL,
    valid_from date,
    valid_until date,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[]))
);
CREATE UNIQUE INDEX idx_property_dimension_manual_overrides_active ON public.property_dimension_manual_overrides (target_type, target_id, dimension_key) WHERE revoked_at IS NULL;
```

### Dimension Values

Values are selected canonical facts. They are written only for canonical targets.

```sql
CREATE TABLE public.property_dimension_values (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    confidence double precision NOT NULL,
    selected_claim_id uuid REFERENCES public.property_dimension_claims(property_dimension_claim_id),
    selected_reason text NOT NULL,
    conflict_status text NOT NULL DEFAULT 'none',
    supporting_claim_ids uuid[] NOT NULL DEFAULT '{}',
    rejected_claim_ids uuid[] NOT NULL DEFAULT '{}',
    resolved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id, dimension_key),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (conflict_status = ANY (ARRAY['none','compatible','conflicting','manual_override']::text[]))
);
CREATE INDEX idx_property_dimension_values_dimension ON public.property_dimension_values (dimension_key);
CREATE INDEX idx_property_dimension_values_selected_claim ON public.property_dimension_values (selected_claim_id);
```

### Dimension Profiles

Profiles are read models for valuation and UI.

```sql
CREATE TABLE public.property_dimension_profiles (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimensions jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    conflicts jsonb NOT NULL DEFAULT '{}'::jsonb,
    resolved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[]))
);
CREATE INDEX idx_property_dimension_profiles_dimensions_gin ON public.property_dimension_profiles USING gin (dimensions jsonb_path_ops);
CREATE INDEX idx_unit_dimension_profiles_area ON public.property_dimension_profiles (((dimensions #>> '{unit,area_m2}')::double precision)) WHERE target_type = 'unit';
CREATE INDEX idx_unit_dimension_profiles_total_charge ON public.property_dimension_profiles (((dimensions #>> '{charges,total_monthly_eur}')::double precision)) WHERE target_type = 'unit';
CREATE INDEX idx_building_dimension_profiles_build_year ON public.property_dimension_profiles (((dimensions #>> '{building,build_year}')::integer)) WHERE target_type = 'building';
```

### Catalog And Resolution Policies

The catalog is the executable contract for projector output, profile paths, and valuation input mapping.

```sql
CREATE TABLE public.property_dimension_catalog (
    dimension_key text PRIMARY KEY,
    target_type text NOT NULL,
    value_kind text NOT NULL,
    unit text,
    profile_section text NOT NULL,
    profile_key text NOT NULL,
    promoted_to_valuation boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[]))
);
CREATE TABLE public.property_dimension_resolution_policies (
    dimension_key text PRIMARY KEY REFERENCES public.property_dimension_catalog(dimension_key),
    strategy text NOT NULL,
    freshness_half_life_days integer,
    conflict_tolerance jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (strategy = ANY (ARRAY['manual_override','latest_reliable','highest_reliability','document_preferred','stable_identity','numeric_consensus']::text[]))
);
CREATE TABLE public.property_dimension_source_priorities (
    dimension_key text NOT NULL REFERENCES public.property_dimension_catalog(dimension_key),
    source_table text NOT NULL,
    source_field text,
    priority integer NOT NULL,
    default_reliability double precision NOT NULL,
    CHECK (default_reliability >= 0 AND default_reliability <= 1)
);
CREATE UNIQUE INDEX idx_property_dimension_source_priorities_unique ON public.property_dimension_source_priorities (dimension_key, source_table, COALESCE(source_field, ''));
```

## Typed Renovations And Systems

Renovations are event-like and remain source/manual scoped. Linking decides which target graph consumes them; it does not rewrite them as canonical events.

```sql
CREATE TABLE public.property_renovation_events (
    property_renovation_event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_dimension_projection_run_id uuid NOT NULL REFERENCES public.property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
    projection_version text NOT NULL,
    event_scope text NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    source_field text,
    source_event_id uuid REFERENCES public.property_renovation_events(property_renovation_event_id),
    category text NOT NULL,
    component text,
    status text NOT NULL,
    stage text,
    scope text,
    responsibility text,
    year integer,
    start_year integer,
    end_year integer,
    cost_estimate_eur bigint,
    summary text,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    confidence double precision NOT NULL DEFAULT 0.5,
    source_reliability double precision NOT NULL DEFAULT 0.5,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (event_scope = ANY (ARRAY['source','manual']::text[])),
    CHECK (target_type = ANY (ARRAY['listing','document','offering','unit','building','housing_company']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (source_reliability >= 0 AND source_reliability <= 1)
);
CREATE UNIQUE INDEX idx_property_renovation_events_unique_source ON public.property_renovation_events (event_scope, target_type, target_id, source_table, source_id, COALESCE(source_field, ''), category, status, COALESCE(stage, ''), COALESCE(scope, ''), COALESCE(year, -1), COALESCE(start_year, -1), COALESCE(end_year, -1), md5(COALESCE(summary, '')), projection_version);
CREATE INDEX idx_property_renovation_events_target ON public.property_renovation_events (event_scope, target_type, target_id, category, status);
CREATE INDEX idx_property_renovation_events_source ON public.property_renovation_events (source_table, source_id, projection_version);
CREATE INDEX idx_property_renovation_events_source_event ON public.property_renovation_events (source_event_id);
```

System profiles are projections from source/manual renovation events and resolved dimension profiles.

```sql
CREATE TABLE public.property_system_profiles (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    system_type text NOT NULL,
    status text NOT NULL,
    last_renovated_year integer,
    next_expected_start_year integer,
    next_expected_end_year integer,
    stage text,
    scope text,
    responsibility text,
    cost_estimate_eur bigint,
    confidence double precision NOT NULL DEFAULT 0.5,
    selected_renovation_event_ids uuid[] NOT NULL DEFAULT '{}',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id, system_type),
    CHECK (target_type = ANY (ARRAY['unit','building','housing_company']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1)
);
CREATE INDEX idx_property_system_profiles_target ON public.property_system_profiles (target_type, target_id);
```

## Initial Catalog

Minimum dimensions to seed before projectors run:

| Target | Dimension | Kind | Unit | Strategy |
| --- | --- | --- | --- | --- |
| `unit` | `unit.area_m2` | `number` | `m2` | `numeric_consensus` |
| `unit` | `unit.living_area_m2` | `number` | `m2` | `numeric_consensus` |
| `unit` | `unit.total_area_m2` | `number` | `m2` | `numeric_consensus` |
| `unit` | `unit.other_area_m2` | `number` | `m2` | `numeric_consensus` |
| `unit` | `unit.floor_level` | `number` | null | `numeric_consensus` |
| `unit` | `unit.total_floors` | `number` | null | `numeric_consensus` |
| `unit` | `unit.apartment_number` | `string` | null | `stable_identity` |
| `unit` | `unit.shares` | `string` | null | `stable_identity` |
| `unit` | `layout.room_layout` | `string` | null | `latest_reliable` |
| `unit` | `layout.room_count` | `number` | null | `numeric_consensus` |
| `unit` | `layout.bedroom_count` | `number` | null | `numeric_consensus` |
| `unit` | `layout.kitchen_type` | `string` | null | `latest_reliable` |
| `unit` | `layout.separate_wc_count` | `number` | null | `numeric_consensus` |
| `unit` | `layout.quality` | `string` | null | `latest_reliable` |
| `unit` | `layout.awkward` | `boolean` | null | `latest_reliable` |
| `unit` | `condition.unit_condition` | `string` | null | `latest_reliable` |
| `unit` | `condition.kitchen_condition` | `string` | null | `latest_reliable` |
| `unit` | `condition.bathroom_condition` | `string` | null | `latest_reliable` |
| `unit` | `condition.surface_renovation_need` | `boolean` | null | `latest_reliable` |
| `unit` | `condition.modernization_need` | `boolean` | null | `latest_reliable` |
| `unit` | `features.sauna` | `boolean` | null | `latest_reliable` |
| `unit` | `features.private_sauna` | `boolean` | null | `latest_reliable` |
| `unit` | `features.balcony` | `boolean` | null | `latest_reliable` |
| `unit` | `features.balcony_glazing` | `boolean` | null | `latest_reliable` |
| `unit` | `features.parking_type` | `string` | null | `latest_reliable` |
| `unit` | `features.storage_quality` | `string` | null | `latest_reliable` |
| `unit` | `features.view_quality` | `string` | null | `latest_reliable` |
| `unit` | `features.noise_risk` | `boolean` | null | `latest_reliable` |
| `unit` | `features.accessibility` | `string` | null | `latest_reliable` |
| `unit` | `charges.maintenance_monthly_eur` | `number` | `eur/month` | `latest_reliable` |
| `unit` | `charges.capital_monthly_eur` | `number` | `eur/month` | `latest_reliable` |
| `unit` | `charges.total_monthly_eur` | `number` | `eur/month` | `latest_reliable` |
| `unit` | `charges.water_monthly_eur` | `number` | `eur/month` | `latest_reliable` |
| `unit` | `charges.parking_monthly_eur` | `number` | `eur/month` | `latest_reliable` |
| `unit` | `charges.debt_share_eur` | `number` | `eur` | `latest_reliable` |
| `unit` | `charges.charge_risk` | `string` | null | `latest_reliable` |
| `unit` | `risk.shareholder_liability` | `string` | null | `document_preferred` |
| `building` | `building.build_year` | `number` | null | `stable_identity` |
| `building` | `building.floor_count` | `number` | null | `numeric_consensus` |
| `building` | `building.apartment_count` | `number` | null | `numeric_consensus` |
| `building` | `building.elevator` | `boolean` | null | `highest_reliability` |
| `building` | `building.energy_class` | `string` | null | `document_preferred` |
| `building` | `building.heating_method` | `string` | null | `document_preferred` |
| `building` | `building.material` | `string` | null | `document_preferred` |
| `building` | `building.roof_type` | `string` | null | `document_preferred` |
| `building` | `building.roof_material` | `string` | null | `document_preferred` |
| `building` | `building.common_area_quality` | `string` | null | `latest_reliable` |
| `building` | `building.accessibility` | `string` | null | `latest_reliable` |
| `housing_company` | `housing_company.name` | `string` | null | `stable_identity` |
| `housing_company` | `housing_company.business_id` | `string` | null | `stable_identity` |
| `housing_company` | `housing_company.apartment_count` | `number` | null | `numeric_consensus` |
| `housing_company` | `housing_company.building_count` | `number` | null | `numeric_consensus` |
| `housing_company` | `site.plot_ownership_type` | `string` | null | `document_preferred` |
| `housing_company` | `site.plot_lease_end_year` | `number` | null | `document_preferred` |
| `housing_company` | `site.plot_redemption_possible` | `boolean` | null | `document_preferred` |
| `housing_company` | `risk.financial_risk` | `string` | null | `document_preferred` |
| `housing_company` | `risk.maintenance_risk` | `string` | null | `document_preferred` |
| `housing_company` | `risk.repair_backlog_risk` | `string` | null | `document_preferred` |
| `housing_company` | `risk.administrative_legal_risk` | `string` | null | `document_preferred` |
| `housing_company` | `risk.restrictions` | `array` | null | `document_preferred` |

## Source Mapping

Provider fields project to `listing` claims first. Resolution reads those source claims through the link graph when canonical targets exist.

| Source field | Source target | Canonical target | Dimension |
| --- | --- | --- | --- |
| `sale_listing_area_value` | `listing` | `unit` | `unit.area_m2` |
| `sale_listing_living_area_value` | `listing` | `unit` | `unit.living_area_m2` |
| `sale_listing_total_area_value` | `listing` | `unit` | `unit.total_area_m2` |
| `sale_listing_other_area_value` | `listing` | `unit` | `unit.other_area_m2` |
| `sale_listing_room_layout` | `listing` | `unit` | `layout.room_layout` |
| `sale_listing_rooms_count` | `listing` | `unit` | `layout.room_count` |
| `sale_listing_bedrooms_count` | `listing` | `unit` | `layout.bedroom_count` |
| `sale_listing_floor_level` | `listing` | `unit` | `unit.floor_level` |
| `sale_listing_total_floors` | `listing` | `building` | `building.floor_count` |
| `sale_listing_condition` | `listing` | `unit` | `condition.unit_condition` |
| `sale_listing_sauna` | `listing` | `unit` | `features.sauna` |
| `sale_listing_balcony` | `listing` | `unit` | `features.balcony` |
| `sale_listing_parking_text` | `listing` | `unit` | `features.parking_type` |
| `sale_listing_maintenance_charge_monthly` | `listing` | `unit` | `charges.maintenance_monthly_eur` |
| `sale_listing_total_charge_monthly` | `listing` | `unit` | `charges.total_monthly_eur` |
| `sale_listing_water_charge` | `listing` | `unit` | `charges.water_monthly_eur` |
| `sale_listing_debt_share_amount` | `listing` | `unit` | `charges.debt_share_eur` |
| `sale_listing_build_year` | `listing` | `building` | `building.build_year` |
| `sale_listing_elevator` | `listing` | `building` | `building.elevator` |
| `sale_listing_heating_system` | `listing` | `building` | `building.heating_method` |
| `sale_listing_energy_efficiency_label` | `listing` | `building` | `building.energy_class` |
| `sale_listing_building_material` | `listing` | `building` | `building.material` |
| `sale_listing_roof_type` | `listing` | `building` | `building.roof_type` |
| `sale_listing_roof_material` | `listing` | `building` | `building.roof_material` |
| `sale_listing_apartment_count` | `listing` | `housing_company` | `housing_company.apartment_count` |
| `sale_listing_housing_company_name` | `listing` | `housing_company` | `housing_company.name` |
| `sale_listing_housing_company_business_id` | `listing` | `housing_company` | `housing_company.business_id` |
| `sale_listing_plot_type_code` | `listing` | `housing_company` | `site.plot_ownership_type` |

Manager certificate extraction writes `document` claims first. If a document is already linked, it may write source-scoped claims directly to the linked target, but changing the document link must not rewrite extracted evidence.

Transaction projection writes `transaction` claims first. Matching later lets the resolver treat price evidence as evidence for the linked `offering` or `unit`.

## Resolution Semantics

Each candidate claim receives:

```text
score = source_priority * source_reliability * confidence * freshness_factor
```

Freshness is applied only when the policy has `freshness_half_life_days`.

Policy defaults:

- `manual_override`: select active manual claim.
- `stable_identity`: prefer manual, registry/PDF, then consensus; flag conflicts.
- `document_preferred`: prefer manager certificate/PDF over listings.
- `latest_reliable`: score with freshness decay.
- `highest_reliability`: score without freshness unless configured.
- `numeric_consensus`: choose a compatible cluster; reject outliers.

Conflict rules:

- `none`: one selected claim or all claims equivalent.
- `compatible`: multiple claims differ inside tolerance.
- `conflicting`: material disagreement outside tolerance.
- `manual_override`: active manual override selected.

## Profile JSON Shape

Profile JSON keys are derived from `property_dimension_catalog.profile_section` and `profile_key`.

Unit profile:

```json
{
  "unit": {
    "area_m2": 52.5,
    "living_area_m2": 52.5,
    "other_area_m2": 0,
    "floor_level": 4,
    "total_floors": 6,
    "apartment_number": "A 12",
    "shares": "1234-1285"
  },
  "layout": {
    "room_layout": "2h+k",
    "room_count": 2,
    "bedroom_count": 1,
    "kitchen_type": "separate",
    "separate_wc_count": 0,
    "quality": "good",
    "awkward": false
  },
  "condition": {
    "unit_condition": "good",
    "kitchen_condition": "good",
    "bathroom_condition": "fair",
    "surface_renovation_need": false,
    "modernization_need": false
  },
  "features": {
    "sauna": false,
    "private_sauna": false,
    "balcony": true,
    "balcony_glazing": true,
    "parking_type": "yard",
    "storage_quality": "normal",
    "view_quality": "good",
    "noise_risk": false,
    "accessibility": "average"
  },
  "charges": {
    "maintenance_monthly_eur": 312.5,
    "capital_monthly_eur": 0,
    "total_monthly_eur": 312.5,
    "water_monthly_eur": 25,
    "parking_monthly_eur": 20,
    "debt_share_eur": 0,
    "charge_risk": "low"
  }
}
```

Building profile:

```json
{
  "building": {
    "build_year": 1968,
    "floor_count": 6,
    "apartment_count": 42,
    "elevator": true,
    "energy_class": "D2018",
    "heating_method": "district_heating",
    "material": "concrete",
    "roof_type": "flat",
    "roof_material": "bitumen",
    "common_area_quality": "normal",
    "accessibility": "average"
  }
}
```

Housing company profile:

```json
{
  "housing_company": {
    "name": "As Oy Example",
    "business_id": "1234567-8",
    "apartment_count": 42,
    "building_count": 1
  },
  "site": {
    "plot_ownership_type": "owned",
    "plot_lease_end_year": null,
    "plot_redemption_possible": null
  },
  "risk": {
    "financial_risk": "low",
    "maintenance_risk": "medium",
    "repair_backlog_risk": "medium",
    "administrative_legal_risk": "low",
    "restrictions": []
  },
  "systems": {
    "pipes": {
      "status": "renewed",
      "last_renovated_year": 2015,
      "confidence": 0.9
    }
  }
}
```

## Executable Migration Plan

### Phase 1: Drop Old Derived Layer And Add New Tables

1. Create a migration that drops old derived tables.
2. Create `property_dimension_projection_runs`.
3. Create `property_dimension_catalog`.
4. Create `property_dimension_resolution_policies`.
5. Create `property_dimension_source_priorities`.
6. Create `property_dimension_claims`.
7. Create `property_dimension_manual_overrides`.
8. Create `property_dimension_values`.
9. Create `property_dimension_profiles`.
10. Create `property_renovation_events`.
11. Create `property_system_profiles`.
12. Run `mise run backend:db:generate`.

Acceptance checks:

- source tables still exist;
- old derived tables are gone;
- new tables are generated into `internal/db`;
- migration applies on an empty local database and on a restored local dump.

### Phase 2: Seed Catalog And Policies

1. Add seed SQL for the initial catalog.
2. Add seed SQL for resolution policies.
3. Add source priorities for provider fields, listing extraction, manager certificate extraction, transactions, and manual overrides.
4. Add Go constants/helpers generated from or validated against the catalog.

Acceptance checks:

- every source mapping dimension exists in `property_dimension_catalog`;
- every catalog row has a policy;
- valuation-promoted dimensions have profile section/key mappings.

### Phase 3: Source Projectors

1. Implement provider-field projector from `property_source_offerings` to `listing` claims.
2. Implement listing text extraction projector to `listing` claims.
3. Implement listing renovation projector to source `listing` renovation events.
4. Implement manager certificate projector to `document` claims and source `document` renovation events.
5. Implement transaction projector to `transaction` claims.
6. Implement manual override projector to manual target claims.

Acceptance checks:

- each projector can rebuild one source row idempotently;
- each projector records a projection run;
- rerunning a projector replaces claims/events for the same source/version;
- unknown old extraction keys are either translated or rejected with a recorded warning.

### Phase 4: Entity Linking

1. Rebuild source listing to `property_offering` matching.
2. Rebuild offering to `property_unit` matching.
3. Rebuild unit to `physical_building` and `housing_company` links.
4. Rebuild document to company/building/unit links.
5. Rebuild transaction to offering/unit/building links.

Acceptance checks:

- multiple Shortcut/Frontdoor listings can link to one `property_offering`;
- multiple offerings can link to one `property_unit` when they are the same apartment over time;
- listings in the same building can share one `physical_building` and `housing_company`;
- transactions can link with confidence without mutating source evidence.

### Phase 5: Link-Aware Resolution

1. Resolve linked listing claims as evidence for `offering`, `unit`, `building`, and `housing_company`.
2. Resolve linked document claims as evidence for `unit`, `building`, and `housing_company`.
3. Resolve linked transaction claims as evidence for `offering` or `unit`.
4. Project source renovation events into rebuildable system/profile read models.
5. Preserve source claim/event IDs in resolved value provenance.
6. Mark affected targets dirty from source projection and link changes, then resolve them through canonical queue jobs.

Acceptance checks:

- a source claim can be traced to every resolved value that used it;
- changing a link and rerunning resolution moves derived values to the new target;
- unmatched source claims remain available but do not create canonical values.
- dirty target jobs are idempotent and can be safely retried.

### Phase 6: Project Profiles

1. Implement profile projection by target type and target ID.
2. Apply manual overrides first.
3. Apply policy-specific source scoring.
4. Store selected, supporting, and rejected claim IDs.
5. Store conflict status and reason.

Acceptance checks:

- conflict examples show selected and rejected provenance;
- manager certificate values beat listing values for document-preferred dimensions;
- fresh listings beat stale listings for freshness-sensitive dimensions;
- stable identity dimensions do not change only because a newer listing disagrees.

### Phase 7: Project Profiles And Systems

1. Project `property_dimension_profiles` from resolved values.
2. Project `property_system_profiles` from source/manual renovation events.
3. Move valuation input reads to `property_dimension_profiles`.
4. Move detail API reads to `property_dimension_profiles` and `property_system_profiles`.
5. Move quality/score outputs to `score.*` dimensions or remove old score projections.

Acceptance checks:

- valuation reads no old derived table;
- property detail reads no old derived table;
- profile JSON paths match catalog profile mappings;
- profile rebuild is deterministic from values/events.

### Phase 8: Backfill And Validate

1. Backfill source claims/events for all source listings, PDFs, and transactions.
2. Run entity linking.
3. Fan out listing dimension rebuild jobs.
4. Resolve dirty targets through canonical jobs.
5. Project profiles and systems.
6. Validate sample cases.

Sample cases:

- same unit with Shortcut and Frontdoor listings;
- same building with several listings;
- listing with manager certificate;
- listing without manager certificate;
- conflicting building facts;
- fresh listing with stale certificate;
- old listing with fresh certificate;
- transaction matched to listing/offering/unit.

Acceptance checks:

- no source row needed by valuation is missing source claims;
- canonical profiles exist for linked units/buildings/companies;
- unmatched listings remain inspectable through source claims;
- transaction evidence is visible after matching;
- repeated full rebuild produces the same resolved values except timestamps/run IDs.
