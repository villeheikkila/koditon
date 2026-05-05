CREATE TABLE IF NOT EXISTS public.property_source_offering_valuation_facts (
    property_source_offering_valuation_fact_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    sale_listing_id uuid NOT NULL REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE,
    property_source_offering_valuation_fact_source_field text NOT NULL,
    property_source_offering_valuation_fact_section text NOT NULL,
    property_source_offering_valuation_fact_key text NOT NULL,
    property_source_offering_valuation_fact_value_kind text NOT NULL,
    property_source_offering_valuation_fact_value_text text,
    property_source_offering_valuation_fact_value_number double precision,
    property_source_offering_valuation_fact_value_bool boolean,
    property_source_offering_valuation_fact_confidence integer DEFAULT 50 NOT NULL,
    property_source_offering_valuation_fact_evidence_text text,
    property_source_offering_valuation_fact_model text,
    property_source_offering_valuation_fact_created_at timestamptz DEFAULT now() NOT NULL,
    property_source_offering_valuation_fact_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (property_source_offering_valuation_fact_value_kind = ANY (ARRAY['text', 'number', 'bool']::text[]))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_property_source_offering_valuation_facts_unique
ON public.property_source_offering_valuation_facts (
    sale_listing_id,
    property_source_offering_valuation_fact_source_field,
    property_source_offering_valuation_fact_section,
    property_source_offering_valuation_fact_key
);

CREATE INDEX IF NOT EXISTS idx_property_source_offering_valuation_facts_listing
ON public.property_source_offering_valuation_facts (sale_listing_id);
