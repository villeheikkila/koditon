CREATE OR REPLACE FUNCTION public.fnc__condition_match_code(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE public.fnc__match_alias_key(value)
        WHEN 'good' THEN 'good'
        WHEN 'hyvä' THEN 'good'
        WHEN 'hyva' THEN 'good'
        WHEN 'satisfactory' THEN 'satisfactory'
        WHEN 'tyyd' THEN 'satisfactory'
        WHEN 'tyydyttävä' THEN 'satisfactory'
        WHEN 'tyydyttava' THEN 'satisfactory'
        WHEN 'tolerable' THEN 'poor'
        WHEN 'poor' THEN 'poor'
        WHEN 'bad' THEN 'poor'
        WHEN 'huono' THEN 'poor'
        WHEN 'välttävä' THEN 'poor'
        WHEN 'valttava' THEN 'poor'
        WHEN 'unclassified' THEN 'unknown'
        WHEN 'not_known' THEN 'unknown'
        WHEN 'not_shown' THEN 'unknown'
        ELSE NULL
    END
$$;
