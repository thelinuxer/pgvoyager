import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Database Switching Tests
 *
 * Phase B of the server-level connections refactor: a connection targets a server;
 * the user can browse and switch databases without creating a separate connection.
 *
 * Assumes two databases exist on the test server: `postgres` (always present) and
 * the test database from TEST_PG_DATABASE (default: pgvoyager_test).
 */
test.describe('Database Switching', () => {
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

  test('database switcher shows current database', async () => {
    const config = getTestConnectionConfig();
    await app.sidebar.expectCurrentDatabase(config.database);
  });

  test('database switcher lists databases from server', async () => {
    await app.sidebar.openDatabaseSwitcher();

    // At minimum, the maintenance `postgres` db is always available
    await expect(app.sidebar.databaseOption('postgres')).toBeVisible({ timeout: 10000 });

    const config = getTestConnectionConfig();
    await expect(app.sidebar.databaseOption(config.database)).toBeVisible();

    // Close the menu
    await app.page.keyboard.press('Escape');
    await expect(app.sidebar.databaseSwitcherMenu).not.toBeVisible();
  });

  test('switching to postgres reloads the schema tree', async () => {
    const config = getTestConnectionConfig();

    // Switch to postgres
    await app.sidebar.switchToDatabase('postgres');
    await app.sidebar.expectCurrentDatabase('postgres');

    // The schema tree should reload — waitForSchemaLoad ensures the reload happened
    await app.sidebar.waitForSchemaLoad();

    // test_schema should NOT be visible in the postgres db (it only exists in pgvoyager_test)
    await expect(app.sidebar.getSchemaNode('test_schema')).not.toBeVisible({ timeout: 5000 });

    // Switch back
    await app.sidebar.switchToDatabase(config.database);
    await app.sidebar.expectCurrentDatabase(config.database);
    await app.sidebar.waitForSchemaLoad();

    // test_schema is visible again after switching back
    await app.sidebar.expectSchemaVisible('test_schema');
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
      database: '', // explicitly blank
    });
    await app.connectionModal.saveAndConnect();
    await app.connectionModal.waitForClose();

    await app.expectConnected('No-Default-DB Connection');
    await app.sidebar.waitForSchemaLoad();

    // Backend should have defaulted to `postgres`
    await app.sidebar.expectCurrentDatabase('postgres');
  });
});
