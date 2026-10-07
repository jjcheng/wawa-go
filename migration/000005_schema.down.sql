BEGIN;
DROP SCHEMA "ai_worker" CASCADE;
DROP TABLE "ai_worker"."conversations";
DROP INDEX "ai_worker"."conversations_user_id_idx";
DROP TABLE "ai_worker"."messages";
DROP INDEX "ai_worker"."messages_conversation_id_idx";
DROP TRIGGER conversations_set_last_update ON ai_worker.conversations;
DROP TRIGGER messages_set_last_update ON ai_worker.messages;
COMMIT;