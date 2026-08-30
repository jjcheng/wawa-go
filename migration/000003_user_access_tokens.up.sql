BEGIN;
ALTER TABLE account.users ADD COLUMN access_token_hash text NULL;
ALTER TABLE account.users ADD COLUMN access_token_expiry timestamptz NULL;
CREATE UNIQUE INDEX users_access_token_hash_unique ON account.users (access_token_hash);
DROP TABLE account.sessions;
COMMIT;