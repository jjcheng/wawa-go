BEGIN;
ALTER TABLE "customer"."customers" ALTER COLUMN "from_webhook" SET DEFAULT false;
ALTER TABLE "wa"."messages" DROP COLUMN "read";
COMMIT;