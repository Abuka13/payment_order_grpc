CREATE TABLE IF NOT EXISTS orders (
                                      id TEXT PRIMARY KEY,
                                      customer_id TEXT NOT NULL,
                                      item_name TEXT NOT NULL,
                                      amount BIGINT NOT NULL CHECK (amount > 0),
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
    );