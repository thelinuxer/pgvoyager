import { testWithCleanState as test, expect, getTestConnectionConfig, getTestDbClient } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Schema DDL Tests
 * Tests create schema, create table, add constraint, and drop schema operations.
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion.
 */
test.describe('Schema DDL Operations', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  const TEST_SCHEMA = 'e2e_test_schema';
  const DROP_SCHEMA = 'e2e_drop_schema';
  const TEST_TABLE = 'e2e_test_table';

  test.beforeAll(async ({ browser }) => {
    // Clean up any leftover schemas from previous runs
    const client = await getTestDbClient();
    try {
      await client.query(`DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
      await client.query(`DROP SCHEMA IF EXISTS ${DROP_SCHEMA} CASCADE`);
    } finally {
      await client.end();
    }

    const page = await browser.newPage();
    app = new AppPage(page);

    await app.goto();
    await app.waitForLoad();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    // Clean up schemas
    const client = await getTestDbClient();
    try {
      await client.query(`DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
      await client.query(`DROP SCHEMA IF EXISTS ${DROP_SCHEMA} CASCADE`);
    } finally {
      await client.end();
    }

    await app.page.close();
  });

  test.describe('Create Schema', () => {
    test('should create a new schema via modal', async () => {
      await app.sidebar.createSchema(TEST_SCHEMA);

      // Wait for schema refresh and verify schema appears
      await app.sidebar.waitForSchemaLoad();
      await app.sidebar.expectSchemaVisible(TEST_SCHEMA);
    });

    test('should create a second schema for drop testing', async () => {
      await app.sidebar.createSchema(DROP_SCHEMA);
      await app.sidebar.waitForSchemaLoad();
      await app.sidebar.expectSchemaVisible(DROP_SCHEMA);
    });
  });

  test.describe('Create Table', () => {
    test('should create a table in the new schema via context menu', async () => {
      await app.sidebar.createTableViaContextMenu(TEST_SCHEMA, TEST_TABLE, [
        { name: 'id', type: 'SERIAL', pk: true },
        { name: 'name', type: 'TEXT' },
        { name: 'value', type: 'INTEGER' }
      ]);

      // Refresh and verify table appears under the schema
      await app.sidebar.refreshSchema();
      await app.sidebar.expandNode(TEST_SCHEMA);

      // The table should be under a "Tables" folder
      await app.sidebar.expandNode('Tables');
      await app.sidebar.expectTableVisible(TEST_TABLE);
    });
  });

  test.describe('Add Constraint', () => {
    test('should add a UNIQUE constraint via context menu', async () => {
      // Make sure the table node is visible
      await app.sidebar.expandNode(TEST_SCHEMA);
      await app.sidebar.expandNode('Tables');

      await app.sidebar.addConstraintViaContextMenu(TEST_TABLE, {
        type: 'unique',
        name: 'uq_test_name',
        columns: 'name'
      });

      // Verify constraint exists via query
      await app.sidebar.openNewQuery();
      await app.page.waitForTimeout(500);
      await app.queryEditor.setQuery(`
        SELECT conname FROM pg_constraint
        WHERE conrelid = '${TEST_SCHEMA}.${TEST_TABLE}'::regclass
        AND contype = 'u';
      `);
      await app.queryEditor.executeQuery();
      await app.queryEditor.waitForQueryExecution();

      // Should find our constraint in the query results table
      const resultTable = app.page.locator('table');
      await expect(resultTable.last()).toContainText('uq_test_name', { timeout: 10000 });
    });
  });

  test.describe('Drop Schema', () => {
    test('should drop a schema via context menu with CASCADE', async () => {
      // The drop schema should still be visible
      await app.sidebar.expectSchemaVisible(DROP_SCHEMA);

      await app.sidebar.dropSchemaViaContextMenu(DROP_SCHEMA, true);

      // Wait for schema tree to refresh
      await app.sidebar.waitForSchemaLoad();

      // Verify schema is removed from sidebar
      await expect(app.sidebar.getSchemaNode(DROP_SCHEMA)).not.toBeVisible({ timeout: 5000 });
    });
  });
});
