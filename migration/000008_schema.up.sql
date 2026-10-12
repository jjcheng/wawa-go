BEGIN;
ALTER TABLE "customer"."customers" ALTER COLUMN "from_webhook" DROP DEFAULT;
ALTER TABLE "wa"."messages" ADD COLUMN "read" boolean NOT NULL DEFAULT false;
COMMIT;
ANALYZE;