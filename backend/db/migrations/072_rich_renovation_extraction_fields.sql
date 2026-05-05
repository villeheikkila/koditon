ALTER TABLE public.property_source_offering_renovations
    ADD COLUMN IF NOT EXISTS property_source_offering_renovation_component text,
    ADD COLUMN IF NOT EXISTS property_source_offering_renovation_scope text,
    ADD COLUMN IF NOT EXISTS property_source_offering_renovation_stage text,
    ADD COLUMN IF NOT EXISTS property_source_offering_renovation_responsibility text,
    ADD COLUMN IF NOT EXISTS property_source_offering_renovation_cost_estimate_eur bigint;

DROP INDEX IF EXISTS public.idx_property_source_offering_renovations_unique;

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
