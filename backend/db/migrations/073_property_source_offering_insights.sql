CREATE TABLE IF NOT EXISTS public.property_source_offering_insights (
    property_source_offering_insight_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    sale_listing_id uuid NOT NULL REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE,
    property_source_offering_insight_source_field text NOT NULL,
    property_source_offering_insight_key text NOT NULL,
    property_source_offering_insight_value text NOT NULL,
    property_source_offering_insight_direction text NOT NULL,
    property_source_offering_insight_severity text NOT NULL,
    property_source_offering_insight_confidence integer DEFAULT 50 NOT NULL,
    property_source_offering_insight_text text,
    property_source_offering_insight_created_at timestamptz DEFAULT now() NOT NULL,
    property_source_offering_insight_updated_at timestamptz DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_property_source_offering_insights_unique
ON public.property_source_offering_insights (
    sale_listing_id,
    property_source_offering_insight_source_field,
    property_source_offering_insight_key
);

CREATE INDEX IF NOT EXISTS idx_property_source_offering_insights_listing
ON public.property_source_offering_insights (sale_listing_id);
