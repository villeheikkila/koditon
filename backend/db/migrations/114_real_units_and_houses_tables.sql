DROP VIEW IF EXISTS public.units;
DROP VIEW IF EXISTS public.houses;
CREATE TABLE public.units (
    unit_id uuid PRIMARY KEY,
    housing_company_id uuid NOT NULL,
    physical_building_id uuid,
    identity_key text NOT NULL,
    address_norm text,
    apartment text,
    floor_level integer,
    area_m2 double precision,
    room_layout text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX units_identity_key_key
ON public.units (identity_key);
CREATE INDEX idx_units_housing_company
ON public.units (housing_company_id);
CREATE INDEX idx_units_physical_building
ON public.units (physical_building_id) WHERE physical_building_id IS NOT NULL;
CREATE TABLE public.houses (
    house_id uuid PRIMARY KEY,
    identity_key text NOT NULL,
    address_norm text,
    postal_norm text,
    city_norm text,
    latitude double precision,
    longitude double precision,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX houses_identity_key_key
ON public.houses (identity_key);
CREATE INDEX idx_houses_address
ON public.houses (postal_norm, city_norm, address_norm);
CREATE INDEX idx_houses_lat_lng
ON public.houses (latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE OR REPLACE FUNCTION public.fnc__sync_unit_from_property_unit()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.units WHERE unit_id = OLD.property_unit_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.units (
        unit_id,
        housing_company_id,
        physical_building_id,
        identity_key,
        address_norm,
        apartment,
        floor_level,
        area_m2,
        room_layout,
        created_at,
        updated_at
    )
    VALUES (
        NEW.property_unit_id,
        NEW.housing_company_id,
        NEW.physical_building_id,
        NEW.property_unit_identity_key,
        NEW.property_unit_address_norm,
        NULL,
        NEW.property_unit_floor_level,
        NEW.property_unit_area_value,
        NEW.property_unit_room_layout,
        NEW.property_unit_created_at,
        NEW.property_unit_updated_at
    )
    ON CONFLICT (unit_id) DO UPDATE SET
        housing_company_id = EXCLUDED.housing_company_id,
        physical_building_id = EXCLUDED.physical_building_id,
        identity_key = EXCLUDED.identity_key,
        address_norm = EXCLUDED.address_norm,
        apartment = EXCLUDED.apartment,
        floor_level = EXCLUDED.floor_level,
        area_m2 = EXCLUDED.area_m2,
        room_layout = EXCLUDED.room_layout,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_house_from_property_house()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.houses WHERE house_id = OLD.property_house_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.houses (
        house_id,
        identity_key,
        address_norm,
        postal_norm,
        city_norm,
        latitude,
        longitude,
        created_at,
        updated_at
    )
    VALUES (
        NEW.property_house_id,
        NEW.property_house_identity_key,
        NEW.property_house_address_norm,
        NEW.property_house_postal_norm,
        NEW.property_house_city_norm,
        NEW.property_house_latitude,
        NEW.property_house_longitude,
        NEW.property_house_created_at,
        NEW.property_house_updated_at
    )
    ON CONFLICT (house_id) DO UPDATE SET
        identity_key = EXCLUDED.identity_key,
        address_norm = EXCLUDED.address_norm,
        postal_norm = EXCLUDED.postal_norm,
        city_norm = EXCLUDED.city_norm,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$;
INSERT INTO public.units (
    unit_id,
    housing_company_id,
    physical_building_id,
    identity_key,
    address_norm,
    apartment,
    floor_level,
    area_m2,
    room_layout,
    created_at,
    updated_at
)
SELECT
    property_unit_id,
    housing_company_id,
    physical_building_id,
    property_unit_identity_key,
    property_unit_address_norm,
    NULL::text,
    property_unit_floor_level,
    property_unit_area_value,
    property_unit_room_layout,
    property_unit_created_at,
    property_unit_updated_at
FROM public.property_units
ON CONFLICT (unit_id) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    physical_building_id = EXCLUDED.physical_building_id,
    identity_key = EXCLUDED.identity_key,
    address_norm = EXCLUDED.address_norm,
    apartment = EXCLUDED.apartment,
    floor_level = EXCLUDED.floor_level,
    area_m2 = EXCLUDED.area_m2,
    room_layout = EXCLUDED.room_layout,
    updated_at = EXCLUDED.updated_at;
INSERT INTO public.houses (
    house_id,
    identity_key,
    address_norm,
    postal_norm,
    city_norm,
    latitude,
    longitude,
    created_at,
    updated_at
)
SELECT
    property_house_id,
    property_house_identity_key,
    property_house_address_norm,
    property_house_postal_norm,
    property_house_city_norm,
    property_house_latitude,
    property_house_longitude,
    property_house_created_at,
    property_house_updated_at
FROM public.property_houses
ON CONFLICT (house_id) DO UPDATE SET
    identity_key = EXCLUDED.identity_key,
    address_norm = EXCLUDED.address_norm,
    postal_norm = EXCLUDED.postal_norm,
    city_norm = EXCLUDED.city_norm,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    updated_at = EXCLUDED.updated_at;
DROP TRIGGER IF EXISTS trg__sync_unit_from_property_unit ON public.property_units;
CREATE TRIGGER trg__sync_unit_from_property_unit
AFTER INSERT OR UPDATE OR DELETE ON public.property_units
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_unit_from_property_unit();
DROP TRIGGER IF EXISTS trg__sync_house_from_property_house ON public.property_houses;
CREATE TRIGGER trg__sync_house_from_property_house
AFTER INSERT OR UPDATE OR DELETE ON public.property_houses
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_house_from_property_house();
