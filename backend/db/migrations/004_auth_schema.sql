CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE auth.users (
    user_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_created_at timestamptz NOT NULL DEFAULT now(),
    user_updated_at timestamptz NOT NULL DEFAULT now(),
    user_deleted_at timestamptz
);

CREATE INDEX idx_auth_users_created_at ON auth.users(user_created_at);

CREATE TYPE auth.auth_provider AS ENUM ('apple', 'anonymous');

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

CREATE INDEX idx_auth_identities_user_id ON auth.identities(user_id);
CREATE INDEX idx_auth_identities_provider ON auth.identities(identity_provider);
CREATE INDEX idx_auth_identities_external_id ON auth.identities(identity_provider, identity_external_id);

CREATE TABLE auth.sessions (
    session_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    session_device_id uuid,
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

COMMENT ON COLUMN auth.sessions.session_not_after IS 'Session expiration time - session is invalid after this timestamp';
COMMENT ON COLUMN auth.sessions.session_refresh_token_hmac_key IS 'HMAC-SHA256 key used to sign refresh tokens for this session';
COMMENT ON COLUMN auth.sessions.session_refresh_token_counter IS 'Counter for the last issued refresh token';

CREATE INDEX idx_auth_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_auth_sessions_not_after ON auth.sessions(session_not_after DESC);
CREATE INDEX idx_auth_sessions_user_created ON auth.sessions(user_id, session_created_at);

CREATE TABLE auth.refresh_tokens (
    refresh_token_id bigserial PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES auth.sessions(session_id) ON DELETE CASCADE,
    refresh_token_token_hash text NOT NULL,
    refresh_token_counter int8 NOT NULL,
    refresh_token_revoked bool NOT NULL DEFAULT false,
    refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    refresh_token_updated_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE auth.refresh_tokens IS 'Store of tokens used to refresh JWT tokens once they expire';

CREATE UNIQUE INDEX idx_auth_refresh_tokens_token_hash ON auth.refresh_tokens(refresh_token_token_hash);
CREATE INDEX idx_auth_refresh_tokens_session_id ON auth.refresh_tokens(session_id);
CREATE INDEX idx_auth_refresh_tokens_session_revoked ON auth.refresh_tokens(session_id, refresh_token_revoked);
CREATE INDEX idx_auth_refresh_tokens_updated_at ON auth.refresh_tokens(refresh_token_updated_at DESC);
