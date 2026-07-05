ALTER TABLE origin.source_listings
    ADD COLUMN IF NOT EXISTS shortcut_ad_id bigint,
    ADD COLUMN IF NOT EXISTS frontdoor_ad_id uuid,
    ADD COLUMN IF NOT EXISTS frontdoor_building_announcement_id uuid;
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'origin'
            AND table_name = 'source_listings'
            AND column_name = 'raw_table'
    ) THEN
        EXECUTE $sql$
            UPDATE origin.source_listings
            SET shortcut_ad_id = raw_id::bigint
            WHERE raw_table = 'shortcut_ads'
                AND shortcut_ad_id IS NULL
                AND raw_id ~ '^[0-9]+$'
        $sql$;
        EXECUTE $sql$
            UPDATE origin.source_listings
            SET frontdoor_ad_id = raw_id::uuid
            WHERE raw_table = 'frontdoor_ads'
                AND frontdoor_ad_id IS NULL
        $sql$;
        EXECUTE $sql$
            UPDATE origin.source_listings
            SET frontdoor_building_announcement_id = raw_id::uuid
            WHERE raw_table = 'frontdoor_building_announcements'
                AND frontdoor_building_announcement_id IS NULL
        $sql$;
    END IF;
END $$;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'origin'::regnamespace
            AND conrelid = 'origin.source_listings'::regclass
            AND conname = 'source_listings_one_source_check'
    ) THEN
        ALTER TABLE origin.source_listings
            ADD CONSTRAINT source_listings_one_source_check CHECK ((num_nonnulls(shortcut_ad_id, frontdoor_ad_id, frontdoor_building_announcement_id) = 1));
    END IF;
END $$;
DROP INDEX IF EXISTS origin.idx_source_listings_raw;
CREATE UNIQUE INDEX IF NOT EXISTS idx_source_listings_frontdoor_ad ON origin.source_listings USING btree (frontdoor_ad_id) WHERE (frontdoor_ad_id IS NOT NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_source_listings_frontdoor_building_announcement ON origin.source_listings USING btree (frontdoor_building_announcement_id) WHERE (frontdoor_building_announcement_id IS NOT NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_source_listings_shortcut_ad ON origin.source_listings USING btree (shortcut_ad_id) WHERE (shortcut_ad_id IS NOT NULL);
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'origin'::regnamespace
            AND conrelid = 'origin.source_listings'::regclass
            AND conname = 'source_listings_frontdoor_ad_id_fkey'
    ) THEN
        ALTER TABLE origin.source_listings
            ADD CONSTRAINT source_listings_frontdoor_ad_id_fkey FOREIGN KEY (frontdoor_ad_id) REFERENCES origin.frontdoor_ads(frontdoor_ad_id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'origin'::regnamespace
            AND conrelid = 'origin.source_listings'::regclass
            AND conname = 'source_listings_frontdoor_building_announcement_id_fkey'
    ) THEN
        ALTER TABLE origin.source_listings
            ADD CONSTRAINT source_listings_frontdoor_building_announcement_id_fkey FOREIGN KEY (frontdoor_building_announcement_id) REFERENCES origin.frontdoor_building_announcements(frontdoor_building_announcement_id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'origin'::regnamespace
            AND conrelid = 'origin.source_listings'::regclass
            AND conname = 'source_listings_shortcut_ad_id_fkey'
    ) THEN
        ALTER TABLE origin.source_listings
            ADD CONSTRAINT source_listings_shortcut_ad_id_fkey FOREIGN KEY (shortcut_ad_id) REFERENCES origin.shortcut_ads(shortcut_ad_id) ON DELETE CASCADE;
    END IF;
END $$;
ALTER TABLE origin.source_listings
    DROP COLUMN IF EXISTS raw_table,
    DROP COLUMN IF EXISTS raw_id;
