ALTER TABLE public.property_documents
    ALTER COLUMN property_offering_id DROP NOT NULL;

CREATE UNIQUE INDEX idx_property_documents_detached_type_hash
ON public.property_documents (property_document_type, property_document_sha256)
WHERE property_offering_id IS NULL;

CREATE OR REPLACE FUNCTION public.fnc__relink_property_document_offering(
    p_property_document_id uuid,
    p_target_property_offering_id uuid,
    p_reason text DEFAULT 'property_document_relinked'
)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_offering_id uuid;
    v_old_unit_id uuid;
    v_old_building_id uuid;
    v_old_housing_company_id uuid;
    v_new_unit_id uuid;
    v_new_building_id uuid;
    v_new_housing_company_id uuid;
    v_dirty integer := 0;
BEGIN
    SELECT property_offering_id, property_unit_id, physical_building_id, housing_company_id
    INTO v_old_offering_id, v_old_unit_id, v_old_building_id, v_old_housing_company_id
    FROM public.property_documents
    WHERE property_document_id = p_property_document_id
    FOR UPDATE;
    SELECT po.property_unit_id, pu.physical_building_id, pu.housing_company_id
    INTO v_new_unit_id, v_new_building_id, v_new_housing_company_id
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE po.property_offering_id = p_target_property_offering_id;
    IF v_new_unit_id IS NULL THEN
        RAISE EXCEPTION 'property offering % not found', p_target_property_offering_id USING ERRCODE = 'foreign_key_violation';
    END IF;
    UPDATE public.property_documents
    SET property_offering_id = p_target_property_offering_id,
        property_unit_id = v_new_unit_id,
        physical_building_id = v_new_building_id,
        housing_company_id = v_new_housing_company_id,
        property_document_updated_at = now()
    WHERE property_document_id = p_property_document_id;
    IF v_old_offering_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_offering_dimension_targets_dirty(v_old_offering_id, p_reason || '_old');
    END IF;
    v_dirty := v_dirty + public.fnc__mark_property_offering_dimension_targets_dirty(p_target_property_offering_id, p_reason || '_new');
    IF v_old_building_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('building', v_old_building_id, p_reason || '_old');
    END IF;
    IF v_old_housing_company_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('housing_company', v_old_housing_company_id, p_reason || '_old');
    END IF;
    IF v_new_building_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('building', v_new_building_id, p_reason || '_new');
    END IF;
    IF v_new_housing_company_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('housing_company', v_new_housing_company_id, p_reason || '_new');
    END IF;
    RETURN jsonb_build_object(
        'property_document_id', p_property_document_id,
        'old_property_offering_id', v_old_offering_id,
        'new_property_offering_id', p_target_property_offering_id,
        'old_property_unit_id', v_old_unit_id,
        'new_property_unit_id', v_new_unit_id,
        'old_physical_building_id', v_old_building_id,
        'new_physical_building_id', v_new_building_id,
        'old_housing_company_id', v_old_housing_company_id,
        'new_housing_company_id', v_new_housing_company_id,
        'dirty_targets', v_dirty
    );
END;
$$;
