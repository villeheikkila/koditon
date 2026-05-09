ALTER TABLE public.property_dimension_values
    DROP CONSTRAINT IF EXISTS property_dimension_values_selected_claim_id_fkey;
ALTER TABLE public.property_dimension_values
    ADD CONSTRAINT property_dimension_values_selected_claim_id_fkey
    FOREIGN KEY (selected_claim_id)
    REFERENCES public.property_dimension_claims(property_dimension_claim_id)
    ON DELETE CASCADE;
