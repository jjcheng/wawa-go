BEGIN;
ALTER TABLE "account"."notifications" DROP COLUMN "icon_type", DROP COLUMN "category";
ALTER TABLE "wa"."messages" DROP CONSTRAINT "messages_user_id_fkey", DROP COLUMN "sender_user_id", ADD COLUMN "user_id" integer NOT NULL, ADD
CONSTRAINT "messages_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "account"."users" ("id") ON UPDATE CASCADE ON DELETE RESTRICT;
COMMIT;