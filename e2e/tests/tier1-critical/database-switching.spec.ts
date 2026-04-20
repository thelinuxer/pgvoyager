import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Database panel tests — server-level connections allow listing, switching,
 * creating, and dropping databases without creating a separate connection per db.
 *
 * Assumes two databases exist on the test server: `postgres` and the test
 * database from TEST_PG_DATABASE (default: pgvoyager_test).
 */
test.describe('Database Panel — switching', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();

    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('panel marks the current database as active', async () => {
    const config = getTestConnectionConfig();
    await app.sidebar.expectCurrentDatabase(config.database);
  });

  test('panel lists databases from the server', async () => {
    await expect(app.sidebar.databaseOption('postgres')).toBeVisible({ timeout: 10000 });

    const config = getTestConnectionConfig();
    await expect(app.sidebar.databaseOption(config.database)).toBeVisible();
  });

  test('switching to postgres reloads the schema tree', async () => {
    const config = getTestConnectionConfig();

    await app.sidebar.switchToDatabase('postgres');
    await app.sidebar.expectCurrentDatabase('postgres');
    await app.sidebar.waitForSchemaLoad();

    // test_schema only exists in pgvoyager_test
    await expect(app.sidebar.getSchemaNode('test_schema')).not.toBeVisible({ timeout: 5000 });

    await app.sidebar.switchToDatabase(config.database);
    await app.sidebar.expectCurrentDatabase(config.database);
    await app.sidebar.waitForSchemaLoad();

    await app.sidebar.expectSchemaVisible('test_schema');
  });
});

test.describe('Database Panel — create and drop', () => {
  test.describe.configure({ mode: 'serial' });

  // Unique name per run so re-runs don't collide with leftover state.
  const throwawayDb = `e2e_tmpdb_${Date.now()}`;
  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();

    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('creates a new database from the panel', async () => {
    await app.sidebar.createDatabase(throwawayDb);
    await expect(app.sidebar.databaseOption(throwawayDb)).toBeVisible();
  });

  test('drops a database via the context menu', async () => {
    await app.sidebar.dropDatabase(throwawayDb);
    await expect(app.sidebar.databaseOption(throwawayDb)).not.toBeVisible();
  });

  test('dropping the currently-selected database auto-switches to postgres', async () => {
    const config = getTestConnectionConfig();
    const dbToDrop = `e2e_current_drop_${Date.now()}`;

    await app.sidebar.createDatabase(dbToDrop);
    await app.sidebar.switchToDatabase(dbToDrop);
    await app.sidebar.expectCurrentDatabase(dbToDrop);

    await app.sidebar.dropDatabase(dbToDrop);

    // Backend auto-switches to `postgres` since we dropped the current db.
    await app.sidebar.expectCurrentDatabase('postgres');

    // Restore original selection for subsequent tests / fixtures.
    await app.sidebar.switchToDatabase(config.database);
    await app.sidebar.waitForSchemaLoad();
  });
});

test.describe('Connection without explicit database', () => {
  test('creates a connection with empty database and defaults to postgres', async ({ page }) => {
    const app = new AppPage(page);
    await app.goto();

    const config = getTestConnectionConfig();

    await app.openNewConnectionModal();
    await app.connectionModal.fillConnectionForm({
      ...config,
      name: 'No-Default-DB Connection',
      database: '',
    });
    await app.connectionModal.saveAndConnect();
    await app.connectionModal.waitForClose();

    await app.expectConnected('No-Default-DB Connection');
    await app.sidebar.waitForSchemaLoad();

    await app.sidebar.expectCurrentDatabase('postgres');
  });
});
