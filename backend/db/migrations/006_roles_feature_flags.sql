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

CREATE INDEX idx_user_roles_user_id ON auth.user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON auth.user_roles(role_id);
CREATE INDEX idx_role_feature_flags_role_id ON auth.role_feature_flags(role_id);
CREATE INDEX idx_user_feature_flags_user_id ON auth.user_feature_flags(user_id);

INSERT INTO auth.roles (role_name, role_description) VALUES
    ('user', 'Default user role'),
    ('admin', 'Administrator with full access');

INSERT INTO auth.feature_flags (flag_name, flag_description, flag_default_enabled) VALUES
    ('experiment', 'Experimental feature for testing', false);
