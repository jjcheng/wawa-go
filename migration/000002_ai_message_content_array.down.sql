DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_schema = 'ai'
			AND table_name = 'messages'
			AND column_name = 'content'
			AND data_type = 'ARRAY'
	) THEN
		ALTER TABLE ai.messages
			ALTER COLUMN content TYPE text
			USING array_to_string(content, E'\n');
	END IF;
END;
$$;