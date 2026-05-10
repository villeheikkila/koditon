WITH latest_extraction AS (
    SELECT DISTINCT ON (property_document_id)
        property_document_id,
        property_document_extraction_source_json
    FROM public.property_document_extractions
    WHERE property_document_extraction_kind = 'manager_certificate'
        AND property_document_extraction_status = 'succeeded'
        AND property_document_extraction_superseded_at IS NULL
    ORDER BY property_document_id, property_document_extraction_extracted_at DESC, property_document_extraction_created_at DESC
),
document_dates AS (
    SELECT
        property_document_id,
        (property_document_extraction_source_json #>> '{document,document_date}')::date AS document_date
    FROM latest_extraction
    WHERE property_document_extraction_source_json #>> '{document,document_date}' ~ '^\d{4}-\d{2}-\d{2}$'
)
UPDATE public.property_dimension_claims claims
SET source_observed_at = document_dates.document_date::timestamptz + interval '12 hours',
    updated_at = now()
FROM document_dates
WHERE claims.source_table = 'property_documents'
    AND claims.source_id = document_dates.property_document_id
    AND claims.extraction_model IS NOT NULL;
