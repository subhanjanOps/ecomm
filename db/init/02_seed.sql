INSERT INTO users (email, password_hash, name)
VALUES ('demo@example.com', '$2y$10$examplehashnotreal', 'Demo User')
ON CONFLICT (email) DO NOTHING;

INSERT INTO products (sku, name, description, price_cents, currency, stock) VALUES
('SKU-001', 'T-Shirt', 'Classic tee', 1999, 'USD', 100),
('SKU-002', 'Mug', 'Ceramic mug', 1299, 'USD', 200),
('SKU-003', 'Sticker Pack', 'Assorted stickers', 599, 'USD', 500)
ON CONFLICT (sku) DO NOTHING;
