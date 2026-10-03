BEGIN;
DROP SCHEMA "ai" CASCADE;
DROP TABLE "ai"."conversations";
DROP INDEX "ai"."conversations_user_id_idx";
DROP TABLE "ai"."messages";
DROP INDEX "ai"."messages_conversation_id_idx";
DROP TRIGGER conversations_set_last_update ON ai.conversations;
DROP TRIGGER messages_set_last_update ON ai.messages;
COMMIT;