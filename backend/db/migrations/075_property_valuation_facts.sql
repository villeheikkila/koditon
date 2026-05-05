CREATE TABLE IF NOT EXISTS public.property_valuation_facts (
    property_valuation_fact_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_valuation_fact_entity_type text NOT NULL,
    property_valuation_fact_entity_id uuid NOT NULL,
    property_valuation_fact_source_field text NOT NULL,
    property_valuation_fact_section text NOT NULL,
    property_valuation_fact_key text NOT NULL,
    property_valuation_fact_value_kind text NOT NULL,
    property_valuation_fact_value_text text,
    property_valuation_fact_value_number double precision,
    property_valuation_fact_value_bool boolean,
    property_valuation_fact_confidence integer DEFAULT 50 NOT NULL,
    property_valuation_fact_evidence_text text,
    property_valuation_fact_model text,
    property_valuation_fact_prompt_version text,
    property_valuation_fact_created_at timestamptz DEFAULT now() NOT NULL,
    property_valuation_fact_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (property_valuation_fact_entity_type = ANY (ARRAY['sale_listing', 'building', 'transaction', 'document']::text[])),
    CHECK (property_valuation_fact_value_kind = ANY (ARRAY['text', 'number', 'bool']::text[]))
);

INSERT INTO public.property_valuation_facts (
    property_valuation_fact_entity_type,
    property_valuation_fact_entity_id,
    property_valuation_fact_source_field,
    property_valuation_fact_section,
    property_valuation_fact_key,
    property_valuation_fact_value_kind,
    property_valuation_fact_value_text,
    property_valuation_fact_value_number,
    property_valuation_fact_value_bool,
    property_valuation_fact_confidence,
    property_valuation_fact_evidence_text,
    property_valuation_fact_model,
    property_valuation_fact_created_at,
    property_valuation_fact_updated_at
)
SELECT
    'sale_listing',
    sale_listing_id,
    property_source_offering_valuation_fact_source_field,
    property_source_offering_valuation_fact_section,
    property_source_offering_valuation_fact_key,
    property_source_offering_valuation_fact_value_kind,
    property_source_offering_valuation_fact_value_text,
    property_source_offering_valuation_fact_value_number,
    property_source_offering_valuation_fact_value_bool,
    property_source_offering_valuation_fact_confidence,
    property_source_offering_valuation_fact_evidence_text,
    property_source_offering_valuation_fact_model,
    property_source_offering_valuation_fact_created_at,
    property_source_offering_valuation_fact_updated_at
FROM public.property_source_offering_valuation_facts
ON CONFLICT DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS idx_property_valuation_facts_unique
ON public.property_valuation_facts (
    property_valuation_fact_entity_type,
    property_valuation_fact_entity_id,
    property_valuation_fact_source_field,
    property_valuation_fact_section,
    property_valuation_fact_key
);

CREATE INDEX IF NOT EXISTS idx_property_valuation_facts_entity
ON public.property_valuation_facts (
    property_valuation_fact_entity_type,
    property_valuation_fact_entity_id
);

CREATE INDEX IF NOT EXISTS idx_property_valuation_facts_section_key
ON public.property_valuation_facts (
    property_valuation_fact_section,
    property_valuation_fact_key
);

DROP TABLE IF EXISTS public.property_source_offering_valuation_facts;
