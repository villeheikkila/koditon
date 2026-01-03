-- Auth schema for JWT-based authentication with Sign in with Apple support
-- This file is used by sqlc to generate Go types

CREATE SCHEMA IF NOT EXISTS auth;

CREATE TYPE auth.auth_provider AS ENUM ('apple', 'anonymous');

CREATE TYPE auth.push_token_type AS ENUM ('apns');

CREATE TABLE auth.users (
    user_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_created_at timestamptz NOT NULL DEFAULT now(),
    user_updated_at timestamptz NOT NULL DEFAULT now(),
    user_deleted_at timestamptz
);

CREATE TABLE auth.identities (
    identity_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    identity_provider auth.auth_provider NOT NULL,
    identity_external_id text NOT NULL,
    identity_email text,
    identity_email_verified bool DEFAULT false,
    identity_data jsonb DEFAULT '{}'::jsonb,
    identity_created_at timestamptz NOT NULL DEFAULT now(),
    identity_updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identities_provider_external_id_unique UNIQUE (identity_provider, identity_external_id)
);

CREATE TABLE auth.devices (
    device_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    device_name text,
    device_os text,
    device_app_version text,
    device_push_token text,
    device_push_token_type auth.push_token_type,
    device_push_token_updated_at timestamptz,
    device_created_at timestamptz NOT NULL DEFAULT now(),
    device_updated_at timestamptz NOT NULL DEFAULT now(),
    device_last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.sessions (
    session_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    session_device_id uuid REFERENCES auth.devices(device_id) ON DELETE SET NULL,
    session_user_agent text,
    session_ip inet,
    session_provider auth.auth_provider NOT NULL,
    session_refresh_token_hmac_key text NOT NULL,
    session_refresh_token_counter int8 NOT NULL DEFAULT 0,
    session_created_at timestamptz NOT NULL DEFAULT now(),
    session_updated_at timestamptz NOT NULL DEFAULT now(),
    session_refreshed_at timestamptz,
    session_not_after timestamptz,
    session_revoked_at timestamptz
);

CREATE TABLE auth.refresh_tokens (
    refresh_token_id bigserial PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES auth.sessions(session_id) ON DELETE CASCADE,
    refresh_token_token_hash text NOT NULL,
    refresh_token_counter int8 NOT NULL,
    refresh_token_revoked bool NOT NULL DEFAULT false,
    refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    refresh_token_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.roles (
    role_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_name text NOT NULL UNIQUE,
    role_description text,
    role_created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.feature_flags (
    flag_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    flag_name text NOT NULL UNIQUE,
    flag_description text,
    flag_default_enabled bool NOT NULL DEFAULT false,
    flag_created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.role_feature_flags (
    role_id uuid NOT NULL REFERENCES auth.roles(role_id) ON DELETE CASCADE,
    flag_id uuid NOT NULL REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, flag_id)
);

CREATE TABLE auth.user_roles (
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES auth.roles(role_id) ON DELETE CASCADE,
    user_role_created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE auth.user_feature_flags (
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    flag_id uuid NOT NULL REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE,
    user_flag_enabled bool NOT NULL,
    user_flag_created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, flag_id)
);
