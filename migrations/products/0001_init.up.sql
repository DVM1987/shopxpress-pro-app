CREATE TABLE IF NOT EXISTS products (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    price      NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    stock      INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO products (id, name, price, stock) VALUES
    ('p-001', 'Wireless Mouse',      19.99, 42),
    ('p-002', 'Mechanical Keyboard', 89.00, 15),
    ('p-003', 'USB-C Hub',           34.50,  0)
ON CONFLICT (id) DO NOTHING;
