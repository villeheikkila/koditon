ALTER TABLE public.shortcut_ads
ADD COLUMN IF NOT EXISTS shortcut_ad_data_schema_version int2 NOT NULL DEFAULT 1;
