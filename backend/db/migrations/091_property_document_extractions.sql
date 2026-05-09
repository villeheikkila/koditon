CREATE TABLE public.property_document_extractions (
    property_document_extraction_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT property_document_extractions_pkey PRIMARY KEY,
    property_document_id uuid NOT NULL CONSTRAINT property_document_extractions_property_document_id_fkey REFERENCES public.property_documents(property_document_id) ON DELETE CASCADE,
    property_document_extraction_kind text NOT NULL,
    property_document_extraction_schema_version text NOT NULL,
    property_document_extraction_model text NOT NULL,
    property_document_extraction_prompt_version text NOT NULL,
    property_document_extraction_source_json jsonb NOT NULL,
    property_document_extraction_status text DEFAULT 'succeeded'::text NOT NULL,
    property_document_extraction_error text,
    property_document_extraction_created_at timestamptz DEFAULT now() NOT NULL,
    property_document_extraction_extracted_at timestamptz DEFAULT now() NOT NULL,
    property_document_extraction_superseded_at timestamptz,
    CONSTRAINT property_document_extractions_kind_check CHECK (property_document_extraction_kind = ANY (ARRAY['manager_certificate'::text])),
    CONSTRAINT property_document_extractions_status_check CHECK (property_document_extraction_status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'superseded'::text])),
    CONSTRAINT property_document_extractions_schema_version_check CHECK (property_document_extraction_schema_version <> '')
);

CREATE UNIQUE INDEX idx_property_document_extractions_latest
ON public.property_document_extractions (property_document_id, property_document_extraction_kind)
WHERE property_document_extraction_superseded_at IS NULL;

CREATE INDEX idx_property_document_extractions_document
ON public.property_document_extractions (property_document_id, property_document_extraction_created_at DESC);

---- create above / drop below ----

DROP TABLE IF EXISTS public.property_document_extractions;
