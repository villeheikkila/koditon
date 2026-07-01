DROP VIEW IF EXISTS public.listings;
CREATE TABLE public.listings (
    listing_id uuid PRIMARY KEY,
    listing_type text NOT NULL,
    listing_status text,
    primary_source_listing_id uuid,
    unit_id uuid,
    house_id uuid,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (listing_type = ANY (ARRAY['sale']))
);
CREATE INDEX idx_listings_primary_source_listing
ON public.listings (primary_source_listing_id);
CREATE INDEX idx_listings_unit
ON public.listings (unit_id) WHERE unit_id IS NOT NULL;
CREATE INDEX idx_listings_house
ON public.listings (house_id) WHERE house_id IS NOT NULL;
CREATE INDEX idx_listings_last_seen
ON public.listings (last_seen_at DESC);
CREATE OR REPLACE FUNCTION public.fnc__sync_listing_from_property_offering()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.listings WHERE listing_id = OLD.property_offering_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.listings (
        listing_id,
        listing_type,
        listing_status,
        primary_source_listing_id,
        unit_id,
        house_id,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    VALUES (
        NEW.property_offering_id,
        NEW.property_offering_type,
        NEW.property_offering_status,
        NEW.primary_sale_listing_id,
        NEW.property_unit_id,
        NEW.property_house_id,
        NEW.property_offering_first_seen_at,
        NEW.property_offering_last_seen_at,
        NEW.property_offering_created_at,
        NEW.property_offering_updated_at
    )
    ON CONFLICT (listing_id) DO UPDATE SET
        listing_type = EXCLUDED.listing_type,
        listing_status = EXCLUDED.listing_status,
        primary_source_listing_id = EXCLUDED.primary_source_listing_id,
        unit_id = EXCLUDED.unit_id,
        house_id = EXCLUDED.house_id,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$;
INSERT INTO public.listings (
    listing_id,
    listing_type,
    listing_status,
    primary_source_listing_id,
    unit_id,
    house_id,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    property_offering_id,
    property_offering_type,
    property_offering_status,
    primary_sale_listing_id,
    property_unit_id,
    property_house_id,
    property_offering_first_seen_at,
    property_offering_last_seen_at,
    property_offering_created_at,
    property_offering_updated_at
FROM public.property_offerings
ON CONFLICT (listing_id) DO UPDATE SET
    listing_type = EXCLUDED.listing_type,
    listing_status = EXCLUDED.listing_status,
    primary_source_listing_id = EXCLUDED.primary_source_listing_id,
    unit_id = EXCLUDED.unit_id,
    house_id = EXCLUDED.house_id,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;
DROP TRIGGER IF EXISTS trg__sync_listing_from_property_offering ON public.property_offerings;
CREATE TRIGGER trg__sync_listing_from_property_offering
AFTER INSERT OR UPDATE OR DELETE ON public.property_offerings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_listing_from_property_offering();
