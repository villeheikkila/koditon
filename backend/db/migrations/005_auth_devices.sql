CREATE TYPE auth.push_token_type AS ENUM ('apns');

CREATE TABLE auth.devices (
    device_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    device_name TEXT,
    device_os TEXT,
    device_app_version TEXT,
    device_push_token TEXT,
    device_push_token_type auth.push_token_type,
    device_push_token_updated_at TIMESTAMPTZ,
    device_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    device_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    device_last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE auth.devices IS 'User devices for push notifications and session tracking';
COMMENT ON COLUMN auth.devices.device_push_token IS 'Push notification token (APNs device token)';
COMMENT ON COLUMN auth.devices.device_push_token_type IS 'Type of push token (apns, etc.)';

CREATE INDEX idx_auth_devices_user_id ON auth.devices(user_id);
CREATE INDEX idx_auth_devices_push_token ON auth.devices(device_push_token) WHERE device_push_token IS NOT NULL;

ALTER TABLE auth.sessions
    ADD CONSTRAINT fk_sessions_device_id
    FOREIGN KEY (session_device_id) REFERENCES auth.devices(device_id) ON DELETE SET NULL;
