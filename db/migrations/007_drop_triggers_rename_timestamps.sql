DROP TRIGGER IF EXISTS tg__00__prices_cities__timestamps ON public.prices_cities;
DROP TRIGGER IF EXISTS tg__00__prices_postal_codes__timestamps ON public.prices_postal_codes;
DROP TRIGGER IF EXISTS tg__00__prices_neighborhoods__timestamps ON public.prices_neighborhoods;
DROP TRIGGER IF EXISTS tg__00__prices_transactions__timestamps ON public.prices_transactions;

DROP FUNCTION IF EXISTS public.tg__set_timestamps();

ALTER TABLE public.prices_transactions
    RENAME COLUMN created_at TO prices_transactions_created_at;

ALTER TABLE public.prices_transactions
    RENAME COLUMN updated_at TO prices_transactions_updated_at;

ALTER TABLE public.prices_transactions
    ALTER COLUMN prices_transactions_created_at SET DEFAULT now(),
    ALTER COLUMN prices_transactions_updated_at SET DEFAULT now();
