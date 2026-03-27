-- Add neighborhood column to postal codes
ALTER TABLE public.postal_postal_codes
ADD COLUMN postal_postal_code_neighborhood_fi text;

CREATE INDEX idx_postal_postal_code_neighborhood_fi ON public.postal_postal_codes(postal_postal_code_neighborhood_fi);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_postal_postal_code_neighborhood_fi;

ALTER TABLE public.postal_postal_codes
DROP COLUMN postal_postal_code_neighborhood_fi;
