ALTER TABLE public.apartment_profiles
    ADD COLUMN IF NOT EXISTS apartment_profile_maintenance_charge_monthly double precision,
    ADD COLUMN IF NOT EXISTS apartment_profile_capital_charge_monthly double precision,
    ADD COLUMN IF NOT EXISTS apartment_profile_total_charge_monthly double precision,
    ADD COLUMN IF NOT EXISTS apartment_profile_debt_share_eur bigint,
    ADD COLUMN IF NOT EXISTS apartment_profile_shareholder_liability text;
