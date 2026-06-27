CREATE INDEX IF NOT EXISTS idx_property_source_offerings_street_name_number_ascii
ON public.property_source_offerings (
    (translate(sale_listing_street_name_norm, 'åäö', 'aao')),
    sale_listing_street_number_norm,
    sale_listing_last_seen_at DESC
);
