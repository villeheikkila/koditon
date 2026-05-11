# Canonical Building And Housing Company Migration

This is the target shape for the core property model.

```text
housing_company
  -> physical_buildings
    -> property_units
      -> property_offerings
        -> property_source_offerings
          -> provider source rows
```

Map and detail pages are building/company first. Listings, provider pages, certificates, and transactions are evidence or child records, not the primary spatial entity.

Source rows remain durable inputs. Canonical rows, link rows, claims, resolved values, profiles, search read models, and map read models are rebuildable.

## Correct User Model

### Map

The map shows every known place-level target:

- `physical_buildings` with coordinates.
- `housing_companies` with coordinates when no building row exists yet.

Clicking a marker opens a building or housing company page. It must not open a listing/offering page.

### Housing Company Page

The housing company page shows:

- resolved housing company facts;
- physical buildings in the company;
- canonical listings/offerings under those buildings;
- source evidence from Frontdoor, Shortcut, documents, and listing-derived facts;
- certificates/documents attached to the company, its buildings, units, or listings.

### Building Page

The building page shows:

- resolved building facts;
- parent housing company link;
- units in the building;
- canonical listings/offerings for those units;
- source evidence from building pages, listings, and documents.

### Offering Page

The offering page is a child page:

- canonical offering facts;
- unit link;
- building link;
- housing company link;
- source listings from Frontdoor/Shortcut as provenance.

Provider URLs are secondary source evidence. They are not the stable app navigation path.

## Data Model Decision

The current schema is good enough to evolve. Do not replace the whole model.

Keep:

- `housing_companies`
- `physical_buildings`
- `property_units`
- `property_offerings`
- `property_source_offerings`
- raw provider tables
- `property_documents`
- `property_dimension_claims`
- `property_renovation_events`
- `property_dimension_values`
- `property_dimension_profiles`

Fix:

- Make `physical_buildings` first-class in API and map read models.
- Normalize source links into one generic table.
- Stop using provider/source pages as primary UI links.
- Stop returning huge ungrouped listing/source lists from target overview.

## New Generic Source Link Table

Add one generic source link table for all target types.

```sql
CREATE TABLE public.property_target_sources (
    property_target_source_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_provider text NOT NULL,
    source_kind text NOT NULL,
    source_table text NOT NULL,
    source_id uuid,
    source_id_value text NOT NULL,
    source_external_id text,
    source_url text,
    link_status text NOT NULL DEFAULT 'confirmed',
    link_method text NOT NULL,
    link_score integer NOT NULL DEFAULT 100,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company','document','transaction']::text[])),
    CHECK (link_status = ANY (ARRAY['confirmed','candidate','rejected']::text[]))
);

CREATE UNIQUE INDEX idx_property_target_sources_unique_source
ON public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id_value
);

CREATE INDEX idx_property_target_sources_target
ON public.property_target_sources (target_type, target_id, link_status);

CREATE INDEX idx_property_target_sources_source
ON public.property_target_sources (source_table, source_id_value, link_status);
```

Backfill from:

- `property_offering_sources` -> `target_type = 'offering'`
- `housing_company_sources` -> `target_type = 'housing_company'`
- source building rows connected through listings -> `target_type = 'building'` when a `physical_building_id` exists, otherwise `housing_company`
- `property_documents` attachments -> target source links only if needed for document provenance; document attachment still remains on `property_documents`

Keep old specific source-link tables during migration if needed, but new API/read models must read `property_target_sources`.

After validation, either drop old source-link tables or keep them only as low-level compatibility tables behind sync code. Do not expose them in API.

## Coordinates

Map coordinates must come from target entities.

For `physical_buildings`:

- use `physical_building_latitude`
- use `physical_building_longitude`

For `housing_companies` fallback:

- use `housing_company_geom`
- only include company marker if it has no building marker with coordinates.

Backfill building coordinates from:

- Frontdoor building page coordinates;
- Shortcut building page coordinates;
- linked listing coordinates;
- housing company geom fallback.

Add indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_physical_buildings_lat_lng
ON public.physical_buildings (physical_building_latitude, physical_building_longitude)
WHERE physical_building_latitude IS NOT NULL
  AND physical_building_longitude IS NOT NULL;
```

## API Shape

Replace current target overview behavior with explicit resources.

### Map Endpoint

`GET /api/v1/property-targets/map`

Returns:

```json
{
  "markers": [
    {
      "target": { "type": "building", "id": "..." },
      "fallback_target": { "type": "housing_company", "id": "..." },
      "title": "Asunto Oy ...",
      "address": "...",
      "city": "...",
      "postal": "...",
      "lat": 60.0,
      "lng": 24.0,
      "building_count": 1,
      "unit_count": 10,
      "offering_count": 3,
      "source_count": 8,
      "document_count": 1
    }
  ]
}
```

Rules:

- Prefer building markers.
- Include housing company marker only when no building marker exists.
- Query filter searches company name, address, city, postal, business id, and source identifiers.
- No provider URL links in marker payload.

### Target Detail Endpoint

`GET /api/v1/property-targets/{targetType}/{targetID}`

For `building` and `housing_company`, return:

- `overview`
- `resolved_values`
- `renovation_events`
- `documents`
- `buildings`
- `units`
- `offerings`
- `sources`

Offering children must be canonical links:

```json
{
  "offerings": [
    {
      "target": { "type": "offering", "id": "..." },
      "unit_target": { "type": "unit", "id": "..." },
      "title": "...",
      "layout": "...",
      "area_m2": 78,
      "asking_price_eur": 462900,
      "last_seen_at": "..."
    }
  ]
}
```

Provider pages are source evidence:

```json
{
  "sources": [
    {
      "provider": "frontdoor",
      "kind": "building",
      "source_table": "frontdoor_buildings",
      "source_id_value": "...",
      "url": "...",
      "link_status": "confirmed"
    }
  ]
}
```

The UI should show canonical children before source evidence.

## Jobs

Add jobs or update existing canonical jobs:

1. `canonical_backfill_target_sources`
   - backfills `property_target_sources` from existing link/source tables.

2. `canonical_backfill_building_coordinates`
   - computes `physical_building_latitude/longitude`.

3. `canonical_rebuild_spatial_read_model`
   - optional if map remains direct SQL; required if map needs denormalized table later.

4. Existing dirty target resolution jobs remain:
   - `canonical_resolve_dimension_target`
   - `canonical_resolve_dirty_dimension_targets`

Do not use triggers for semantic linking or profile projection. Triggers may only maintain local timestamps or dirty flags.

## Migration Order

### Phase 1: Schema

- Add `property_target_sources`.
- Add coordinate index for `physical_buildings`.
- Add missing API structs for building/company detail children.

### Phase 2: Backfill Source Links

- Backfill offering source links from `property_offering_sources`.
- Backfill housing company source links from `housing_company_sources`.
- Backfill building source links from source building rows reachable through linked listings.
- Mark old and new targets dirty where links imply claim resolution changes.

### Phase 3: Backfill Coordinates

- Fill physical building coordinates from explicit provider building rows.
- Fill from listing coordinates where provider building rows are absent.
- Fill company geom where building has no better coordinate.

### Phase 4: Replace Map

- Map endpoint returns building/company targets only.
- Marker click links to `/target/building/{id}` or `/target/housing_company/{id}`.
- Remove offering/listing marker behavior.

### Phase 5: Replace Detail Pages

- Building/company pages show canonical child offerings first.
- Provider source links move into a source evidence section.
- Offering pages show source listings, parent unit, parent building, and parent housing company.

### Phase 6: Cleanup

- Remove old overview source-link hacks.
- Stop reading `housing_company_sources` and `property_offering_sources` directly from API.
- Keep old tables only if sync internals still write them; otherwise drop after validation.

## Validation Queries

Map coverage:

```sql
SELECT
    count(*) FILTER (WHERE physical_building_latitude IS NOT NULL AND physical_building_longitude IS NOT NULL) AS mapped_buildings,
    count(*) AS total_buildings
FROM public.physical_buildings;
```

Company fallback coverage:

```sql
SELECT count(*)
FROM public.housing_companies hc
WHERE hc.housing_company_geom IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM public.physical_buildings pb
      WHERE pb.housing_company_id = hc.housing_company_id
        AND pb.physical_building_latitude IS NOT NULL
        AND pb.physical_building_longitude IS NOT NULL
  );
```

Canonical offerings per company:

```sql
SELECT hc.housing_company_id, count(DISTINCT po.property_offering_id) AS offerings
FROM public.housing_companies hc
LEFT JOIN public.property_units pu ON pu.housing_company_id = hc.housing_company_id
LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
GROUP BY hc.housing_company_id;
```

Generic source links:

```sql
SELECT target_type, source_provider, source_kind, link_status, count(*)
FROM public.property_target_sources
GROUP BY target_type, source_provider, source_kind, link_status
ORDER BY target_type, source_provider, source_kind, link_status;
```

No orphan map offering-primary behavior:

```sql
-- This should be zero after the map endpoint is replaced.
-- There should be no frontend route from map marker directly to offering target.
```

## Done Criteria

- Map shows building/company targets only.
- Every marker opens building/company detail.
- Building/company detail shows canonical child listings/offerings.
- Provider links are visible only as source evidence.
- Claims and renovation events resolve through current links.
- Relinking a unit/building/company dirties old and new targets.
- A disappeared provider page does not break internal navigation.
