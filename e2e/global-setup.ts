import { FullConfig } from '@playwright/test';
import { Client } from 'pg';
import path from 'path';
import fs from 'fs';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config({ path: path.join(__dirname, '.env') });

// Test schema SQL - creates tables, views, functions, etc.
const TEST_SCHEMA_SQL = `
-- Drop existing test schema if exists
DROP SCHEMA IF EXISTS test_schema CASCADE;

-- Create test schema
CREATE SCHEMA test_schema;

-- Users table
CREATE TABLE test_schema.users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE test_schema.products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  stock INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders table with FK to users
CREATE TABLE test_schema.orders (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES test_schema.users(id),
  total DECIMAL(10,2) NOT NULL,
  status VARCHAR(50) DEFAULT 'pending',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Order items (junction table)
CREATE TABLE test_schema.order_items (
  id SERIAL PRIMARY KEY,
  order_id INTEGER NOT NULL REFERENCES test_schema.orders(id),
  product_id INTEGER NOT NULL REFERENCES test_schema.products(id),
  quantity INTEGER NOT NULL,
  unit_price DECIMAL(10,2) NOT NULL
);

-- Create indexes
CREATE INDEX idx_orders_user_id ON test_schema.orders(user_id);
CREATE INDEX idx_orders_status ON test_schema.orders(status);
CREATE INDEX idx_order_items_order_id ON test_schema.order_items(order_id);
CREATE INDEX idx_order_items_product_id ON test_schema.order_items(product_id);

-- Test view
CREATE VIEW test_schema.user_order_summary AS
SELECT
  u.id,
  u.name,
  u.email,
  COUNT(o.id) as order_count,
  COALESCE(SUM(o.total), 0) as total_spent
FROM test_schema.users u
LEFT JOIN test_schema.orders o ON u.id = o.user_id
GROUP BY u.id, u.name, u.email;

-- Test function
CREATE OR REPLACE FUNCTION test_schema.get_user_orders(p_user_id INTEGER)
RETURNS TABLE(order_id INTEGER, total DECIMAL, status VARCHAR, created_at TIMESTAMP) AS $$
BEGIN
  RETURN QUERY
  SELECT o.id, o.total, o.status, o.created_at
  FROM test_schema.orders o
  WHERE o.user_id = p_user_id
  ORDER BY o.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Test sequence
CREATE SEQUENCE test_schema.custom_seq START 1000 INCREMENT 10;

-- Test enum type
CREATE TYPE test_schema.order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');

-- Insert test data
INSERT INTO test_schema.users (name, email) VALUES
  ('Alice Johnson', 'alice@example.com'),
  ('Bob Smith', 'bob@example.com'),
  ('Charlie Brown', 'charlie@example.com'),
  ('Diana Prince', 'diana@example.com'),
  ('Eve Wilson', 'eve@example.com');

INSERT INTO test_schema.products (name, price, stock) VALUES
  ('Widget A', 29.99, 100),
  ('Widget B', 49.99, 50),
  ('Gadget X', 99.99, 25),
  ('Gadget Y', 149.99, 10),
  ('Super Device', 299.99, 5);

INSERT INTO test_schema.orders (user_id, total, status) VALUES
  (1, 79.98, 'shipped'),
  (1, 99.99, 'pending'),
  (2, 149.98, 'delivered'),
  (2, 299.99, 'processing'),
  (3, 29.99, 'cancelled'),
  (4, 449.97, 'shipped');

INSERT INTO test_schema.order_items (order_id, product_id, quantity, unit_price) VALUES
  (1, 1, 2, 29.99),
  (1, 2, 1, 49.99),
  (2, 3, 1, 99.99),
  (3, 2, 3, 49.99),
  (4, 5, 1, 299.99),
  (5, 1, 1, 29.99),
  (6, 4, 3, 149.99);

-- Add comments for testing
COMMENT ON TABLE test_schema.users IS 'User accounts for the e-commerce system';
COMMENT ON COLUMN test_schema.users.email IS 'Unique email address for login';
COMMENT ON TABLE test_schema.orders IS 'Customer orders';
`;

async function globalSetup(config: FullConfig) {
  console.log('\\n🚀 Global Setup: Preparing test environment...\\n');

  const pgConfig = {
    host: process.env.TEST_PG_HOST || 'localhost',
    port: parseInt(process.env.TEST_PG_PORT || '5432'),
    user: process.env.TEST_PG_USER || 'postgres',
    password: process.env.TEST_PG_PASSWORD || 'postgres',
  };

  const testDbName = process.env.TEST_PG_DATABASE || 'pgvoyager_test';

  // Connect to default postgres database first
  const adminClient = new Client({
    ...pgConfig,
    database: 'postgres',
  });

  try {
    await adminClient.connect();

    // Check if test database exists
    const dbCheck = await adminClient.query(
      `SELECT 1 FROM pg_database WHERE datname = $1`,
      [testDbName]
    );

    if (dbCheck.rows.length === 0) {
      console.log(`📦 Creating test database: ${testDbName}`);
      await adminClient.query(`CREATE DATABASE ${testDbName}`);
    } else {
      console.log(`✅ Test database exists: ${testDbName}`);

      // Terminate idle connections from previous test runs to prevent "too many clients" error
      console.log('🔄 Terminating idle connections from previous test runs...');
      try {
        const terminateResult = await adminClient.query(`
          SELECT pg_terminate_backend(pid)
          FROM pg_stat_activity
          WHERE datname = $1
            AND pid <> pg_backend_pid()
            AND state = 'idle'
            AND query_start < NOW() - INTERVAL '1 minute'
        `, [testDbName]);
        console.log(`✅ Terminated ${terminateResult.rowCount} idle connections`);
      } catch (err) {
        console.log('ℹ️  Could not terminate idle connections (may not have permission)');
      }
    }

    await adminClient.end();

    // Connect to test database and create schema
    const testClient = new Client({
      ...pgConfig,
      database: testDbName,
    });

    await testClient.connect();
    console.log('📝 Creating test schema and sample data...');
    await testClient.query(TEST_SCHEMA_SQL);
    console.log('✅ Test schema created successfully');
    await testClient.end();

    // Note: We don't clean PgVoyager's SQLite database here because
    // the app may be running. Tests should handle existing connections.

    console.log('\\n✅ Global setup complete!\\n');
  } catch (error) {
    console.error('❌ Global setup failed:', error);
    throw error;
  }
}

async function cleanPgVoyagerState() {
  // Get the SQLite database path
  const configDir = process.env.XDG_CONFIG_HOME ||
    path.join(process.env.HOME || '', '.config');
  const dbPath = process.env.PGVOYAGER_DB_PATH ||
    path.join(configDir, 'pgvoyager', 'pgvoyager.db');

  if (fs.existsSync(dbPath)) {
    console.log(`🧹 Cleaning PgVoyager state: ${dbPath}`);
    fs.unlinkSync(dbPath);
    console.log('✅ PgVoyager state cleaned');
  } else {
    console.log('ℹ️  No PgVoyager state to clean');
  }
}

export default globalSetup;
