ALTER TABLE public.property_buildings
ADD COLUMN IF NOT EXISTS property_building_geom postgis.geometry(Point, 4326);

CREATE INDEX IF NOT EXISTS idx_property_buildings_geom
ON public.property_buildings USING gist (property_building_geom);

CREATE OR REPLACE FUNCTION public.fnc__refresh_property_building_geom(target_property_building_id uuid DEFAULT NULL)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    updated_count integer;
BEGIN
    WITH chosen AS (
        SELECT DISTINCT ON (pb.property_building_id)
            pb.property_building_id,
            COALESCE(
                sb.shortcut_building_geom,
                fb.frontdoor_building_geom,
                CASE
                    WHEN fa.frontdoor_ad_data #>> '{property,geoCode,longitude}' IS NOT NULL
                        AND fa.frontdoor_ad_data #>> '{property,geoCode,latitude}' IS NOT NULL
                    THEN postgis.ST_SetSRID(postgis.ST_MakePoint((fa.frontdoor_ad_data #>> '{property,geoCode,longitude}')::double precision, (fa.frontdoor_ad_data #>> '{property,geoCode,latitude}')::double precision), 4326)
                    ELSE NULL
                END
            ) AS geom
        FROM public.property_buildings pb
        JOIN public.property_units pu ON pu.property_building_id = pb.property_building_id
        JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
        JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
            AND pos.property_offering_source_link_status <> 'rejected'
        JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
        LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
        LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
        LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE (target_property_building_id IS NULL OR pb.property_building_id = target_property_building_id)
            AND COALESCE(
                sb.shortcut_building_geom,
                fb.frontdoor_building_geom,
                CASE
                    WHEN fa.frontdoor_ad_data #>> '{property,geoCode,longitude}' IS NOT NULL
                        AND fa.frontdoor_ad_data #>> '{property,geoCode,latitude}' IS NOT NULL
                    THEN postgis.ST_SetSRID(postgis.ST_MakePoint((fa.frontdoor_ad_data #>> '{property,geoCode,longitude}')::double precision, (fa.frontdoor_ad_data #>> '{property,geoCode,latitude}')::double precision), 4326)
                    ELSE NULL
                END
            ) IS NOT NULL
        ORDER BY
            pb.property_building_id,
            CASE
                WHEN sb.shortcut_building_geom IS NOT NULL THEN 0
                WHEN fb.frontdoor_building_geom IS NOT NULL THEN 1
                ELSE 2
            END,
            sl.sale_listing_last_seen_at DESC NULLS LAST
    ),
    updated AS (
        UPDATE public.property_buildings pb
        SET property_building_geom = chosen.geom,
            property_building_updated_at = now()
        FROM chosen
        WHERE pb.property_building_id = chosen.property_building_id
            AND (pb.property_building_geom IS NULL OR NOT postgis.ST_Equals(pb.property_building_geom, chosen.geom))
        RETURNING pb.property_building_id
    )
    SELECT count(*)::integer INTO updated_count FROM updated;
    RETURN updated_count;
END;
$$;

SELECT public.fnc__refresh_property_building_geom(NULL);

CREATE OR REPLACE FUNCTION public.fnc__refresh_property_building_geom_for_source_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    target_building_id uuid;
BEGIN
    SELECT pu.property_building_id INTO target_building_id
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE po.property_offering_id = NEW.property_offering_id
    LIMIT 1;
    IF target_building_id IS NOT NULL THEN
        PERFORM public.fnc__refresh_property_building_geom(target_building_id);
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_refresh_property_building_geom_for_source ON public.property_offering_sources;
CREATE TRIGGER trg_refresh_property_building_geom_for_source
AFTER INSERT OR UPDATE OF property_offering_id, sale_listing_id, property_offering_source_link_status ON public.property_offering_sources
FOR EACH ROW
EXECUTE FUNCTION public.fnc__refresh_property_building_geom_for_source_trigger();
