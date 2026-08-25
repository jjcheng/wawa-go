BEGIN;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;
CREATE OR REPLACE FUNCTION update_last_update_column()
RETURNS TRIGGER AS $$
BEGIN
    -- Only operate on UPDATEs
    IF TG_OP = 'UPDATE' THEN
        -- Only consider when the row data actually changed
        IF (NEW.* IS DISTINCT FROM OLD.*) THEN
            -- If caller did not explicitly set last_update (NULL)
            -- or left it equal to the old value, then set it to now().
            -- If caller set a different last_update, respect that value.
            IF NEW.last_update IS NULL OR NEW.last_update = OLD.last_update THEN
                NEW.last_update = now();
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
;
COMMIT;
ANALYZE;