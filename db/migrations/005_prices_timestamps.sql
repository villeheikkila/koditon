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

ALTER TABLE public.prices_cities
    ALTER COLUMN prices_cities_created_at SET DEFAULT now(),
    ALTER COLUMN prices_cities_updated_at SET DEFAULT now();

ALTER TABLE public.prices_postal_codes
    ALTER COLUMN prices_postal_codes_created_at SET DEFAULT now(),
    ALTER COLUMN prices_postal_codes_updated_at SET DEFAULT now();

ALTER TABLE public.prices_neighborhoods
    ALTER COLUMN prices_neighborhoods_created_at SET DEFAULT now(),
    ALTER COLUMN prices_neighborhoods_updated_at SET DEFAULT now();

ALTER TABLE public.prices_transactions
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET DEFAULT now();

CREATE TRIGGER tg__00__prices_cities__timestamps
    BEFORE INSERT OR UPDATE ON public.prices_cities
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__prices_postal_codes__timestamps
    BEFORE INSERT OR UPDATE ON public.prices_postal_codes
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__prices_neighborhoods__timestamps
    BEFORE INSERT OR UPDATE ON public.prices_neighborhoods
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();

CREATE TRIGGER tg__00__prices_transactions__timestamps
    BEFORE INSERT OR UPDATE ON public.prices_transactions
    FOR EACH ROW EXECUTE FUNCTION public.tg__set_timestamps();
