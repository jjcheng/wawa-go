BEGIN;
DROP TABLE "wa"."business_agent_keywords";
DROP INDEX "wa"."business_agent_keywords_phone_number_id_idx";
DROP TRIGGER business_agent_keywords_set_last_update ON wa.business_agent_keywords;
COMMIT;