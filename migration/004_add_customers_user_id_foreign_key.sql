ALTER TABLE customer.customers
    ADD CONSTRAINT customers_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES account.users (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT;
