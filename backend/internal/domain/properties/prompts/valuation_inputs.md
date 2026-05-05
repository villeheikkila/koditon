Extract canonical apartment valuation inputs from Finnish real-estate listing fields.

Return only facts directly supported by the provided fields. Do not infer facts that are not stated, except simple layout parsing from room_layout.

Use these sections:
- layout
- rooms
- unit
- floor
- kitchen
- bathroom
- balcony
- sauna
- storage
- parking
- views
- condition
- materials
- heating
- building
- charges
- location

Use stable snake_case keys. Prefer these keys when applicable:
- layout: room_count, bedroom_count, kitchen_type, has_separate_kitchen, has_open_kitchen, has_alcove, has_walk_in_closet, layout_quality, awkward_layout
- rooms: living_room_count, bedroom_count, separate_wc_count, utility_room, work_room
- unit: area_m2, living_area_m2, total_area_m2, other_area_m2, accessibility, premium_features
- floor: floor_level, total_floors, ground_floor, top_floor, high_floor, elevator_relevance
- kitchen: condition, renovated, appliance_level, open_kitchen, separate_kitchen
- bathroom: condition, renovated, has_washing_machine_space, separate_wc, moisture_risk
- balcony: has_balcony, glazing, direction, size_quality, privacy
- sauna: has_sauna, private_sauna, shared_sauna
- storage: storage_quality, walk_in_closet, basement_storage
- parking: parking_available, parking_type, garage_or_carport
- views: view_quality, view_direction, privacy, noise_risk
- condition: unit_condition, surface_renovation_need, move_in_ready, modernization_need
- materials: floor_material, wall_material, building_material, roof_material
- heating: heating_method, heating_risk, energy_efficiency_signal
- building: building_material, roof_type, common_area_quality, accessibility
- charges: charge_risk, financing_charge_signal, included_utilities
- location: transit_quality, service_quality, quietness

Value rules:
- For yes/no facts use value_kind "bool" and value_bool.
- For numeric facts use value_kind "number" and value_number.
- For categorical or descriptive facts use value_kind "text" and value_text.
- Confidence is 0-100.
- source_field must be the exact input field name that supports the fact.
- evidence_text should be a short supporting phrase, not a long quote.

Input fields:

room_layout:
{{room_layout}}

rooms_count:
{{rooms_count}}

bedrooms_count:
{{bedrooms_count}}

area_m2:
{{area_m2}}

living_area_m2:
{{living_area_m2}}

total_area_m2:
{{total_area_m2}}

other_area_m2:
{{other_area_m2}}

floor_level:
{{floor_level}}

total_floors:
{{total_floors}}

floor_text:
{{floor_text}}

condition:
{{condition}}

sauna:
{{sauna}}

balcony:
{{balcony}}

parking_text:
{{parking_text}}

description_text:
{{description_text}}

additional_info_text:
{{additional_info_text}}

kitchen_description_text:
{{kitchen_description_text}}

bathroom_description_text:
{{bathroom_description_text}}

storage_description_text:
{{storage_description_text}}

floor_materials_description_text:
{{floor_materials_description_text}}

wall_materials_description_text:
{{wall_materials_description_text}}

balcony_description_text:
{{balcony_description_text}}

sauna_description_text:
{{sauna_description_text}}

views_description_text:
{{views_description_text}}

building_material:
{{building_material}}

heating_system:
{{heating_system}}

roof_type:
{{roof_type}}

roof_material:
{{roof_material}}

car_storage_text:
{{car_storage_text}}

building_description_text:
{{building_description_text}}

building_other_info_text:
{{building_other_info_text}}

charges_text:
{{charges_text}}
