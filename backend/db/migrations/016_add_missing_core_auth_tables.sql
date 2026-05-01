DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typnamespace = 'public'::regnamespace AND typname = 'enum__name_display') THEN
        CREATE TYPE public.enum__name_display AS ENUM ('full_name', 'username');
    END IF;
END
$$;
CREATE TABLE IF NOT EXISTS public.users (
    user_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT users_uuid_key UNIQUE,
    user_first_name text,
    user_last_name text,
    user_username text CONSTRAINT users_username_key UNIQUE,
    user_name_display public.enum__name_display DEFAULT 'username'::public.enum__name_display,
    user_is_private boolean DEFAULT false NOT NULL,
    user_is_onboarded boolean DEFAULT false NOT NULL,
    user_joined_at timestamp with time zone DEFAULT now() NOT NULL,
    user_search text GENERATED ALWAYS AS (((user_username || COALESCE(user_first_name, ''::text)) || COALESCE(user_last_name, ''::text))) STORED,
    user_preferred_name text GENERATED ALWAYS AS (
        CASE
            WHEN ((user_name_display = 'full_name'::public.enum__name_display) AND (user_first_name IS NOT NULL) AND (user_last_name IS NOT NULL)) THEN ((user_first_name || ' '::text) || user_last_name)
            ELSE user_username
        END
    ) STORED,
    user_id bigint generated always as identity NOT NULL CONSTRAINT users_pkey PRIMARY KEY,
    user_email text,
    user_has_seen_passkey_onboarding boolean DEFAULT false NOT NULL,
    CONSTRAINT users_user_username_length CHECK (((user_username IS NULL) OR ((char_length(user_username) >= 2) AND (char_length(user_username) <= 16))))
);
CREATE INDEX IF NOT EXISTS idx_users_username ON public.users USING btree (user_username);
CREATE INDEX IF NOT EXISTS idx_users_created_by_private ON public.users USING btree (user_uuid) WHERE (user_is_private = true);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_user_email_normalized_unique ON public.users (lower(btrim(user_email))) WHERE user_email IS NOT NULL;
CREATE TABLE IF NOT EXISTS public.user_identities (
    user_identity_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT user_identities_uuid_key UNIQUE,
    user_identity_external_id text NOT NULL,
    user_identity_email text,
    user_identity_email_verified boolean DEFAULT false NOT NULL,
    user_identity_data jsonb DEFAULT '{}'::jsonb,
    user_identity_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_provider text NOT NULL,
    user_identity_id bigint generated always as identity NOT NULL CONSTRAINT user_identities_pkey PRIMARY KEY,
    user_id bigint NOT NULL CONSTRAINT user_identities_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT user_identity_provider_check CHECK ((user_identity_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON public.user_identities USING btree (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identities_provider_external_id_unique ON public.user_identities (user_identity_provider, user_identity_external_id);
CREATE TABLE IF NOT EXISTS public.user_devices (
    user_device_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT user_devices_uuid_key UNIQUE,
    user_device_name text,
    user_device_os text,
    user_device_app_version text,
    user_device_push_token text,
    user_device_push_token_updated_at timestamp with time zone,
    user_device_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_push_token_type text,
    user_device_id bigint generated always as identity NOT NULL CONSTRAINT user_devices_pkey PRIMARY KEY,
    user_id bigint NOT NULL CONSTRAINT user_devices_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    user_device_push_is_development boolean DEFAULT false NOT NULL,
    user_device_push_token_invalidated_at timestamp with time zone,
    user_device_push_token_invalidated_reason text,
    user_device_model text,
    user_device_locale text,
    user_device_time_zone text,
    CONSTRAINT user_device_push_token_type_check CHECK (((user_device_push_token_type IS NULL) OR (user_device_push_token_type = 'apns'::text)))
);
CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON public.user_devices USING btree (user_id);
CREATE INDEX IF NOT EXISTS idx_user_devices_push_token ON public.user_devices USING btree (user_device_push_token) WHERE (user_device_push_token IS NOT NULL);
CREATE TABLE IF NOT EXISTS public.device_sessions (
    device_session_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT device_sessions_uuid_key UNIQUE,
    device_session_user_agent text,
    device_session_ip inet,
    device_session_created_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_refreshed_at timestamp with time zone,
    device_session_not_after timestamp with time zone,
    device_session_revoked_at timestamp with time zone,
    device_session_provider text NOT NULL,
    device_session_id bigint generated always as identity NOT NULL CONSTRAINT device_sessions_pkey PRIMARY KEY,
    device_session_user_device_id bigint NOT NULL CONSTRAINT device_sessions_device_session_user_device_id_fkey REFERENCES public.user_devices(user_device_id) ON DELETE CASCADE,
    user_id bigint NOT NULL CONSTRAINT device_sessions_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    device_session_device_name text,
    device_session_device_os text,
    device_session_device_model text,
    device_session_app_version text,
    device_session_locale text,
    device_session_time_zone text,
    device_session_location_city text,
    device_session_location_region text,
    device_session_location_country_code text,
    device_session_location_source text,
    CONSTRAINT device_session_provider_check CHECK ((device_session_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);
CREATE INDEX IF NOT EXISTS idx_device_sessions_not_after ON public.device_sessions USING btree (device_session_not_after DESC);
CREATE INDEX IF NOT EXISTS idx_device_sessions_user_created ON public.device_sessions USING btree (user_id, device_session_created_at);
CREATE INDEX IF NOT EXISTS idx_device_sessions_user_device_id ON public.device_sessions USING btree (device_session_user_device_id);
CREATE INDEX IF NOT EXISTS idx_device_sessions_user_id ON public.device_sessions USING btree (user_id);
CREATE TABLE IF NOT EXISTS public.user_passkeys (
    user_passkey_id bigint generated always as identity primary key,
    user_passkey_uuid uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id bigint NOT NULL REFERENCES public.users(user_id) ON DELETE CASCADE,
    user_identity_id bigint NOT NULL REFERENCES public.user_identities(user_identity_id) ON DELETE CASCADE,
    user_passkey_credential_id bytea NOT NULL,
    user_passkey_credential_id_b64url text NOT NULL,
    user_passkey_public_key bytea NOT NULL,
    user_passkey_attestation_type text NOT NULL,
    user_passkey_transports text[] NOT NULL DEFAULT '{}',
    user_passkey_user_handle bytea NOT NULL,
    user_passkey_sign_count bigint NOT NULL DEFAULT 0,
    user_passkey_flags integer,
    user_passkey_aaguid uuid,
    user_passkey_name text,
    user_passkey_backup_eligible boolean,
    user_passkey_backup_state boolean,
    user_passkey_last_used_at timestamptz,
    user_passkey_created_at timestamptz NOT NULL DEFAULT now(),
    user_passkey_updated_at timestamptz NOT NULL DEFAULT now(),
    user_passkey_revoked_at timestamptz,
    CONSTRAINT user_passkeys_credential_id_key UNIQUE (user_passkey_credential_id),
    CONSTRAINT user_passkeys_credential_id_b64url_key UNIQUE (user_passkey_credential_id_b64url)
);
CREATE INDEX IF NOT EXISTS idx_user_passkeys_user_id ON public.user_passkeys (user_id);
CREATE INDEX IF NOT EXISTS idx_user_passkeys_user_handle ON public.user_passkeys (user_passkey_user_handle);
CREATE INDEX IF NOT EXISTS idx_user_passkeys_active_user_id ON public.user_passkeys (user_id) WHERE user_passkey_revoked_at IS NULL;
CREATE TABLE IF NOT EXISTS public.auth_webauthn_challenges (
    auth_webauthn_challenge_id bigint generated always as identity primary key,
    auth_webauthn_challenge_uuid uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    auth_webauthn_challenge_flow text NOT NULL,
    auth_webauthn_challenge_session jsonb NOT NULL,
    auth_webauthn_challenge_expires_at timestamptz NOT NULL,
    auth_webauthn_challenge_user_handle bytea,
    auth_webauthn_challenge_user_display_name text,
    auth_webauthn_challenge_device_id uuid,
    auth_webauthn_challenge_consumed_at timestamptz,
    auth_webauthn_challenge_created_at timestamptz NOT NULL DEFAULT now(),
    auth_webauthn_challenge_verified_email text,
    user_id bigint REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT auth_webauthn_challenge_flow_check CHECK (auth_webauthn_challenge_flow = ANY (ARRAY['authenticate'::text, 'register'::text, 'signup'::text]))
);
CREATE INDEX IF NOT EXISTS idx_auth_webauthn_challenges_uuid ON public.auth_webauthn_challenges (auth_webauthn_challenge_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_webauthn_challenges_active ON public.auth_webauthn_challenges (auth_webauthn_challenge_uuid, auth_webauthn_challenge_flow) WHERE auth_webauthn_challenge_consumed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_webauthn_challenges_expires ON public.auth_webauthn_challenges (auth_webauthn_challenge_expires_at);
CREATE TABLE IF NOT EXISTS public.oauth_authorization_codes (
    oauth_authorization_code_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_authorization_code_code_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL REFERENCES public.users(user_uuid) ON DELETE CASCADE,
    oauth_authorization_code_redirect_uri text NOT NULL,
    oauth_authorization_code_scopes text[] NOT NULL DEFAULT '{}',
    oauth_authorization_code_code_challenge text NOT NULL,
    oauth_authorization_code_code_challenge_method text NOT NULL,
    oauth_authorization_code_audience text NOT NULL DEFAULT '',
    oauth_authorization_code_expires_at timestamptz NOT NULL,
    oauth_authorization_code_consumed_at timestamptz,
    oauth_authorization_code_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_authorization_code_updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_client_id ON public.oauth_authorization_codes (oauth_client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_user_uuid ON public.oauth_authorization_codes (user_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_expires_at ON public.oauth_authorization_codes (oauth_authorization_code_expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_audience ON public.oauth_authorization_codes (oauth_authorization_code_audience);
CREATE TABLE IF NOT EXISTS public.oauth_refresh_tokens (
    oauth_refresh_token_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_refresh_token_token_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL REFERENCES public.users(user_uuid) ON DELETE CASCADE,
    device_session_uuid uuid REFERENCES public.device_sessions(device_session_uuid) ON DELETE SET NULL,
    oauth_refresh_token_scopes text[] NOT NULL DEFAULT '{}',
    oauth_refresh_token_audience text NOT NULL DEFAULT '',
    oauth_refresh_token_expires_at timestamptz NOT NULL,
    oauth_refresh_token_revoked_at timestamptz,
    oauth_refresh_token_rotated_from uuid REFERENCES public.oauth_refresh_tokens(oauth_refresh_token_id) ON DELETE SET NULL,
    oauth_refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_refresh_token_updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_client_id ON public.oauth_refresh_tokens (oauth_client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_user_uuid ON public.oauth_refresh_tokens (user_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_device_session_uuid ON public.oauth_refresh_tokens (device_session_uuid);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires_at ON public.oauth_refresh_tokens (oauth_refresh_token_expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_audience ON public.oauth_refresh_tokens (oauth_refresh_token_audience);
