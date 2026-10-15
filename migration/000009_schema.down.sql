BEGIN;
ALTER TABLE "ai_agent"."skills" DROP COLUMN "always_apply", DROP COLUMN "inputs", DROP COLUMN "call_function";
COMMIT;