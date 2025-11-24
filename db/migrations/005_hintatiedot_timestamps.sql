DROP FUNCTION IF EXISTS public.t__set_timestamps();
CREATE OR REPLACE FUNCTION public.tg__set_timestamps()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    IF NEW.created_at IS NULL THEN
        NEW.created_at = now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE public.hintatiedot_cities
    ALTER COLUMN hintatiedot_cities_created_at SET DEFAULT now(),
    ALTER COLUMN hintatiedot_cities_updated_at SET DEFAULT now();

ALTER TABLE public.hintatiedot_postal_codes
    ALTER COLUMN hintatiedot_postal_codes_created_at SET DEFAULT now(),
    ALTER COLUMN hintatiedot_postal_codes_updated_at SET DEFAULT now();

ALTER TABLE public.hintatiedot_neighborhoods
    ALTER COLUMN hintatiedot_neighborhoods_created_at SET DEFAULT now(),
    ALTER COLUMN hintatiedot_neighborhoods_updated_at SET DEFAULT now();

ALTER TABLE public.hintatiedot_transactions
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET DEFAULT now();

CREATE TRIGGER tg__00__hintatiedot_cities__timestamps
    BEFORE INSERT OR UPDATE ON public.hintatiedot_cities
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__hintatiedot_postal_codes__timestamps
    BEFORE INSERT OR UPDATE ON public.hintatiedot_postal_codes
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__hintatiedot_neighborhoods__timestamps
    BEFORE INSERT OR UPDATE ON public.hintatiedot_neighborhoods
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__hintatiedot_transactions__timestamps
    BEFORE INSERT OR UPDATE ON public.hintatiedot_transactions
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();
