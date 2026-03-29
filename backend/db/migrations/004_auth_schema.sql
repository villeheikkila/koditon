-- ============================================================
-- Auth Schema (squashed to current state)
-- Includes: users, identities, sessions, devices, roles/flags,
--           passkeys, oauth, email auth, runtime idempotency
-- ============================================================

-- Types
CREATE TYPE public.enum__name_display AS ENUM (
    'full_name',
    'username'
);

-- Users
CREATE TABLE public.users (
    user_uuid uuid CONSTRAINT profiles_profile_id_not_null NOT NULL,
    user_first_name text,
    user_last_name text,
    user_username text,
    user_name_display public.enum__name_display DEFAULT 'username'::public.enum__name_display,
    user_is_private boolean DEFAULT false CONSTRAINT profiles_profile_is_private_not_null NOT NULL,
    user_is_onboarded boolean DEFAULT false CONSTRAINT profiles_profile_is_onboarded_not_null NOT NULL,
    user_joined_at timestamp with time zone DEFAULT now() CONSTRAINT profiles_profile_joined_at_not_null NOT NULL,
    user_search text GENERATED ALWAYS AS (((user_username || COALESCE(user_first_name, ''::text)) || COALESCE(user_last_name, ''::text))) STORED,
    user_preferred_name text GENERATED ALWAYS AS (
CASE
    WHEN ((user_name_display = 'full_name'::public.enum__name_display) AND (user_first_name IS NOT NULL) AND (user_last_name IS NOT NULL)) THEN ((user_first_name || ' '::text) || user_last_name)
    ELSE user_username
END) STORED,
    user_id bigint CONSTRAINT profiles_profile_id_bigint_not_null NOT NULL,
    user_email text,
    user_has_seen_passkey_onboarding boolean DEFAULT false NOT NULL,
    CONSTRAINT users_user_username_length CHECK (((user_username IS NULL) OR ((char_length(user_username) >= 2) AND (char_length(user_username) <= 16))))
);

ALTER TABLE public.users ALTER COLUMN user_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.profiles_profile_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (user_username);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_uuid_key UNIQUE (user_uuid);

CREATE INDEX idx_users_username ON public.users USING btree (user_username);
CREATE INDEX idx_users_created_by_private ON public.users USING btree (user_uuid) WHERE (user_is_private = true);
CREATE UNIQUE INDEX idx_users_user_email_normalized_unique ON public.users (lower(btrim(user_email))) WHERE user_email IS NOT NULL;

-- User Identities
CREATE TABLE public.user_identities (
    user_identity_uuid uuid DEFAULT gen_random_uuid() CONSTRAINT user_identities_user_identity_id_not_null NOT NULL,
    user_identity_external_id text NOT NULL,
    user_identity_email text,
    user_identity_email_verified boolean DEFAULT false NOT NULL,
    user_identity_data jsonb DEFAULT '{}'::jsonb,
    user_identity_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_provider text CONSTRAINT user_identities_user_identity_provider_new_not_null NOT NULL,
    user_identity_id bigint CONSTRAINT user_identities_user_identity_id_bigint_not_null NOT NULL,
    user_id bigint CONSTRAINT user_identities_user_id_bigint_not_null NOT NULL,
    CONSTRAINT user_identity_provider_check CHECK ((user_identity_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);

ALTER TABLE public.user_identities ALTER COLUMN user_identity_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_identities_user_identity_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_pkey PRIMARY KEY (user_identity_id);

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_uuid_key UNIQUE (user_identity_uuid);

CREATE INDEX idx_user_identities_user_id ON public.user_identities USING btree (user_id);
CREATE UNIQUE INDEX idx_user_identities_provider_external_id_unique ON public.user_identities (user_identity_provider, user_identity_external_id);

-- User Devices
CREATE TABLE public.user_devices (
    user_device_uuid uuid DEFAULT gen_random_uuid() CONSTRAINT user_devices_user_device_id_not_null NOT NULL,
    user_device_name text,
    user_device_os text,
    user_device_app_version text,
    user_device_push_token text,
    user_device_push_token_updated_at timestamp with time zone,
    user_device_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_push_token_type text,
    user_device_id bigint CONSTRAINT user_devices_user_device_id_bigint_not_null NOT NULL,
    user_id bigint CONSTRAINT user_devices_user_id_bigint_not_null NOT NULL,
    user_device_push_is_development boolean DEFAULT false NOT NULL,
    user_device_push_token_invalidated_at timestamp with time zone,
    user_device_push_token_invalidated_reason text,
    user_device_model text,
    user_device_locale text,
    user_device_time_zone text,
    CONSTRAINT user_device_push_token_type_check CHECK (((user_device_push_token_type IS NULL) OR (user_device_push_token_type = 'apns'::text)))
);

ALTER TABLE public.user_devices ALTER COLUMN user_device_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_devices_user_device_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_pkey PRIMARY KEY (user_device_id);

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_uuid_key UNIQUE (user_device_uuid);

CREATE INDEX idx_user_devices_user_id ON public.user_devices USING btree (user_id);
CREATE INDEX idx_user_devices_push_token ON public.user_devices USING btree (user_device_push_token) WHERE (user_device_push_token IS NOT NULL);

-- Device Sessions (note: no hmac_key/counter — using OAuth tokens)
CREATE TABLE public.device_sessions (
    device_session_uuid uuid DEFAULT gen_random_uuid() CONSTRAINT device_sessions_device_session_id_not_null NOT NULL,
    device_session_user_agent text,
    device_session_ip inet,
    device_session_created_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_refreshed_at timestamp with time zone,
    device_session_not_after timestamp with time zone,
    device_session_revoked_at timestamp with time zone,
    device_session_provider text CONSTRAINT device_sessions_device_session_provider_new_not_null NOT NULL,
    device_session_id bigint CONSTRAINT device_sessions_device_session_id_bigint_not_null NOT NULL,
    device_session_user_device_id bigint CONSTRAINT device_sessions_device_session_user_device_id_bigint_not_null NOT NULL,
    user_id bigint CONSTRAINT device_sessions_user_id_bigint_not_null NOT NULL,
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

ALTER TABLE public.device_sessions ALTER COLUMN device_session_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.device_sessions_device_session_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_pkey PRIMARY KEY (device_session_id);

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_uuid_key UNIQUE (device_session_uuid);

CREATE INDEX idx_device_sessions_not_after ON public.device_sessions USING btree (device_session_not_after DESC);
CREATE INDEX idx_device_sessions_user_created ON public.device_sessions USING btree (user_id, device_session_created_at);
CREATE INDEX idx_device_sessions_user_device_id ON public.device_sessions USING btree (device_session_user_device_id);
CREATE INDEX idx_device_sessions_user_id ON public.device_sessions USING btree (user_id);

-- Feature Flags
CREATE TABLE public.feature_flags (
    flag_uuid uuid DEFAULT gen_random_uuid() CONSTRAINT feature_flags_flag_id_not_null NOT NULL,
    flag_name text NOT NULL,
    flag_description text,
    flag_default_enabled boolean DEFAULT false NOT NULL,
    flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint CONSTRAINT feature_flags_flag_id_bigint_not_null NOT NULL
);

ALTER TABLE public.feature_flags ALTER COLUMN flag_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.feature_flags_flag_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_id);

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_flag_name_key UNIQUE (flag_name);

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_uuid_key UNIQUE (flag_uuid);

-- Roles
CREATE TABLE public.roles (
    role_uuid uuid DEFAULT gen_random_uuid() CONSTRAINT roles_role_id_not_null NOT NULL,
    role_name text NOT NULL,
    role_description text,
    role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint CONSTRAINT roles_role_id_bigint_not_null NOT NULL
);

ALTER TABLE public.roles ALTER COLUMN role_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.roles_role_id_bigint_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (role_id);

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_role_name_key UNIQUE (role_name);

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_uuid_key UNIQUE (role_uuid);

-- User Roles
CREATE TABLE public.user_roles (
    user_role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint CONSTRAINT user_roles_role_id_bigint_not_null NOT NULL,
    user_id bigint CONSTRAINT user_roles_user_id_bigint_not_null NOT NULL
);

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);

CREATE INDEX idx_user_roles_user_id ON public.user_roles USING btree (user_id);
CREATE INDEX idx_user_roles_role_id ON public.user_roles USING btree (role_id);

-- Role Feature Flags
CREATE TABLE public.role_feature_flags (
    flag_id bigint CONSTRAINT role_feature_flags_flag_id_bigint_not_null NOT NULL,
    role_id bigint CONSTRAINT role_feature_flags_role_id_bigint_not_null NOT NULL
);

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_pkey PRIMARY KEY (role_id, flag_id);

CREATE INDEX idx_role_feature_flags_flag_id ON public.role_feature_flags USING btree (flag_id);
CREATE INDEX idx_role_feature_flags_role_id ON public.role_feature_flags USING btree (role_id);

-- User Feature Flags
CREATE TABLE public.user_feature_flags (
    user_flag_enabled boolean NOT NULL,
    user_flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint CONSTRAINT user_feature_flags_flag_id_bigint_not_null NOT NULL,
    user_id bigint CONSTRAINT user_feature_flags_user_id_bigint_not_null NOT NULL
);

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_pkey PRIMARY KEY (user_id, flag_id);

CREATE INDEX idx_user_feature_flags_user_id ON public.user_feature_flags USING btree (user_id);
CREATE INDEX idx_user_feature_flags_flag_id ON public.user_feature_flags USING btree (flag_id);

-- Personal Access Tokens
CREATE TABLE public.personal_access_tokens (
    personal_access_token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    personal_access_token_name text NOT NULL,
    personal_access_token_prefix text NOT NULL,
    personal_access_token_token_hash text CONSTRAINT personal_access_tokens_personal_access_token_token_has_not_null NOT NULL,
    personal_access_token_scopes text[],
    personal_access_token_created_at timestamp with time zone DEFAULT now() CONSTRAINT personal_access_tokens_personal_access_token_created_a_not_null NOT NULL,
    personal_access_token_last_used_at timestamp with time zone,
    personal_access_token_expires_at timestamp with time zone,
    personal_access_token_revoked_at timestamp with time zone,
    user_id bigint NOT NULL
);

ALTER TABLE ONLY public.personal_access_tokens
    ADD CONSTRAINT personal_access_tokens_pkey PRIMARY KEY (personal_access_token_id);

CREATE UNIQUE INDEX idx_personal_access_tokens_prefix ON public.personal_access_tokens USING btree (personal_access_token_prefix);
CREATE INDEX idx_personal_access_tokens_user_id ON public.personal_access_tokens USING btree (user_id);

-- Email Change Tokens
CREATE TABLE public.user_email_change_tokens (
    user_email_change_token_id bigint generated always as identity NOT NULL CONSTRAINT user_email_change_tokens_pkey PRIMARY KEY,
    user_email_change_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT user_email_change_tokens_uuid_key UNIQUE,
    user_id bigint NOT NULL CONSTRAINT user_email_change_tokens_user_id_fkey REFERENCES users (user_id) ON DELETE CASCADE,
    user_email_change_target_email text NOT NULL,
    user_email_change_token_hash text NOT NULL CONSTRAINT user_email_change_tokens_token_hash_key UNIQUE,
    user_email_change_expires_at timestamp with time zone NOT NULL,
    user_email_change_consumed_at timestamp with time zone,
    user_email_change_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_email_change_tokens_target_email_not_blank CHECK (btrim(user_email_change_target_email) <> '')
);

CREATE INDEX idx_user_email_change_tokens_user_id ON public.user_email_change_tokens USING btree (user_id);
CREATE INDEX idx_user_email_change_tokens_active_expires_at ON public.user_email_change_tokens USING btree (user_email_change_expires_at) WHERE user_email_change_consumed_at IS NULL;

-- Passkeys / WebAuthn
CREATE TABLE public.user_passkeys (
    user_passkey_id bigint generated always as identity primary key,
    user_passkey_uuid uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    user_identity_id bigint NOT NULL REFERENCES user_identities(user_identity_id) ON DELETE CASCADE,
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

CREATE INDEX idx_user_passkeys_user_id ON public.user_passkeys (user_id);
CREATE INDEX idx_user_passkeys_user_handle ON public.user_passkeys (user_passkey_user_handle);
CREATE INDEX idx_user_passkeys_active_user_id ON public.user_passkeys (user_id) WHERE user_passkey_revoked_at IS NULL;

-- WebAuthn Challenges
CREATE TABLE public.auth_webauthn_challenges (
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
    user_id bigint REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT auth_webauthn_challenge_flow_check CHECK (
        auth_webauthn_challenge_flow = ANY (ARRAY['authenticate'::text, 'register'::text, 'signup'::text])
    )
);

CREATE INDEX idx_auth_webauthn_challenges_uuid ON public.auth_webauthn_challenges (auth_webauthn_challenge_uuid);
CREATE INDEX idx_auth_webauthn_challenges_active ON public.auth_webauthn_challenges (auth_webauthn_challenge_uuid, auth_webauthn_challenge_flow)
    WHERE auth_webauthn_challenge_consumed_at IS NULL;
CREATE INDEX idx_auth_webauthn_challenges_expires ON public.auth_webauthn_challenges (auth_webauthn_challenge_expires_at);

-- Signup Email Tokens
CREATE TABLE public.auth_signup_email_tokens (
    auth_signup_email_token_id bigint generated always as identity NOT NULL CONSTRAINT auth_signup_email_tokens_pkey PRIMARY KEY,
    auth_signup_email_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT auth_signup_email_tokens_uuid_key UNIQUE,
    auth_signup_email_target_email text NOT NULL,
    auth_signup_email_token_hash text NOT NULL CONSTRAINT auth_signup_email_tokens_token_hash_key UNIQUE,
    auth_signup_email_expires_at timestamp with time zone NOT NULL,
    auth_signup_email_consumed_at timestamp with time zone,
    auth_signup_email_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_email_tokens_target_email_not_blank CHECK (btrim(auth_signup_email_target_email) <> '')
);

CREATE INDEX idx_auth_signup_email_tokens_active_expires_at
    ON public.auth_signup_email_tokens USING btree (auth_signup_email_expires_at)
    WHERE auth_signup_email_consumed_at IS NULL;

CREATE INDEX idx_auth_signup_email_tokens_target_email
    ON public.auth_signup_email_tokens USING btree (lower(btrim(auth_signup_email_target_email)));

-- Signup Tickets
CREATE TABLE public.auth_signup_tickets (
    auth_signup_ticket_id bigint generated always as identity NOT NULL CONSTRAINT auth_signup_tickets_pkey PRIMARY KEY,
    auth_signup_ticket_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT auth_signup_tickets_uuid_key UNIQUE,
    auth_signup_ticket_target_email text NOT NULL,
    auth_signup_ticket_hash text NOT NULL CONSTRAINT auth_signup_tickets_hash_key UNIQUE,
    auth_signup_ticket_expires_at timestamp with time zone NOT NULL,
    auth_signup_ticket_consumed_at timestamp with time zone,
    auth_signup_ticket_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_tickets_target_email_not_blank CHECK (btrim(auth_signup_ticket_target_email) <> '')
);

CREATE INDEX idx_auth_signup_tickets_active_expires_at
    ON public.auth_signup_tickets USING btree (auth_signup_ticket_expires_at)
    WHERE auth_signup_ticket_consumed_at IS NULL;

CREATE INDEX idx_auth_signup_tickets_target_email
    ON public.auth_signup_tickets USING btree (lower(btrim(auth_signup_ticket_target_email)));

-- OAuth Authorization Codes
CREATE TABLE public.oauth_authorization_codes (
    oauth_authorization_code_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_authorization_code_code_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL REFERENCES users(user_uuid) ON DELETE CASCADE,
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

CREATE INDEX idx_oauth_authorization_codes_client_id ON public.oauth_authorization_codes (oauth_client_id);
CREATE INDEX idx_oauth_authorization_codes_user_uuid ON public.oauth_authorization_codes (user_uuid);
CREATE INDEX idx_oauth_authorization_codes_expires_at ON public.oauth_authorization_codes (oauth_authorization_code_expires_at);
CREATE INDEX idx_oauth_authorization_codes_audience ON public.oauth_authorization_codes (oauth_authorization_code_audience);

-- OAuth Refresh Tokens
CREATE TABLE public.oauth_refresh_tokens (
    oauth_refresh_token_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_refresh_token_token_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL REFERENCES users(user_uuid) ON DELETE CASCADE,
    oauth_refresh_token_scopes text[] NOT NULL DEFAULT '{}',
    oauth_refresh_token_audience text NOT NULL DEFAULT '',
    oauth_refresh_token_expires_at timestamptz NOT NULL,
    oauth_refresh_token_revoked_at timestamptz,
    oauth_refresh_token_rotated_from uuid REFERENCES oauth_refresh_tokens(oauth_refresh_token_id) ON DELETE SET NULL,
    oauth_refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_refresh_token_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_refresh_tokens_client_id ON public.oauth_refresh_tokens (oauth_client_id);
CREATE INDEX idx_oauth_refresh_tokens_user_uuid ON public.oauth_refresh_tokens (user_uuid);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON public.oauth_refresh_tokens (oauth_refresh_token_expires_at);
CREATE INDEX idx_oauth_refresh_tokens_audience ON public.oauth_refresh_tokens (oauth_refresh_token_audience);

-- OAuth Device Authorizations
CREATE TABLE public.oauth_device_authorizations (
    oauth_device_authorization_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT oauth_device_authorizations_pkey PRIMARY KEY,
    oauth_device_authorization_device_code_hash text NOT NULL CONSTRAINT oauth_device_authorizations_device_code_hash_key UNIQUE,
    oauth_client_id text NOT NULL,
    oauth_device_authorization_user_code text NOT NULL CONSTRAINT oauth_device_authorizations_user_code_key UNIQUE,
    oauth_device_authorization_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    oauth_device_authorization_audience text NOT NULL DEFAULT '',
    user_uuid uuid CONSTRAINT oauth_device_authorizations_user_uuid_fkey REFERENCES users(user_uuid) ON DELETE SET NULL,
    oauth_device_authorization_expires_at timestamp with time zone NOT NULL,
    oauth_device_authorization_approved_at timestamp with time zone,
    oauth_device_authorization_denied_at timestamp with time zone,
    oauth_device_authorization_consumed_at timestamp with time zone,
    oauth_device_authorization_created_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_device_authorization_updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX idx_oauth_device_authorizations_client_id ON public.oauth_device_authorizations (oauth_client_id);
CREATE INDEX idx_oauth_device_authorizations_user_code ON public.oauth_device_authorizations (oauth_device_authorization_user_code);
CREATE INDEX idx_oauth_device_authorizations_expires_at ON public.oauth_device_authorizations (oauth_device_authorization_expires_at);
CREATE INDEX idx_oauth_device_authorizations_audience ON public.oauth_device_authorizations (oauth_device_authorization_audience);

-- OAuth Dynamic Clients
CREATE TABLE public.oauth_dynamic_clients (
    oauth_dynamic_client_id text PRIMARY KEY,
    oauth_dynamic_client_type text NOT NULL DEFAULT 'public',
    oauth_dynamic_client_redirect_uris text[] NOT NULL DEFAULT '{}',
    oauth_dynamic_client_scopes text[] NOT NULL DEFAULT '{}',
    oauth_dynamic_client_token_endpoint_auth_method text NOT NULL DEFAULT 'none',
    oauth_dynamic_client_name text,
    oauth_dynamic_client_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    oauth_dynamic_client_issued_at timestamptz NOT NULL DEFAULT now(),
    oauth_dynamic_client_disabled_at timestamptz,
    oauth_dynamic_client_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_dynamic_client_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_dynamic_clients_disabled_at ON public.oauth_dynamic_clients (oauth_dynamic_client_disabled_at);

-- OAuth Authorization Handoffs
CREATE TABLE public.oauth_authorization_handoffs (
    oauth_authorization_handoff_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_authorization_handoff_token_hash text NOT NULL UNIQUE,
    oauth_authorization_handoff_user_code text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    oauth_authorization_handoff_redirect_uri text NOT NULL,
    oauth_authorization_handoff_scopes text[] NOT NULL DEFAULT '{}',
    oauth_authorization_handoff_audience text NOT NULL DEFAULT '',
    oauth_authorization_handoff_state text NOT NULL DEFAULT '',
    oauth_authorization_handoff_code_challenge text NOT NULL,
    oauth_authorization_handoff_code_challenge_method text NOT NULL,
    user_uuid uuid REFERENCES users(user_uuid) ON DELETE SET NULL,
    oauth_authorization_handoff_authorization_code text,
    oauth_authorization_handoff_redirect_url text,
    oauth_authorization_handoff_denied_at timestamptz,
    oauth_authorization_handoff_completed_at timestamptz,
    oauth_authorization_handoff_expires_at timestamptz NOT NULL,
    oauth_authorization_handoff_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_authorization_handoff_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_authorization_handoffs_client_id ON public.oauth_authorization_handoffs (oauth_client_id);
CREATE INDEX idx_oauth_authorization_handoffs_user_code ON public.oauth_authorization_handoffs (oauth_authorization_handoff_user_code);
CREATE INDEX idx_oauth_authorization_handoffs_expires_at ON public.oauth_authorization_handoffs (oauth_authorization_handoff_expires_at);

-- Runtime Idempotency
CREATE SCHEMA IF NOT EXISTS runtime;

CREATE TABLE IF NOT EXISTS runtime.idempotency_keys (
    scope text NOT NULL,
    actor text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    response_payload bytea,
    lock_expires_at timestamptz NOT NULL,
    result_expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scope, actor, idempotency_key)
);

CREATE INDEX IF NOT EXISTS runtime_idempotency_keys_result_expires_at_idx ON runtime.idempotency_keys (result_expires_at);
CREATE INDEX IF NOT EXISTS runtime_idempotency_keys_lock_expires_at_idx ON runtime.idempotency_keys (lock_expires_at);

-- Foreign Keys
ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_device_session_user_device_id_fkey FOREIGN KEY (device_session_user_device_id) REFERENCES public.user_devices(user_device_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;

ALTER TABLE ONLY public.personal_access_tokens
    ADD CONSTRAINT personal_access_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
