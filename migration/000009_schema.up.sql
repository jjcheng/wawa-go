BEGIN;
ALTER TABLE "ai_agent"."skills" ADD COLUMN "call_function" boolean NOT NULL DEFAULT false, ADD COLUMN "inputs" jsonb NULL, ADD COLUMN "always_apply" boolean NOT NULL DEFAULT false;
COMMIT;
ANALYZE;