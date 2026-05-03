ALTER TABLE public.frontdoor_ads
DROP COLUMN IF EXISTS frontdoor_ad_address,
DROP COLUMN IF EXISTS frontdoor_ad_area,
DROP COLUMN IF EXISTS frontdoor_ad_room_layout,
DROP COLUMN IF EXISTS frontdoor_ad_asking_price;

ALTER TABLE public.shortcut_ads
DROP COLUMN IF EXISTS shortcut_ad_address,
DROP COLUMN IF EXISTS shortcut_ad_area,
DROP COLUMN IF EXISTS shortcut_ad_room_layout,
DROP COLUMN IF EXISTS shortcut_ad_asking_price;
