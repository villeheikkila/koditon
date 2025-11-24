DROP TRIGGER IF EXISTS tg__00__hintatiedot_cities__timestamps ON public.hintatiedot_cities;
DROP TRIGGER IF EXISTS tg__00__hintatiedot_postal_codes__timestamps ON public.hintatiedot_postal_codes;
DROP TRIGGER IF EXISTS tg__00__hintatiedot_neighborhoods__timestamps ON public.hintatiedot_neighborhoods;
DROP TRIGGER IF EXISTS tg__00__hintatiedot_transactions__timestamps ON public.hintatiedot_transactions;

DROP FUNCTION IF EXISTS public.tg__set_timestamps();

ALTER TABLE public.hintatiedot_transactions
    RENAME COLUMN created_at TO hintatiedot_transactions_created_at;

ALTER TABLE public.hintatiedot_transactions
    RENAME COLUMN updated_at TO hintatiedot_transactions_updated_at;

ALTER TABLE public.hintatiedot_transactions
    ALTER COLUMN hintatiedot_transactions_created_at SET DEFAULT now(),
    ALTER COLUMN hintatiedot_transactions_updated_at SET DEFAULT now();
