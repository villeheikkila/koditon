CREATE TABLE IF NOT EXISTS public.property_documents (
    property_document_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT property_documents_pkey PRIMARY KEY,
    property_offering_id uuid NOT NULL CONSTRAINT property_documents_property_offering_id_fkey REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    property_unit_id uuid CONSTRAINT property_documents_property_unit_id_fkey REFERENCES public.property_units(property_unit_id) ON DELETE SET NULL,
    physical_building_id uuid CONSTRAINT property_documents_physical_building_id_fkey REFERENCES public.physical_buildings(physical_building_id) ON DELETE SET NULL,
    housing_company_id uuid CONSTRAINT property_documents_housing_company_id_fkey REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    property_document_type text NOT NULL,
    property_document_filename text NOT NULL,
    property_document_mime_type text NOT NULL,
    property_document_size_bytes bigint NOT NULL,
    property_document_sha256 text NOT NULL,
    property_document_bytes bytea NOT NULL,
    property_document_extraction_status text DEFAULT 'uploaded'::text NOT NULL,
    property_document_extraction_error text,
    property_document_uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    property_document_extracted_at timestamp with time zone,
    property_document_created_at timestamp with time zone DEFAULT now() NOT NULL,
    property_document_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_documents_document_type_check CHECK (property_document_type = ANY (ARRAY['manager_certificate'::text])),
    CONSTRAINT property_documents_extraction_status_check CHECK (property_document_extraction_status = ANY (ARRAY['uploaded'::text, 'extracting'::text, 'extracted'::text, 'failed'::text])),
    CONSTRAINT property_documents_mime_type_check CHECK (property_document_mime_type = 'application/pdf'::text),
    CONSTRAINT property_documents_size_bytes_check CHECK (property_document_size_bytes > 0 AND property_document_size_bytes <= 26214400),
    CONSTRAINT property_documents_sha256_check CHECK (property_document_sha256 ~ '^[0-9a-f]{64}$'::text)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_property_documents_offering_type_hash
ON public.property_documents (property_offering_id, property_document_type, property_document_sha256);

CREATE INDEX IF NOT EXISTS idx_property_documents_offering
ON public.property_documents (property_offering_id, property_document_type, property_document_uploaded_at DESC);

CREATE INDEX IF NOT EXISTS idx_property_documents_housing_company
ON public.property_documents (housing_company_id, property_document_type)
WHERE housing_company_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.property_document_extraction_runs (
    property_document_extraction_run_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT property_document_extraction_runs_pkey PRIMARY KEY,
    property_document_id uuid NOT NULL CONSTRAINT property_document_extraction_runs_property_document_id_fkey REFERENCES public.property_documents(property_document_id) ON DELETE CASCADE,
    property_document_extraction_run_model text NOT NULL,
    property_document_extraction_run_prompt_version text NOT NULL,
    property_document_extraction_run_status text NOT NULL,
    property_document_extraction_run_raw_json jsonb,
    property_document_extraction_run_error text,
    property_document_extraction_run_started_at timestamp with time zone DEFAULT now() NOT NULL,
    property_document_extraction_run_finished_at timestamp with time zone,
    CONSTRAINT property_document_extraction_runs_status_check CHECK (property_document_extraction_run_status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text]))
);

CREATE INDEX IF NOT EXISTS idx_property_document_extraction_runs_document
ON public.property_document_extraction_runs (property_document_id, property_document_extraction_run_started_at DESC);
