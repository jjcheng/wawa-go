BEGIN;
ALTER TABLE "customer"."customers" DROP COLUMN "agent_running";
ALTER TABLE "wa"."messages" ADD COLUMN "by_agent" boolean NOT NULL;
COMMIT;
ANALYZE;