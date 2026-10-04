BEGIN;
ALTER TABLE "wa"."phone_numbers" DROP COLUMN "agent_enabled", ADD COLUMN "agent_running" boolean NOT NULL;
COMMIT;