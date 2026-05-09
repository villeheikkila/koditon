# Parsed Dimensions

This lists the dimensions currently parsed or normalized from source data. “Claim key” means the value can be stored in `property_dimension_claims` as `{namespace}.{key}`. Renovations are stored in typed renovation tables instead of generic claims.

## Source Types

- Listing provider fields from `property_source_offerings`, Frontdoor, and Shortcut.
- Listing text LLM extraction from listing layout, description, room, material, building, charge, parking, balcony, sauna, and view text fields.
- Listing renovation LLM extraction from done/planned renovation text.
- Manager certificate PDF LLM extraction from stored `property_documents`.

## Provider Field Claims

These are deterministic claims created directly from normalized listing fields.

| Claim key | Kind | Target | Source field |
| --- | --- | --- | --- |
| `unit.area_m2` | number | `sale_listing` | `sale_listing_area_value` |
| `unit.living_area_m2` | number | `sale_listing` | `sale_listing_living_area_value` |
| `layout.room_layout` | text | `sale_listing` | `sale_listing_room_layout` |
| `layout.room_count` | number | `sale_listing` | `sale_listing_rooms_count` |
| `condition.condition` | text | `sale_listing` | `sale_listing_condition` |
| `unit.sauna` | bool | `sale_listing` | `sale_listing_sauna` |
| `unit.balcony` | bool | `sale_listing` | `sale_listing_balcony` |
| `parking.parking_text` | text | `sale_listing` | `sale_listing_parking_text` |
| `building.build_year` | number | `sale_listing` | `sale_listing_build_year` |
| `building.heating_method` | text | `sale_listing` | `sale_listing_heating_system` |

## Listing Text LLM Claims

The listing valuation extractor can emit any stable snake_case key in the allowed sections below. These are stored as `sale_listing` claims and unsupported keys are still stored, but only cataloged keys are promoted into valuation input logic.

| Section | Preferred keys from the prompt |
| --- | --- |
| `layout` | `room_count`, `bedroom_count`, `kitchen_type`, `has_separate_kitchen`, `has_open_kitchen`, `has_alcove`, `has_walk_in_closet`, `layout_quality`, `awkward_layout` |
| `rooms` | `living_room_count`, `bedroom_count`, `separate_wc_count`, `utility_room`, `work_room` |
| `unit` | `area_m2`, `living_area_m2`, `total_area_m2`, `other_area_m2`, `accessibility`, `premium_features` |
| `floor` | `floor_level`, `total_floors`, `ground_floor`, `top_floor`, `high_floor`, `elevator_relevance` |
| `kitchen` | `condition`, `renovated`, `appliance_level`, `open_kitchen`, `separate_kitchen` |
| `bathroom` | `condition`, `renovated`, `has_washing_machine_space`, `separate_wc`, `moisture_risk` |
| `balcony` | `has_balcony`, `glazing`, `direction`, `size_quality`, `privacy` |
| `sauna` | `has_sauna`, `private_sauna`, `shared_sauna` |
| `storage` | `storage_quality`, `walk_in_closet`, `basement_storage` |
| `parking` | `parking_available`, `parking_type`, `garage_or_carport` |
| `views` | `view_quality`, `view_direction`, `privacy`, `noise_risk` |
| `condition` | `unit_condition`, `surface_renovation_need`, `move_in_ready`, `modernization_need` |
| `materials` | `floor_material`, `wall_material`, `building_material`, `roof_material` |
| `heating` | `heating_method`, `heating_risk`, `energy_efficiency_signal` |
| `building` | `building_material`, `roof_type`, `common_area_quality`, `accessibility` |
| `charges` | `charge_risk`, `financing_charge_signal`, `included_utilities` |
| `location` | `transit_quality`, `service_quality`, `quietness` |

## Manager Certificate PDF Claims

These are extracted from the PDF and stored as claims against the document, housing company, physical building, or property unit.

| Claim key | Kind | Target |
| --- | --- | --- |
| `document.raw_extraction` | json | `document` |
| `document.document_date` | text | `document` |
| `document.issuer` | text | `document` |
| `document.property_manager` | text | `document` |
| `document.warnings` | json | `document` |
| `housing_company.name` | text | `housing_company` |
| `housing_company.business_id` | text | `housing_company` |
| `housing_company.build_year` | number | `housing_company` |
| `housing_company.apartment_count` | number | `housing_company` |
| `site.plot_ownership_type` | text | `housing_company` |
| `building.energy_class` | text | `housing_company` |
| `risk.financial_risk` | text | `housing_company` |
| `risk.maintenance_risk` | text | `housing_company` |
| `risk.repair_backlog_risk` | text | `housing_company` |
| `risk.administrative_legal_risk` | text | `housing_company` |
| `risk.restrictions` | json | `housing_company` |
| `finances.loan_summary` | text | `housing_company` |
| `finances.charge_summary` | text | `housing_company` |
| `building.build_year` | number | `physical_building` |
| `building.floor_count` | number | `physical_building` |
| `building.apartment_count` | number | `physical_building` |
| `building.energy_class` | text | `physical_building` |
| `building.heating_method` | text | `physical_building` |
| `building.material` | text | `physical_building` |
| `building.roof_type` | text | `physical_building` |
| `building.roof_material` | text | `physical_building` |
| `building.elevator` | bool | `physical_building` |
| `unit.apartment_number` | text | `property_unit` |
| `unit.shares` | text | `property_unit` |
| `unit.area_m2` | number | `property_unit` |
| `layout.room_layout` | text | `property_unit` |
| `unit.floor_level` | number | `property_unit` |
| `charges.maintenance_charge_monthly` | number | `property_unit` |
| `charges.capital_charge_monthly` | number | `property_unit` |
| `charges.total_charge_monthly` | number | `property_unit` |
| `charges.debt_share_eur` | number | `property_unit` |
| `risk.shareholder_liability` | text | `property_unit` |

## Renovation Dimensions

Listing renovation extraction and manager certificate extraction both normalize renovations into typed rows.

| Dimension | Notes |
| --- | --- |
| `category` | Listing: `pipe`, `sewer`, `water_supply`, `facade`, `roof`, `window`, `balcony`, `electricity`, `elevator`, `heating`, `ventilation`, `drainage`, `yard`, `common_area`, `bathroom`, `kitchen`, `other`. Manager certificate profile projection accepts `pipe`, `water_supply`, `sewer`, `roof`, `facade`, `window`, `balcony`, `elevator`, `heating`, `ventilation`, `drainage`, `electricity`, `yard`, `common_areas`, `other`. |
| `component` | More specific component, such as sewer, water supply, riser, bathroom, roof type, courtyard, or foundation. |
| `status` | Listing extraction: `done`, `planned`, `unknown`. Manager certificate: `done`, `planned`, `suspected`, `forecast`, `cancelled`, `unknown`. |
| `year` | Single explicit or clearly implied year. |
| `start_year` | Manager certificate renovation window start. |
| `end_year` | Manager certificate renovation window end. |
| `scope` | Listing extraction: `full`, `partial`, `survey`, `maintenance`, `planning`, `unknown`. Manager certificate profile rows: `full`, `partial`, `maintenance`, `unknown`. |
| `stage` | Listing extraction: `need_assessment`, `condition_survey`, `project_planning`, `tendering`, `decision`, `execution`, `completed`, `maintenance`, `unknown`. Manager certificate profile rows: `study`, `condition_assessment`, `planning`, `tendering`, `execution`, `completed`, `unknown`. |
| `responsibility` | `housing_company`, `shareholder`, `mixed`, `unknown`. |
| `cost_estimate_eur` | Explicit cost estimate. |
| `summary` / `text` | Source-supported summary or evidence text. |
| `confidence` | Listing rows use 0-100 integer confidence. Housing company renovation rows use `low`, `medium`, `high`. |
| `evidence_level` | `ad_only` for listing-derived rows, `manager_certificate` for certificate-derived rows. |

## Valuation Catalog Dimensions

These claim keys are explicitly recognized by `valuation/dimension_catalog.go` and are eligible for valuation input consumption.

| Claim key | Kind |
| --- | --- |
| `balcony.glazing` | bool |
| `balcony.has_balcony` | bool |
| `unit.balcony` | bool |
| `unit.sauna` | bool |
| `unit.area_m2` | number |
| `unit.living_area_m2` | number |
| `unit.total_area_m2` | number |
| `unit.other_area_m2` | number |
| `unit.floor_level` | number |
| `unit.accessibility` | text |
| `layout.room_layout` | text |
| `layout.room_count` | number |
| `layout.kitchen_type` | text |
| `layout.has_separate_kitchen` | bool |
| `layout.has_open_kitchen` | bool |
| `layout.has_alcove` | bool |
| `layout.awkward_layout` | bool |
| `layout.layout_quality` | text |
| `layout.separate_wc_count` | number |
| `rooms.separate_wc_count` | number |
| `sauna.has_sauna` | bool |
| `sauna.private_sauna` | bool |
| `storage.storage_quality` | text |
| `views.view_quality` | text |
| `views.noise_risk` | bool |
| `condition.surface_renovation_need` | bool |
| `condition.modernization_need` | bool |
| `kitchen.renovated` | bool |
| `bathroom.renovated` | bool |
| `heating.heating_method` | text |
| `building.accessibility` | text |
| `building.common_area_quality` | text |
| `building.build_year` | number |
| `building.energy_class` | text |
| `building.heating_method` | text |
| `building.material` | text |
| `building.roof_type` | text |
| `building.roof_material` | text |
| `building.elevator` | bool |
| `building.apartment_count` | number |
| `site.plot_ownership_type` | text |
| `charges.maintenance_charge_monthly` | number |
| `charges.capital_charge_monthly` | number |
| `charges.total_charge_monthly` | number |
| `charges.debt_share_eur` | number |
| `charges.charge_risk` | text |
| `risk.shareholder_liability` | text |
| `risk.financial_risk` | text |
| `risk.maintenance_risk` | text |
| `risk.repair_backlog_risk` | text |

## Profile Projection Dimensions

These are the fields currently projected into durable normalized profiles.

### Apartment Profile

- Identity links: `property_unit_id`, `housing_company_id`, `physical_building_id`, `source_sale_listing_id`.
- Size/layout/floor: `area_m2`, `living_area_m2`, `room_layout`, `room_count`, `bedroom_count`, `floor_level`, `total_floors`.
- Layout quality: `kitchen_type`, `layout_quality`, `awkward_layout`.
- Condition: `condition`, `kitchen_condition`, `bathroom_condition`, `surface_renovation_need`, `modernization_need`.
- Features: `sauna`, `balcony`, `balcony_glazing`, `parking_type`, `storage_quality`, `view_quality`, `noise_risk`, `accessibility`.
- Financial unit fields: `maintenance_charge_monthly`, `capital_charge_monthly`, `total_charge_monthly`, `debt_share_eur`, `shareholder_liability`.
- Metadata: `confidence`, `updated_at`.

### Building Profile

- `build_year`
- `floor_count`
- `apartment_count`
- `energy_class`
- `heating_method`
- `material`
- `roof_type`
- `roof_material`
- `elevator`
- `confidence`

### Housing Company Profile

- `name`
- `business_id`
- `build_year`
- `apartment_count`
- `plot_ownership_type`
- `energy_class`
- `maintenance_risk`
- `financial_risk`
- `repair_backlog_risk`
- `confidence`

### Housing Company Systems

- `system_type`: `pipes`, `water_supply`, `sewer`, `roof`, `facade`, `windows`, `balconies`, `elevator`, `heating`, `ventilation`, `drainage`, `electrical`, `yard`, `common_areas`.
- `status`: `planned`, `renewed`, `unknown`.
- `last_renovated_year`
- `next_expected_start_year`
- `next_expected_end_year`
- `confidence`
- `evidence_level`
- `summary`

## Gaps To Note

- The listing text prompt allows more keys than the valuation catalog currently consumes.
- `charges.capital_charge_monthly` is stored in the dimension layer, but valuation input currently carries it only as a note because `ChargesInput` has no dedicated capital-charge field.
- `document.raw_extraction`, `document.warnings`, `risk.restrictions`, `finances.loan_summary`, and `finances.charge_summary` are stored claims but are not yet deeply projected into the valuation model.
