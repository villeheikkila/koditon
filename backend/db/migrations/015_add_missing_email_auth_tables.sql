CREATE TABLE IF NOT EXISTS public.auth_signup_email_tokens (
    auth_signup_email_token_id bigint generated always as identity NOT NULL CONSTRAINT auth_signup_email_tokens_pkey PRIMARY KEY,
    auth_signup_email_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT auth_signup_email_tokens_uuid_key UNIQUE,
    auth_signup_email_target_email text NOT NULL,
    auth_signup_email_token_hash text NOT NULL CONSTRAINT auth_signup_email_tokens_token_hash_key UNIQUE,
    auth_signup_email_expires_at timestamp with time zone NOT NULL,
    auth_signup_email_consumed_at timestamp with time zone,
    auth_signup_email_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_email_tokens_target_email_not_blank CHECK (btrim(auth_signup_email_target_email) <> '')
);
CREATE INDEX IF NOT EXISTS idx_auth_signup_email_tokens_active_expires_at
    ON public.auth_signup_email_tokens USING btree (auth_signup_email_expires_at)
    WHERE auth_signup_email_consumed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_signup_email_tokens_target_email
    ON public.auth_signup_email_tokens USING btree (lower(btrim(auth_signup_email_target_email)));
CREATE TABLE IF NOT EXISTS public.auth_signup_tickets (
    auth_signup_ticket_id bigint generated always as identity NOT NULL CONSTRAINT auth_signup_tickets_pkey PRIMARY KEY,
    auth_signup_ticket_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT auth_signup_tickets_uuid_key UNIQUE,
    auth_signup_ticket_target_email text NOT NULL,
    auth_signup_ticket_hash text NOT NULL CONSTRAINT auth_signup_tickets_hash_key UNIQUE,
    auth_signup_ticket_expires_at timestamp with time zone NOT NULL,
    auth_signup_ticket_consumed_at timestamp with time zone,
    auth_signup_ticket_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_tickets_target_email_not_blank CHECK (btrim(auth_signup_ticket_target_email) <> '')
);
CREATE INDEX IF NOT EXISTS idx_auth_signup_tickets_active_expires_at
    ON public.auth_signup_tickets USING btree (auth_signup_ticket_expires_at)
    WHERE auth_signup_ticket_consumed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_signup_tickets_target_email
    ON public.auth_signup_tickets USING btree (lower(btrim(auth_signup_ticket_target_email)));
