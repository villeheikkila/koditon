ALTER TABLE public.property_dimension_claims
    DROP CONSTRAINT IF EXISTS property_dimension_claims_source_claim_id_fkey;
ALTER TABLE public.property_dimension_claims
    ADD CONSTRAINT property_dimension_claims_source_claim_id_fkey
    FOREIGN KEY (source_claim_id)
    REFERENCES public.property_dimension_claims(property_dimension_claim_id)
    ON DELETE CASCADE;
