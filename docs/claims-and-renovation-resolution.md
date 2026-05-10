# Claims And Renovation Resolution

The current property model separates evidence extraction from identity linking and resolved read models.

```text
source rows and documents -> source claims/events -> canonical links -> resolved values -> API profiles
```

Source claims and source events are the factual evidence. Link rows decide which real-world entity the evidence belongs to. Resolvers decide which value to show or use downstream.

## Source Data Boundary

Source data is kept as the durable input:

- provider listing rows from Frontdoor and Shortcut
- normalized `property_source_offerings`
- source price transaction rows
- uploaded `property_documents`
- `property_document_extractions`
- extraction/projection run records used for auditability

Canonical entities, links, claims, renovation events, profiles, and search/index read models are rebuildable from source data.

## Claims

Facts are stored as source-scoped rows in `property_dimension_claims`.

Projectors write claims from one source at a time:

- listing provider fields become listing claims
- listing text LLM output becomes listing claims
- manager certificate PDF extraction becomes document, housing company, building, and unit claims when the document is attached
- manual corrections can write directly to canonical targets

Claims keep provenance:

- `source_table`
- `source_id`
- `source_field`
- `source_observed_at`
- confidence and source reliability metadata
- projection version and projection run

The resolver does not need to mutate source claims when links change. It reads claims through the current link graph.

## Links

Identity and attachment decisions live outside claims.

The link layer answers:

- which source listings are the same canonical offering
- which offering belongs to which unit
- which unit belongs to which building
- which building belongs to which housing company
- which uploaded document belongs to which offering, unit, building, or housing company
- which transaction belongs to which listing, offering, unit, building, or housing company

Relinking should update link rows, then dirty both the old target and the new target. Source evidence remains intact.

This makes moving a PDF, listing, or transaction between canonical entities cheap: change the link, re-resolve affected targets, and rebuild profiles.

## Value Resolution

Resolved dimension values are computed from linked source claims.

Manual overrides win first. Otherwise resolution is dimension-specific:

- stable identity and building facts prefer official document evidence
- current commercial/listing values can prefer fresher listing evidence
- numeric values can use source priority, confidence, freshness, or consensus
- resolved rows retain selected source claim provenance

Manager certificate facts usually beat listing text for stable housing company and building facts. Newer listings can beat old documents for time-sensitive values when freshness matters, such as current charges, planned work, and future maintenance signals.

## Renovation Events

Renovations are not generic claims. They are typed source events in `property_renovation_events`.

Listing renovation sources:

- provider renovation fields in `property_source_offering_renovations`
- listing renovation LLM extraction from done/planned renovation text

Document renovation sources:

- manager certificate PDF extraction

Listing rows are projected into `property_renovation_events` by Go code in `ProjectListingRenovationEvents`. This runs after provider renovation refreshes, after LLM renovation extraction, and during listing dimension rebuild/backfill.

Manager certificate renovation events are attached to `property_documents` as source events and include the document observation date when available.

The listing API reads renovation events through the linked housing company/building. It deduplicates evidence in Go:

- manager certificate evidence gets higher authority
- listing evidence remains valid evidence
- planned, suspected, and forecast events decay by source age
- newer listing evidence can beat older certificate evidence for future-looking work
- old listing renovation rows are used only as fallback when no event rows exist

This lets one housing company accumulate renovation evidence from many listings and certificates while each listing page still shows the best current building/company view.

## Jobs

Semantic work runs through queue jobs, not triggers.

Important jobs:

- `canonical_rebuild_dimension_layer_listing`
- `canonical_resolve_dimension_target`
- `canonical_resolve_dirty_dimension_targets`
- `canonical_rebuild_dimension_layer_backfill`
- `canonical_extract_manager_certificate`
- `canonical_project_manager_certificate`

Source writes and projection paths mark affected targets dirty. Queue jobs perform rebuilds and resolution in small idempotent units.

Database functions are still acceptable for local set-based helpers and dirty-row bookkeeping. Business policy, semantic resolution, and evolving source precedence should live in Go jobs and services.

## Display Model

Listing pages show data from the canonical model, not only from the current source listing.

For a listing:

1. Resolve the listing to its canonical offering.
2. Follow offering -> unit -> building -> housing company.
3. Read resolved values and typed events for those canonical entities.
4. Show the listing-specific unit details together with the parent building and housing company facts.
5. Link to the parent housing company so related listings, certificates, and transactions share the same context.

The key invariant is that source evidence is appendable and movable by links, while displayed values are rebuildable outputs.
