-- First check and create the test database
SELECT 'CREATE DATABASE test'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'test')\gexec

    \c test;

-- Create tables if they don't exist
CREATE TABLE IF NOT EXISTS customers (
                                         customer_id SERIAL PRIMARY KEY,
                                         first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS products (
                                        product_id SERIAL PRIMARY KEY,
                                        name VARCHAR(100),
    description TEXT,
    price NUMERIC(10,2),
    category VARCHAR(50)
    );

CREATE TABLE IF NOT EXISTS sales (
                                     sale_id SERIAL PRIMARY KEY,
                                     customer_id INTEGER REFERENCES customers(customer_id),
    product_id INTEGER REFERENCES products(product_id),
    quantity INTEGER,
    total_amount NUMERIC(10,2),
    sale_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Create or replace views (this is PostgreSQL's way of "IF NOT EXISTS" for views)
CREATE OR REPLACE VIEW customer_sales_summary AS
SELECT
    c.customer_id,
    c.first_name || ' ' || c.last_name AS customer_name,
    COUNT(s.sale_id) AS total_purchases,
    SUM(s.total_amount) AS total_spent,
    MAX(s.sale_date) AS last_purchase_date
FROM customers c
         LEFT JOIN sales s ON c.customer_id = s.customer_id
GROUP BY c.customer_id, c.first_name, c.last_name;

CREATE OR REPLACE VIEW product_sales_summary AS
SELECT
    p.product_id,
    p.name AS product_name,
    p.category,
    COUNT(s.sale_id) AS times_sold,
    SUM(s.quantity) AS total_quantity_sold,
    SUM(s.total_amount) AS total_revenue
FROM products p
         LEFT JOIN sales s ON p.product_id = s.product_id
GROUP BY p.product_id, p.name, p.category;

-- Insert initial customer data (you might want to add ON CONFLICT DO NOTHING to prevent duplicates)
INSERT INTO customers (first_name, last_name, email, phone) VALUES
 ('John', 'Doe', 'john.doe@email.com', '555-0101'),
 ('Jane', 'Smith', 'jane.smith@email.com', '555-0102'),
 ('Robert', 'Johnson', 'robert.j@email.com', '555-0103'),
 ('Sarah', 'Williams', 'sarah.w@email.com', '555-0104'),
 ('Michael', 'Brown', 'michael.b@email.com', '555-0105'),
 ('Emily', 'Davis', 'emily.d@email.com', '555-0106'),
 ('David', 'Miller', 'david.m@email.com', '555-0107'),
 ('Lisa', 'Wilson', 'lisa.w@email.com', '555-0108'),
 ('James', 'Taylor', 'james.t@email.com', '555-0109'),
 ('Emma', 'Anderson', 'emma.a@email.com', '555-0110')
    ON CONFLICT (email) DO NOTHING;

-- Insert initial product data
INSERT INTO products (name, description, price, category) VALUES
                                                              ('Laptop Pro', 'High-performance laptop', 1299.99, 'Electronics'),
                                                              ('Wireless Mouse', 'Ergonomic wireless mouse', 29.99, 'Accessories'),
                                                              ('External SSD', '1TB External SSD Drive', 159.99, 'Storage'),
                                                              ('Gaming Monitor', '27-inch 4K Gaming Monitor', 499.99, 'Electronics'),
                                                              ('Mechanical Keyboard', 'RGB Mechanical Gaming Keyboard', 129.99, 'Accessories'),
                                                              ('Webcam HD', '1080p HD Webcam', 79.99, 'Electronics'),
                                                              ('USB-C Hub', 'Multi-port USB-C Hub', 49.99, 'Accessories'),
                                                              ('Graphics Tablet', 'Digital Drawing Tablet', 199.99, 'Electronics'),
                                                              ('Wireless Earbuds', 'Noise-canceling Earbuds', 149.99, 'Audio'),
                                                              ('Power Bank', '20000mAh Portable Charger', 59.99, 'Accessories')
    ON CONFLICT DO NOTHING;

-- Insert initial sales data
INSERT INTO sales (customer_id, product_id, quantity, total_amount, sale_date) VALUES
 (1, 1, 1, 1299.99, '2024-01-15 10:30:00'),
 (2, 3, 2, 319.98, '2024-01-16 11:45:00'),
 (3, 2, 1, 29.99, '2024-01-17 14:20:00'),
 (4, 5, 1, 129.99, '2024-01-18 16:15:00'),
 (1, 7, 2, 99.98, '2024-02-01 09:30:00'),
 (5, 4, 1, 499.99, '2024-02-03 13:45:00'),
 (6, 8, 1, 199.99, '2024-02-05 15:20:00'),
 (7, 10, 2, 119.98, '2024-02-07 10:10:00'),
 (8, 9, 1, 149.99, '2024-02-10 12:30:00'),
 (9, 6, 1, 79.99, '2024-02-12 14:45:00')
    ON CONFLICT DO NOTHING;