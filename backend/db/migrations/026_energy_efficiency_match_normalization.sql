CREATE TABLE IF NOT EXISTS public.energy_efficiency_aliases (
    energy_efficiency_alias text PRIMARY KEY,
    energy_efficiency_class_code text,
    energy_efficiency_standard_year integer,
    energy_efficiency_status text NOT NULL,
    energy_efficiency_match_code text,
    energy_efficiency_label text NOT NULL,
    CONSTRAINT energy_efficiency_aliases_status_check CHECK (energy_efficiency_status = ANY (ARRAY['known'::text, 'not_required'::text, 'not_available'::text, 'unknown'::text]))
);
CREATE OR REPLACE FUNCTION public.fnc__match_alias_key(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '_', 'g')), '')
$$;
INSERT INTO public.energy_efficiency_aliases (
    energy_efficiency_alias,
    energy_efficiency_class_code,
    energy_efficiency_standard_year,
    energy_efficiency_status,
    energy_efficiency_match_code,
    energy_efficiency_label
)
VALUES
    ('not_available', NULL, NULL, 'not_available', NULL, 'Not available'),
    ('no_certificate', NULL, NULL, 'not_available', NULL, 'No certificate'),
    ('ei_energiatodistusta', NULL, NULL, 'not_available', NULL, 'Ei energiatodistusta'),
    ('energiatodistus_on', NULL, NULL, 'known', NULL, 'Energiatodistus on'),
    ('ei_lain_edellyttämää_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta'),
    ('ei_lain_edellytta_ma_a_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta'),
    ('ei_lain_vaatimaa_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain vaatimaa energiatodistusta'),
    ('not_required', NULL, NULL, 'not_required', NULL, 'Not required'),
    ('laki_ei_edellytä_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Laki ei edellytä energiatodistusta'),
    ('kohteelle_ei_lain_mukaan_tarvita_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Kohteelle ei lain mukaan tarvita energiatodistusta'),
    ('energiatodistusta_ei_vaadita_kohteelle_ei_lain_mukaan_tarvitse_hankkia_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Energiatodistusta ei vaadita'),
    ('kohteella_ei_energiatodistuslain_nojalla_tarvitse_olla_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Kohteella ei tarvitse olla energiatodistusta'),
    ('kohteella_ei_ole_lain_edellyttämää_energiatodistusta_ja_sen_vuoksi_energialuokka_ei_ole_tiedossa', NULL, NULL, 'not_required', NULL, 'Kohteella ei ole lain edellyttämää energiatodistusta'),
    ('ei_energiatodistusta_kohteella_ei_ole_lain_edellyttämää_energiatodistusta_ja_sen_vuoksi_energialuokka_ei_ole_tiedossa', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta')
ON CONFLICT (energy_efficiency_alias) DO UPDATE SET
    energy_efficiency_class_code = EXCLUDED.energy_efficiency_class_code,
    energy_efficiency_standard_year = EXCLUDED.energy_efficiency_standard_year,
    energy_efficiency_status = EXCLUDED.energy_efficiency_status,
    energy_efficiency_match_code = EXCLUDED.energy_efficiency_match_code,
    energy_efficiency_label = EXCLUDED.energy_efficiency_label;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_normalized(value text)
RETURNS TABLE (
    energy_efficiency_class_code text,
    energy_efficiency_standard_year integer,
    energy_efficiency_status text,
    energy_efficiency_match_code text
)
LANGUAGE sql
STABLE
AS $$
    WITH normalized_input AS (
        SELECT public.fnc__match_alias_key(value) AS alias_key
    ),
    mapped AS (
        SELECT
            a.energy_efficiency_class_code,
            a.energy_efficiency_standard_year,
            a.energy_efficiency_status,
            a.energy_efficiency_match_code
        FROM public.energy_efficiency_aliases a
        JOIN normalized_input i ON i.alias_key = a.energy_efficiency_alias
    ),
    provider_code AS (
        SELECT regexp_match(alias_key, '^e([0-9]{2})_([a-h])$') AS parts
        FROM normalized_input
    ),
    label_code AS (
        SELECT regexp_match(alias_key, '(^|_)([a-h])_?((?:19|20|21)[0-9]{2})($|_)') AS parts
        FROM normalized_input
    ),
    energy_label_year AS (
        SELECT regexp_match(alias_key, '(^|_)([a-h])_energialuokkaan_((?:19|20|21)[0-9]{2})($|_)') AS parts
        FROM normalized_input
    ),
    energy_label_class AS (
        SELECT regexp_match(alias_key, '(^|_)energialuokka_([a-h])($|_)') AS parts
        FROM normalized_input
    ),
    leading_certificate_class AS (
        SELECT regexp_match(alias_key, '^([a-h])_energiatodistus') AS parts
        FROM normalized_input
    ),
    class_only AS (
        SELECT regexp_match(alias_key, '^([a-h])$') AS parts
        FROM normalized_input
    ),
    derived AS (
        SELECT
            CASE
                WHEN provider_code.parts IS NOT NULL THEN upper(provider_code.parts[2])
                WHEN label_code.parts IS NOT NULL THEN upper(label_code.parts[2])
                WHEN energy_label_year.parts IS NOT NULL THEN upper(energy_label_year.parts[2])
                WHEN energy_label_class.parts IS NOT NULL THEN upper(energy_label_class.parts[2])
                WHEN leading_certificate_class.parts IS NOT NULL THEN upper(leading_certificate_class.parts[1])
                WHEN class_only.parts IS NOT NULL THEN upper(class_only.parts[1])
                ELSE NULL
            END AS class_code,
            CASE
                WHEN provider_code.parts IS NULL THEN NULL
                WHEN provider_code.parts[1]::integer < 50 THEN 2000 + provider_code.parts[1]::integer
                ELSE 1900 + provider_code.parts[1]::integer
            END AS provider_year,
            CASE WHEN label_code.parts IS NOT NULL THEN label_code.parts[3]::integer ELSE NULL END AS label_year,
            CASE WHEN energy_label_year.parts IS NOT NULL THEN energy_label_year.parts[3]::integer ELSE NULL END AS energy_label_year,
            (provider_code.parts IS NOT NULL OR label_code.parts IS NOT NULL OR energy_label_year.parts IS NOT NULL OR energy_label_class.parts IS NOT NULL OR leading_certificate_class.parts IS NOT NULL OR class_only.parts IS NOT NULL) AS is_known
        FROM provider_code, label_code, energy_label_year, energy_label_class, leading_certificate_class, class_only
    )
    SELECT
        COALESCE(mapped.energy_efficiency_class_code, derived.class_code),
        COALESCE(mapped.energy_efficiency_standard_year, derived.provider_year, derived.label_year, derived.energy_label_year),
        COALESCE(mapped.energy_efficiency_status, CASE WHEN normalized_input.alias_key IS NULL THEN 'unknown' WHEN derived.is_known THEN 'known' ELSE 'unknown' END),
        COALESCE(mapped.energy_efficiency_match_code, CASE WHEN derived.class_code IS NULL THEN NULL WHEN COALESCE(derived.provider_year, derived.label_year, derived.energy_label_year) IS NULL THEN derived.class_code ELSE derived.class_code || COALESCE(derived.provider_year, derived.label_year, derived.energy_label_year)::text END)
    FROM normalized_input
    LEFT JOIN mapped ON true
    JOIN derived ON true
$$;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_match_label(VARIADIC labels text[])
RETURNS text
LANGUAGE sql
STABLE
AS $$
    WITH candidates AS (
        SELECT ordinality, NULLIF(trim(value), '') AS label
        FROM unnest(labels) WITH ORDINALITY AS value(value, ordinality)
        WHERE NULLIF(trim(value), '') IS NOT NULL
    ),
    scored AS (
        SELECT c.ordinality, c.label, n.energy_efficiency_match_code, n.energy_efficiency_status
        FROM candidates c
        CROSS JOIN LATERAL public.fnc__energy_efficiency_normalized(c.label) n
    )
    SELECT label
    FROM scored
    ORDER BY
        CASE
            WHEN energy_efficiency_match_code IS NOT NULL THEN 0
            WHEN energy_efficiency_status IN ('not_required', 'not_available') THEN 1
            ELSE 2
        END,
        ordinality
    LIMIT 1
$$;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_class_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT energy_efficiency_class_code FROM public.fnc__energy_efficiency_normalized(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_standard_year(value text)
RETURNS integer
LANGUAGE sql
STABLE
AS $$
    SELECT energy_efficiency_standard_year FROM public.fnc__energy_efficiency_normalized(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_status(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT energy_efficiency_status FROM public.fnc__energy_efficiency_normalized(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__energy_efficiency_match_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT energy_efficiency_match_code FROM public.fnc__energy_efficiency_normalized(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_energy_match_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT public.fnc__energy_efficiency_match_code(value)
$$;
ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_energy_efficiency_class_code text,
ADD COLUMN IF NOT EXISTS sale_listing_energy_efficiency_standard_year integer,
ADD COLUMN IF NOT EXISTS sale_listing_energy_efficiency_status text,
ADD COLUMN IF NOT EXISTS sale_listing_energy_efficiency_match_code text;
CREATE INDEX IF NOT EXISTS idx_sale_listings_energy_efficiency_match_code ON public.sale_listings (sale_listing_energy_efficiency_match_code);
CREATE INDEX IF NOT EXISTS idx_sale_listings_energy_efficiency_class_year ON public.sale_listings (sale_listing_energy_efficiency_class_code, sale_listing_energy_efficiency_standard_year);
CREATE INDEX IF NOT EXISTS idx_sale_listings_energy_efficiency_status ON public.sale_listings (sale_listing_energy_efficiency_status);
CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_transaction_match_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    property_raw text;
    plot_raw text;
    elevator_value boolean;
    energy_label text;
    energy_match_label text;
    energy_normalized record;
BEGIN
    IF NEW.shortcut_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''),
            COALESCE(sa.shortcut_ad_elevator, public.fnc__try_parse_bool(sb.shortcut_building_has_elevator)),
            public.fnc__energy_efficiency_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class),
            public.fnc__energy_efficiency_match_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.shortcut_ads sa
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        WHERE sa.shortcut_ad_id = NEW.shortcut_ad_id;
    ELSIF NEW.frontdoor_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
            fa.frontdoor_ad_elevator,
            public.fnc__energy_efficiency_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class),
            public.fnc__energy_efficiency_match_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_ads fa
        WHERE fa.frontdoor_ad_id = NEW.frontdoor_ad_id;
    ELSIF NEW.frontdoor_building_announcement_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
            NULL::text,
            fb.frontdoor_building_has_elevator,
            public.fnc__energy_efficiency_label(fb.frontdoor_building_energy_certificate_code),
            public.fnc__energy_efficiency_match_label(fb.frontdoor_building_energy_certificate_code)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_building_announcements fba
        JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE fba.frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
    END IF;
    SELECT * INTO energy_normalized FROM public.fnc__energy_efficiency_normalized(energy_match_label);
    NEW.sale_listing_property_type_raw := property_raw;
    NEW.sale_listing_property_type_code := public.fnc__sale_listing_property_type_code(property_raw);
    NEW.sale_listing_room_category_code := public.fnc__sale_listing_room_category_code(NEW.sale_listing_rooms_count, NEW.sale_listing_room_layout);
    NEW.sale_listing_floor_text := public.fnc__sale_listing_floor_text(NEW.sale_listing_floor_level, NEW.sale_listing_total_floors);
    NEW.sale_listing_elevator := elevator_value;
    NEW.sale_listing_plot_type_raw := plot_raw;
    NEW.sale_listing_plot_type_code := public.fnc__sale_listing_plot_type_code(plot_raw);
    NEW.sale_listing_energy_efficiency_label := energy_label;
    NEW.sale_listing_energy_efficiency_class_code := energy_normalized.energy_efficiency_class_code;
    NEW.sale_listing_energy_efficiency_standard_year := energy_normalized.energy_efficiency_standard_year;
    NEW.sale_listing_energy_efficiency_status := energy_normalized.energy_efficiency_status;
    NEW.sale_listing_energy_efficiency_match_code := energy_normalized.energy_efficiency_match_code;
    RETURN NEW;
END;
$$;
UPDATE public.sale_listings sl
SET
    sale_listing_energy_efficiency_class_code = n.energy_efficiency_class_code,
    sale_listing_energy_efficiency_standard_year = n.energy_efficiency_standard_year,
    sale_listing_energy_efficiency_status = n.energy_efficiency_status,
    sale_listing_energy_efficiency_match_code = n.energy_efficiency_match_code
FROM public.shortcut_ads sa
CROSS JOIN LATERAL public.fnc__energy_efficiency_match_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class) AS source_label(value)
CROSS JOIN LATERAL public.fnc__energy_efficiency_normalized(source_label.value) n
WHERE sl.shortcut_ad_id = sa.shortcut_ad_id;
UPDATE public.sale_listings sl
SET
    sale_listing_energy_efficiency_class_code = n.energy_efficiency_class_code,
    sale_listing_energy_efficiency_standard_year = n.energy_efficiency_standard_year,
    sale_listing_energy_efficiency_status = n.energy_efficiency_status,
    sale_listing_energy_efficiency_match_code = n.energy_efficiency_match_code
FROM public.frontdoor_ads fa
CROSS JOIN LATERAL public.fnc__energy_efficiency_match_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class) AS source_label(value)
CROSS JOIN LATERAL public.fnc__energy_efficiency_normalized(source_label.value) n
WHERE sl.frontdoor_ad_id = fa.frontdoor_ad_id;
UPDATE public.sale_listings sl
SET
    sale_listing_energy_efficiency_class_code = n.energy_efficiency_class_code,
    sale_listing_energy_efficiency_standard_year = n.energy_efficiency_standard_year,
    sale_listing_energy_efficiency_status = n.energy_efficiency_status,
    sale_listing_energy_efficiency_match_code = n.energy_efficiency_match_code
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
CROSS JOIN LATERAL public.fnc__energy_efficiency_match_label(fb.frontdoor_building_energy_certificate_code) AS source_label(value)
CROSS JOIN LATERAL public.fnc__energy_efficiency_normalized(source_label.value) n
WHERE sl.frontdoor_building_announcement_id = fba.frontdoor_building_announcement_id;
