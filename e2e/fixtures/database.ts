import { Client } from 'pg';
import path from 'path';
import fs from 'fs';

/**
 * Database fixture utilities for E2E tests
 */

// Get PostgreSQL connection config from environment
export function getTestPgConfig() {
  return {
    host: process.env.TEST_PG_HOST || 'localhost',
    port: parseInt(process.env.TEST_PG_PORT || '5432'),
    user: process.env.TEST_PG_USER || 'postgres',
    password: process.env.TEST_PG_PASSWORD || 'postgres',
    database: process.env.TEST_PG_DATABASE || 'pgvoyager_test',
  };
}

/**
 * Get a PostgreSQL client connected to the test database
 */
export async function getTestDbClient(): Promise<Client> {
  const client = new Client(getTestPgConfig());
  await client.connect();
  return client;
}

/**
 * Reset test data in the test_schema (for CRUD tests)
 * This re-inserts the original sample data
 */
export async function resetTestData(): Promise<void> {
  const client = await getTestDbClient();

  try {
    // Clear existing data (in correct order due to FKs)
    await client.query('DELETE FROM test_schema.order_items');
    await client.query('DELETE FROM test_schema.orders');
    await client.query('DELETE FROM test_schema.products');
    await client.query('DELETE FROM test_schema.users');

    // Reset sequences
    await client.query('ALTER SEQUENCE test_schema.users_id_seq RESTART WITH 1');
    await client.query('ALTER SEQUENCE test_schema.products_id_seq RESTART WITH 1');
    await client.query('ALTER SEQUENCE test_schema.orders_id_seq RESTART WITH 1');
    await client.query('ALTER SEQUENCE test_schema.order_items_id_seq RESTART WITH 1');

    // Re-insert test data
    await client.query(`
      INSERT INTO test_schema.users (name, email) VALUES
        ('Alice Johnson', 'alice@example.com'),
        ('Bob Smith', 'bob@example.com'),
        ('Charlie Brown', 'charlie@example.com'),
        ('Diana Prince', 'diana@example.com'),
        ('Eve Wilson', 'eve@example.com')
    `);

    await client.query(`
      INSERT INTO test_schema.products (name, price, stock) VALUES
        ('Widget A', 29.99, 100),
        ('Widget B', 49.99, 50),
        ('Gadget X', 99.99, 25),
        ('Gadget Y', 149.99, 10),
        ('Super Device', 299.99, 5)
    `);

    await client.query(`
      INSERT INTO test_schema.orders (user_id, total, status) VALUES
        (1, 79.98, 'shipped'),
        (1, 99.99, 'pending'),
        (2, 149.98, 'delivered'),
        (2, 299.99, 'processing'),
        (3, 29.99, 'cancelled'),
        (4, 449.97, 'shipped')
    `);

    await client.query(`
      INSERT INTO test_schema.order_items (order_id, product_id, quantity, unit_price) VALUES
        (1, 1, 2, 29.99),
        (1, 2, 1, 49.99),
        (2, 3, 1, 99.99),
        (3, 2, 3, 49.99),
        (4, 5, 1, 299.99),
        (5, 1, 1, 29.99),
        (6, 4, 3, 149.99)
    `);

    console.log('✅ Test data reset successfully');
  } finally {
    await client.end();
  }
}

/**
 * Clean PgVoyager's SQLite database (connections, history, preferences)
 * This should be called before each test file to ensure clean state
 */
export function cleanPgVoyagerState(): void {
  const configDir = process.env.XDG_CONFIG_HOME ||
    path.join(process.env.HOME || '', '.config');
  const dbPath = process.env.PGVOYAGER_DB_PATH ||
    path.join(configDir, 'pgvoyager', 'pgvoyager.db');

  if (fs.existsSync(dbPath)) {
    fs.unlinkSync(dbPath);
    console.log('🧹 PgVoyager SQLite state cleaned');
  }
}

/**
 * Get the test connection configuration for PgVoyager
 */
export function getTestConnectionConfig() {
  const pgConfig = getTestPgConfig();
  return {
    name: 'E2E Test Connection',
    host: pgConfig.host,
    port: pgConfig.port,
    database: pgConfig.database,
    username: pgConfig.user,
    password: pgConfig.password,
    sslMode: 'disable' as const,
  };
}

/**
 * Verify the test database is accessible and has the expected schema
 */
export async function verifyTestDatabase(): Promise<boolean> {
  try {
    const client = await getTestDbClient();

    // Check test_schema exists
    const schemaCheck = await client.query(`
      SELECT schema_name FROM information_schema.schemata
      WHERE schema_name = 'test_schema'
    `);

    if (schemaCheck.rows.length === 0) {
      console.error('❌ test_schema not found');
      await client.end();
      return false;
    }

    // Check tables exist
    const tableCheck = await client.query(`
      SELECT table_name FROM information_schema.tables
      WHERE table_schema = 'test_schema'
      ORDER BY table_name
    `);

    const expectedTables = ['order_items', 'orders', 'products', 'users'];
    const actualTables = tableCheck.rows.map((r) => r.table_name);

    for (const table of expectedTables) {
      if (!actualTables.includes(table)) {
        console.error(`❌ Table test_schema.${table} not found`);
        await client.end();
        return false;
      }
    }

    await client.end();
    return true;
  } catch (error) {
    console.error('❌ Failed to verify test database:', error);
    return false;
  }
}
