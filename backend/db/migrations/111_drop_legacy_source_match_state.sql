DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer, uuid);
DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer, text);
DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer);
ALTER TABLE public.property_source_offerings
DROP COLUMN IF EXISTS sale_listing_source_match_run_id;
ALTER TABLE public.property_offering_merge_decisions
DROP COLUMN IF EXISTS property_offering_source_match_candidate_id;
DROP TABLE IF EXISTS public.property_offering_source_match_candidates;
DROP TABLE IF EXISTS public.property_offering_source_match_runs;
