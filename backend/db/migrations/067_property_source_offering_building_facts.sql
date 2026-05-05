ALTER TABLE public.property_source_offerings
    ADD COLUMN IF NOT EXISTS sale_listing_housing_company_name text,
    ADD COLUMN IF NOT EXISTS sale_listing_housing_company_business_id text,
    ADD COLUMN IF NOT EXISTS sale_listing_building_material text,
    ADD COLUMN IF NOT EXISTS sale_listing_heating_system text,
    ADD COLUMN IF NOT EXISTS sale_listing_roof_type text,
    ADD COLUMN IF NOT EXISTS sale_listing_roof_material text,
    ADD COLUMN IF NOT EXISTS sale_listing_apartment_count integer,
    ADD COLUMN IF NOT EXISTS sale_listing_car_storage_text text,
    ADD COLUMN IF NOT EXISTS sale_listing_building_description_text text,
    ADD COLUMN IF NOT EXISTS sale_listing_building_other_info_text text,
    ADD COLUMN IF NOT EXISTS sale_listing_latitude double precision,
    ADD COLUMN IF NOT EXISTS sale_listing_longitude double precision;

CREATE TABLE IF NOT EXISTS public.property_source_offering_renovations (
    property_source_offering_renovation_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    sale_listing_id uuid NOT NULL REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE,
    property_source_offering_renovation_source_field text NOT NULL,
    property_source_offering_renovation_category text NOT NULL,
    property_source_offering_renovation_status text NOT NULL,
    property_source_offering_renovation_year integer,
    property_source_offering_renovation_component text,
    property_source_offering_renovation_scope text,
    property_source_offering_renovation_stage text,
    property_source_offering_renovation_responsibility text,
    property_source_offering_renovation_cost_estimate_eur bigint,
    property_source_offering_renovation_text text,
    property_source_offering_renovation_confidence integer DEFAULT 100 NOT NULL,
    property_source_offering_renovation_created_at timestamptz DEFAULT now() NOT NULL,
    property_source_offering_renovation_updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_source_offering_renovation_status_check CHECK (property_source_offering_renovation_status = ANY (ARRAY['done'::text, 'planned'::text, 'unknown'::text]))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_property_source_offering_renovations_unique
ON public.property_source_offering_renovations (
    sale_listing_id,
    property_source_offering_renovation_source_field,
    property_source_offering_renovation_category,
    property_source_offering_renovation_status,
    COALESCE(property_source_offering_renovation_year, 0),
    COALESCE(property_source_offering_renovation_component, ''),
    COALESCE(property_source_offering_renovation_stage, '')
);

CREATE INDEX IF NOT EXISTS idx_property_source_offering_renovations_listing
ON public.property_source_offering_renovations (sale_listing_id);

ALTER TABLE public.frontdoor_building_announcements
    ADD COLUMN IF NOT EXISTS frontdoor_building_announcement_data_normalized_at timestamptz,
    ADD COLUMN IF NOT EXISTS frontdoor_building_announcement_data_normalized_version integer DEFAULT 0 NOT NULL;

CREATE INDEX IF NOT EXISTS idx_frontdoor_building_announcements_normalized
ON public.frontdoor_building_announcements (
    frontdoor_building_announcement_data_normalized_at,
    frontdoor_building_announcement_data_normalized_version
);

DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
