ALTER TABLE public.shortcut_ads
    ADD COLUMN IF NOT EXISTS shortcut_ad_data_normalized_version integer default 0 not null;

ALTER TABLE public.frontdoor_ads
    ADD COLUMN IF NOT EXISTS frontdoor_ad_data_normalized_version integer default 0 not null;

CREATE INDEX IF NOT EXISTS idx_shortcut_ads_data_normalized_version
    ON public.shortcut_ads (shortcut_ad_data_normalized_version)
    WHERE shortcut_ad_data_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_data_normalized_version
    ON public.frontdoor_ads (frontdoor_ad_data_normalized_version)
    WHERE frontdoor_ad_data_hash IS NOT NULL;
