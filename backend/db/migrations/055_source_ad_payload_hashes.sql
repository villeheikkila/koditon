ALTER TABLE public.shortcut_ads
ADD COLUMN IF NOT EXISTS shortcut_ad_data_hash text,
ADD COLUMN IF NOT EXISTS shortcut_ad_data_hash_algorithm text NOT NULL DEFAULT 'sha256',
ADD COLUMN IF NOT EXISTS shortcut_ad_data_changed_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS shortcut_ad_data_normalized_at timestamp with time zone;

ALTER TABLE public.frontdoor_ads
ADD COLUMN IF NOT EXISTS frontdoor_ad_data_hash text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_data_hash_algorithm text NOT NULL DEFAULT 'sha256',
ADD COLUMN IF NOT EXISTS frontdoor_ad_data_changed_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS frontdoor_ad_data_normalized_at timestamp with time zone;

CREATE INDEX IF NOT EXISTS idx_shortcut_ads_data_hash ON public.shortcut_ads (shortcut_ad_data_hash);
CREATE INDEX IF NOT EXISTS idx_shortcut_ads_data_normalized ON public.shortcut_ads (shortcut_ad_data_normalized_at) WHERE shortcut_ad_data_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_data_hash ON public.frontdoor_ads (frontdoor_ad_data_hash);
CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_data_normalized ON public.frontdoor_ads (frontdoor_ad_data_normalized_at) WHERE frontdoor_ad_data_hash IS NOT NULL;
