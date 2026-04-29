CREATE TABLE IF NOT EXISTS public.feature_flags (
    flag_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT feature_flags_uuid_key UNIQUE,
    flag_name text NOT NULL CONSTRAINT feature_flags_flag_name_key UNIQUE,
    flag_description text,
    flag_default_enabled boolean DEFAULT false NOT NULL,
    flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint generated always as identity NOT NULL CONSTRAINT feature_flags_pkey PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS public.roles (
    role_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT roles_uuid_key UNIQUE,
    role_name text NOT NULL CONSTRAINT roles_role_name_key UNIQUE,
    role_description text,
    role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint generated always as identity NOT NULL CONSTRAINT roles_pkey PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS public.user_roles (
    user_role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint NOT NULL CONSTRAINT user_roles_role_id_fkey REFERENCES public.roles(role_id) ON DELETE CASCADE,
    user_id bigint NOT NULL CONSTRAINT user_roles_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON public.user_roles USING btree (user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON public.user_roles USING btree (role_id);
CREATE TABLE IF NOT EXISTS public.role_feature_flags (
    flag_id bigint NOT NULL CONSTRAINT role_feature_flags_flag_id_fkey REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE,
    role_id bigint NOT NULL CONSTRAINT role_feature_flags_role_id_fkey REFERENCES public.roles(role_id) ON DELETE CASCADE,
    CONSTRAINT role_feature_flags_pkey PRIMARY KEY (role_id, flag_id)
);
CREATE INDEX IF NOT EXISTS idx_role_feature_flags_flag_id ON public.role_feature_flags USING btree (flag_id);
CREATE INDEX IF NOT EXISTS idx_role_feature_flags_role_id ON public.role_feature_flags USING btree (role_id);
CREATE TABLE IF NOT EXISTS public.user_feature_flags (
    user_flag_enabled boolean NOT NULL,
    user_flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint NOT NULL CONSTRAINT user_feature_flags_flag_id_fkey REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE,
    user_id bigint NOT NULL CONSTRAINT user_feature_flags_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT user_feature_flags_pkey PRIMARY KEY (user_id, flag_id)
);
CREATE INDEX IF NOT EXISTS idx_user_feature_flags_user_id ON public.user_feature_flags USING btree (user_id);
CREATE INDEX IF NOT EXISTS idx_user_feature_flags_flag_id ON public.user_feature_flags USING btree (flag_id);
CREATE TABLE IF NOT EXISTS public.personal_access_tokens (
    personal_access_token_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT personal_access_tokens_pkey PRIMARY KEY,
    personal_access_token_name text NOT NULL,
    personal_access_token_prefix text NOT NULL,
    personal_access_token_token_hash text NOT NULL,
    personal_access_token_scopes text[],
    personal_access_token_created_at timestamp with time zone DEFAULT now() NOT NULL,
    personal_access_token_last_used_at timestamp with time zone,
    personal_access_token_expires_at timestamp with time zone,
    personal_access_token_revoked_at timestamp with time zone,
    user_id bigint NOT NULL CONSTRAINT personal_access_tokens_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_personal_access_tokens_prefix ON public.personal_access_tokens USING btree (personal_access_token_prefix);
CREATE INDEX IF NOT EXISTS idx_personal_access_tokens_user_id ON public.personal_access_tokens USING btree (user_id);
CREATE TABLE IF NOT EXISTS public.user_email_change_tokens (
    user_email_change_token_id bigint generated always as identity NOT NULL CONSTRAINT user_email_change_tokens_pkey PRIMARY KEY,
    user_email_change_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT user_email_change_tokens_uuid_key UNIQUE,
    user_id bigint NOT NULL CONSTRAINT user_email_change_tokens_user_id_fkey REFERENCES public.users(user_id) ON DELETE CASCADE,
    user_email_change_target_email text NOT NULL,
    user_email_change_token_hash text NOT NULL CONSTRAINT user_email_change_tokens_token_hash_key UNIQUE,
    user_email_change_expires_at timestamp with time zone NOT NULL,
    user_email_change_consumed_at timestamp with time zone,
    user_email_change_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_email_change_tokens_target_email_not_blank CHECK (btrim(user_email_change_target_email) <> '')
);
CREATE INDEX IF NOT EXISTS idx_user_email_change_tokens_user_id ON public.user_email_change_tokens USING btree (user_id);
CREATE INDEX IF NOT EXISTS idx_user_email_change_tokens_active_expires_at ON public.user_email_change_tokens USING btree (user_email_change_expires_at) WHERE user_email_change_consumed_at IS NULL;
CREATE TABLE IF NOT EXISTS public.oauth_device_authorizations (
    oauth_device_authorization_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT oauth_device_authorizations_pkey PRIMARY KEY,
    oauth_device_authorization_device_code_hash text NOT NULL CONSTRAINT oauth_device_authorizations_device_code_hash_key UNIQUE,
    oauth_client_id text NOT NULL,
    oauth_device_authorization_user_code text NOT NULL CONSTRAINT oauth_device_authorizations_user_code_key UNIQUE,
    oauth_device_authorization_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    oauth_device_authorization_audience text NOT NULL DEFAULT '',
    user_uuid uuid CONSTRAINT oauth_device_authorizations_user_uuid_fkey REFERENCES public.users(user_uuid) ON DELETE SET NULL,
    oauth_device_authorization_expires_at timestamp with time zone NOT NULL,
    oauth_device_authorization_approved_at timestamp with time zone,
    oauth_device_authorization_denied_at timestamp with time zone,
    oauth_device_authorization_consumed_at timestamp with time zone,
    oauth_device_authorization_created_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_device_authorization_updated_at timestamp with time zone DEFAULT now() NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_oauth_device_authorizations_client_id ON public.oauth_device_authorizations (oauth_client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_device_authorizations_user_code ON public.oauth_device_authorizations (oauth_device_authorization_user_code);
CREATE INDEX IF NOT EXISTS idx_oauth_device_authorizations_expires_at ON public.oauth_device_authorizations (oauth_device_authorization_expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_device_authorizations_audience ON public.oauth_device_authorizations (oauth_device_authorization_audience);
CREATE TABLE IF NOT EXISTS public.oauth_dynamic_clients (
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
CREATE INDEX IF NOT EXISTS idx_oauth_dynamic_clients_disabled_at ON public.oauth_dynamic_clients (oauth_dynamic_client_disabled_at);
CREATE TABLE IF NOT EXISTS public.oauth_authorization_handoffs (
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
    user_uuid uuid REFERENCES public.users(user_uuid) ON DELETE SET NULL,
    oauth_authorization_handoff_authorization_code text,
    oauth_authorization_handoff_redirect_url text,
    oauth_authorization_handoff_denied_at timestamptz,
    oauth_authorization_handoff_completed_at timestamptz,
    oauth_authorization_handoff_expires_at timestamptz NOT NULL,
    oauth_authorization_handoff_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_authorization_handoff_updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_handoffs_client_id ON public.oauth_authorization_handoffs (oauth_client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_handoffs_user_code ON public.oauth_authorization_handoffs (oauth_authorization_handoff_user_code);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_handoffs_expires_at ON public.oauth_authorization_handoffs (oauth_authorization_handoff_expires_at);
