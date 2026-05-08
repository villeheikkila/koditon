ALTER TABLE public.property_quality_scores
    DROP CONSTRAINT IF EXISTS property_quality_scores_property_quality_score_target_typ_check;
ALTER TABLE public.property_quality_scores
    ADD CONSTRAINT property_quality_scores_property_quality_score_target_typ_check
    CHECK (property_quality_score_target_type = ANY (ARRAY['property_unit','physical_building','housing_company','property_offering']::text[]));
