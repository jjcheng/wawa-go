BEGIN;
ALTER TABLE "customer"."customers" RENAME COLUMN "from_webhook" TO "from_incoming_message";
ALTER TABLE "customer"."customers" ALTER COLUMN "from_incoming_message" DROP DEFAULT;
ALTER TABLE "wa"."messages" DROP COLUMN "echo";
ALTER TABLE "wa"."phone_numbers" DROP COLUMN "history_sync_request_id", DROP COLUMN "contact_sync_request_id";
COMMIT;