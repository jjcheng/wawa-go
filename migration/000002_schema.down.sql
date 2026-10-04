BEGIN;
ALTER TABLE "customer"."customers" ADD COLUMN "agent_running" boolean NOT NULL;
ALTER TABLE "wa"."messages" DROP COLUMN "by_agent";
COMMIT;