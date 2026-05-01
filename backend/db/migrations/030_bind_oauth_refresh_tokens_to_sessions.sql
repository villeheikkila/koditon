ALTER TABLE public.oauth_refresh_tokens
    ADD COLUMN IF NOT EXISTS device_session_uuid uuid REFERENCES public.device_sessions(device_session_uuid) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_device_session_uuid ON public.oauth_refresh_tokens (device_session_uuid);
