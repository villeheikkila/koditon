DROP VIEW IF EXISTS public.dimension_profiles;
DROP VIEW IF EXISTS public.dimension_values;
DROP VIEW IF EXISTS public.dimension_claims;
CREATE TABLE public.dimension_claims (LIKE public.property_dimension_claims INCLUDING DEFAULTS);
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_pkey PRIMARY KEY (property_dimension_claim_id);
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_claim_scope_check CHECK (claim_scope = ANY (ARRAY['source','manual']));
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_confidence_check CHECK (confidence >= 0 AND confidence <= 1);
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_source_reliability_check CHECK (source_reliability >= 0 AND source_reliability <= 1);
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_target_type_check CHECK (target_type = ANY (ARRAY['listing','document','offering','unit','building','housing_company','house']));
ALTER TABLE public.dimension_claims ADD CONSTRAINT dimension_claims_value_kind_check CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']));
CREATE INDEX idx_dimension_claims_dimension ON public.dimension_claims (dimension_key);
CREATE INDEX idx_dimension_claims_source ON public.dimension_claims (source_table, source_id, projection_version);
CREATE INDEX idx_dimension_claims_source_claim ON public.dimension_claims (source_claim_id);
CREATE INDEX idx_dimension_claims_target ON public.dimension_claims (claim_scope, target_type, target_id, dimension_key);
CREATE UNIQUE INDEX idx_dimension_claims_unique_source ON public.dimension_claims (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''), projection_version);
CREATE INDEX idx_dimension_claims_value_gin ON public.dimension_claims USING gin (value jsonb_path_ops);
CREATE TABLE public.dimension_profiles (LIKE public.property_dimension_profiles INCLUDING DEFAULTS);
ALTER TABLE public.dimension_profiles ADD CONSTRAINT dimension_profiles_pkey PRIMARY KEY (target_type, target_id);
ALTER TABLE public.dimension_profiles ADD CONSTRAINT dimension_profiles_target_type_check CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company','house']));
CREATE INDEX idx_dimension_profiles_building_build_year ON public.dimension_profiles ((((dimensions #>> '{building,build_year}'::text[]))::integer)) WHERE target_type = 'building';
CREATE INDEX idx_dimension_profiles_dimensions_gin ON public.dimension_profiles USING gin (dimensions jsonb_path_ops);
CREATE INDEX idx_dimension_profiles_unit_area ON public.dimension_profiles ((((dimensions #>> '{unit,area_m2}'::text[]))::double precision)) WHERE target_type = 'unit';
CREATE INDEX idx_dimension_profiles_unit_total_charge ON public.dimension_profiles ((((dimensions #>> '{charges,total_monthly_eur}'::text[]))::double precision)) WHERE target_type = 'unit';
CREATE TABLE public.dimension_values (LIKE public.property_dimension_values INCLUDING DEFAULTS);
ALTER TABLE public.dimension_values ADD CONSTRAINT dimension_values_pkey PRIMARY KEY (target_type, target_id, dimension_key);
ALTER TABLE public.dimension_values ADD CONSTRAINT dimension_values_confidence_check CHECK (confidence >= 0 AND confidence <= 1);
ALTER TABLE public.dimension_values ADD CONSTRAINT dimension_values_conflict_status_check CHECK (conflict_status = ANY (ARRAY['none','compatible','conflicting','manual_override']));
ALTER TABLE public.dimension_values ADD CONSTRAINT dimension_values_target_type_check CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company','house']));
ALTER TABLE public.dimension_values ADD CONSTRAINT dimension_values_value_kind_check CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']));
CREATE INDEX idx_dimension_values_dimension ON public.dimension_values (dimension_key);
CREATE INDEX idx_dimension_values_selected_claim ON public.dimension_values (selected_claim_id);
CREATE OR REPLACE FUNCTION public.fnc__sync_dimension_claim()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.dimension_claims WHERE property_dimension_claim_id = OLD.property_dimension_claim_id;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        DELETE FROM public.dimension_claims WHERE property_dimension_claim_id = OLD.property_dimension_claim_id;
    END IF;
    INSERT INTO public.dimension_claims SELECT NEW.*;
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_dimension_profile()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.dimension_profiles WHERE target_type = OLD.target_type AND target_id = OLD.target_id;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        DELETE FROM public.dimension_profiles WHERE target_type = OLD.target_type AND target_id = OLD.target_id;
    END IF;
    INSERT INTO public.dimension_profiles SELECT NEW.*;
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_dimension_value()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.dimension_values WHERE target_type = OLD.target_type AND target_id = OLD.target_id AND dimension_key = OLD.dimension_key;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        DELETE FROM public.dimension_values WHERE target_type = OLD.target_type AND target_id = OLD.target_id AND dimension_key = OLD.dimension_key;
    END IF;
    INSERT INTO public.dimension_values SELECT NEW.*;
    RETURN NEW;
END;
$$;
INSERT INTO public.dimension_claims SELECT * FROM public.property_dimension_claims;
INSERT INTO public.dimension_profiles SELECT * FROM public.property_dimension_profiles;
INSERT INTO public.dimension_values SELECT * FROM public.property_dimension_values;
DROP TRIGGER IF EXISTS trg__sync_dimension_claim ON public.property_dimension_claims;
CREATE TRIGGER trg__sync_dimension_claim
AFTER INSERT OR UPDATE OR DELETE ON public.property_dimension_claims
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_dimension_claim();
DROP TRIGGER IF EXISTS trg__sync_dimension_profile ON public.property_dimension_profiles;
CREATE TRIGGER trg__sync_dimension_profile
AFTER INSERT OR UPDATE OR DELETE ON public.property_dimension_profiles
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_dimension_profile();
DROP TRIGGER IF EXISTS trg__sync_dimension_value ON public.property_dimension_values;
CREATE TRIGGER trg__sync_dimension_value
AFTER INSERT OR UPDATE OR DELETE ON public.property_dimension_values
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_dimension_value();
