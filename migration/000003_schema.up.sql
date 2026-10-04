BEGIN;
ALTER TABLE "wa"."phone_numbers" DROP COLUMN "agent_running", ADD COLUMN "agent_enabled" boolean NOT NULL;
COMMIT;
ANALYZE;