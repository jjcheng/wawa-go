BEGIN;
ALTER TABLE "account"."notifications" ADD COLUMN "category" text NOT NULL, ADD COLUMN "icon_type" text NOT NULL;
ALTER TABLE "wa"."messages" DROP CONSTRAINT "messages_user_id_fkey", DROP COLUMN "user_id", ADD COLUMN "sender_user_id" integer NULL, ADD
CONSTRAINT "messages_user_id_fkey" FOREIGN KEY ("sender_user_id") REFERENCES "account"."users" ("id") ON UPDATE CASCADE ON DELETE RESTRICT;
COMMIT;
ANALYZE;