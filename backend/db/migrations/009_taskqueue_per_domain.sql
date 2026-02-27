-- Migration: Replace monolithic task_queue schema with per-domain pgmq queues + sync_tasks tables
-- Each domain gets its own pgmq queue and a sync_tasks table that is the definitive source
-- of entity sync status. One row per (entity_id, task_type) — reused across sync cycles.

-- Step 1: Create per-domain pgmq queues
SELECT pgmq.create('frontdoor');
SELECT pgmq.create('shortcut');
SELECT pgmq.create('prices');
SELECT pgmq.create('postal');

-- Step 2: Create per-domain sync_tasks tables
-- Key design: UNIQUE (entity_id, task_type) — one row per entity, reused across cycles.
-- The upsert resets completed/failed rows to pending; skips already-active rows.
-- Table size is bounded by number of entities, not number of sync runs.

CREATE TABLE public.frontdoor_sync_tasks (
    frontdoor_sync_task_id BIGSERIAL PRIMARY KEY,
    frontdoor_sync_task_entity_id TEXT NOT NULL,
    frontdoor_sync_task_type TEXT NOT NULL,
    frontdoor_sync_task_status TEXT NOT NULL DEFAULT 'pending',
    frontdoor_sync_task_priority INT NOT NULL DEFAULT 0,
    frontdoor_sync_task_max_attempts INT NOT NULL DEFAULT 3,
    frontdoor_sync_task_attempts INT NOT NULL DEFAULT 0,
    frontdoor_sync_task_last_error TEXT,
    frontdoor_sync_task_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    frontdoor_sync_task_started_at TIMESTAMPTZ,
    frontdoor_sync_task_completed_at TIMESTAMPTZ,
    UNIQUE (frontdoor_sync_task_entity_id, frontdoor_sync_task_type)
);

CREATE TABLE public.shortcut_sync_tasks (
    shortcut_sync_task_id BIGSERIAL PRIMARY KEY,
    shortcut_sync_task_entity_id TEXT NOT NULL,
    shortcut_sync_task_type TEXT NOT NULL,
    shortcut_sync_task_status TEXT NOT NULL DEFAULT 'pending',
    shortcut_sync_task_priority INT NOT NULL DEFAULT 0,
    shortcut_sync_task_max_attempts INT NOT NULL DEFAULT 3,
    shortcut_sync_task_attempts INT NOT NULL DEFAULT 0,
    shortcut_sync_task_last_error TEXT,
    shortcut_sync_task_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    shortcut_sync_task_started_at TIMESTAMPTZ,
    shortcut_sync_task_completed_at TIMESTAMPTZ,
    UNIQUE (shortcut_sync_task_entity_id, shortcut_sync_task_type)
);

CREATE TABLE public.prices_sync_tasks (
    prices_sync_task_id BIGSERIAL PRIMARY KEY,
    prices_sync_task_entity_id TEXT NOT NULL,
    prices_sync_task_type TEXT NOT NULL,
    prices_sync_task_status TEXT NOT NULL DEFAULT 'pending',
    prices_sync_task_priority INT NOT NULL DEFAULT 0,
    prices_sync_task_max_attempts INT NOT NULL DEFAULT 3,
    prices_sync_task_attempts INT NOT NULL DEFAULT 0,
    prices_sync_task_last_error TEXT,
    prices_sync_task_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    prices_sync_task_started_at TIMESTAMPTZ,
    prices_sync_task_completed_at TIMESTAMPTZ,
    UNIQUE (prices_sync_task_entity_id, prices_sync_task_type)
);

CREATE TABLE public.postal_sync_tasks (
    postal_sync_task_id BIGSERIAL PRIMARY KEY,
    postal_sync_task_entity_id TEXT NOT NULL,
    postal_sync_task_type TEXT NOT NULL,
    postal_sync_task_status TEXT NOT NULL DEFAULT 'pending',
    postal_sync_task_priority INT NOT NULL DEFAULT 0,
    postal_sync_task_max_attempts INT NOT NULL DEFAULT 3,
    postal_sync_task_attempts INT NOT NULL DEFAULT 0,
    postal_sync_task_last_error TEXT,
    postal_sync_task_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    postal_sync_task_started_at TIMESTAMPTZ,
    postal_sync_task_completed_at TIMESTAMPTZ,
    UNIQUE (postal_sync_task_entity_id, postal_sync_task_type)
);

-- Step 3: Remove old cron jobs
SELECT cron.unschedule(jobname)
FROM cron.job
WHERE jobname IN (
    'trigger-frontdoor-sitemap-sync',
    'trigger-frontdoor-daily-syncs',
    'cleanup-old-completed-tasks',
    'requeue-stuck-tasks',
    'trigger-shortcut-sitemap-sync',
    'trigger-shortcut-daily-scraper-syncs',
    'trigger-shortcut-daily-api-syncs',
    'trigger-prices-cities-init',
    'trigger-prices-daily-syncs',
    'trigger-prices-sync-all',
    'trigger-prices-neighborhood-postal-code-sync'
);

-- Step 4: Drop old task_queue views first (views depend on tables)
DROP VIEW IF EXISTS task_queue.vw_entity_sync_health CASCADE;
DROP VIEW IF EXISTS task_queue.fnc__status_summary CASCADE;
DROP VIEW IF EXISTS task_queue.fnc__active_workers CASCADE;
DROP VIEW IF EXISTS task_queue.fnc__recent_failures CASCADE;
DROP VIEW IF EXISTS task_queue.fnc__stuck_tasks CASCADE;
DROP VIEW IF EXISTS task_queue.fnc__daily_progress CASCADE;

-- Step 5: Drop old task_queue functions
DROP FUNCTION IF EXISTS task_queue.fnc__register_entity(TEXT, TEXT, TEXT, TEXT, JSONB) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__register_entities(TEXT[], TEXT, TEXT) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__enqueue_task(BIGINT) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_daily_syncs(TEXT) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__requeue_stuck_tasks() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__move_to_dlq(BIGINT, TEXT) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__requeue_from_dlq(BIGINT, INT, INT) CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_frontdoor_sitemap_sync() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_shortcut_sitemap_sync() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_prices_cities_init() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_prices_sync_all() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_prices_neighborhood_postal_code_sync() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_postal_sync() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__get_sync_statistics() CASCADE;
DROP FUNCTION IF EXISTS task_queue.fnc__get_entity_sync_status(TEXT) CASCADE;

-- Step 6: Drop old task_queue tables
DROP TABLE IF EXISTS task_queue.dead_letter_queue CASCADE;
DROP TABLE IF EXISTS task_queue.task CASCADE;
DROP TABLE IF EXISTS task_queue.task_type_entity_type_mapping CASCADE;
DROP TABLE IF EXISTS task_queue.entity_registry CASCADE;

-- Step 7: Drop old pgmq queues
SELECT pgmq.drop_queue('tasks');

-- Step 8: Drop old schema
DROP SCHEMA IF EXISTS task_queue CASCADE;
