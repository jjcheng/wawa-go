BEGIN;
ALTER TABLE "customer"."customers" RENAME COLUMN "from_incoming_message" TO "from_webhook";
ALTER TABLE "customer"."customers" ALTER COLUMN "from_webhook" SET DEFAULT FALSE;
ALTER TABLE "wa"."messages" ADD COLUMN "echo" boolean NOT NULL DEFAULT FALSE;
ALTER TABLE "wa"."messages" ALTER COLUMN "echo" DROP DEFAULT;
ALTER TABLE "wa"."phone_numbers" ADD COLUMN "contact_sync_request_id" text NOT NULL DEFAULT '', ADD COLUMN "history_sync_request_id" text NOT NULL DEFAULT '';
ALTER TABLE "wa"."phone_numbers" ALTER COLUMN "contact_sync_request_id" DROP DEFAULT, ALTER COLUMN "history_sync_request_id" DROP DEFAULT;
COMMIT;
ANALYZE;