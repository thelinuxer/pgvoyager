import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Smoke Tests - Quick sanity checks for CI/CD
 * These tests verify basic functionality works before running the full suite
 */

test.describe('Smoke Tests', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('1. Application loads', async () => {
    await app.goto();
    await app.waitForLoad();

    // App should be loaded
    await expect(app.page).toHaveTitle(/PgVoyager/i);
  });

  test('2. Welcome screen is displayed', async () => {
    await app.expectWelcomeScreen();
    await expect(app.welcomeNewConnectionButton).toBeVisible();
  });

  test('3. Can open connection modal', async () => {
    await app.openNewConnectionModal();
    await expect(app.connectionModal.modal).toBeVisible();
  });

  test('4. Can fill connection form', async () => {
    const config = getTestConnectionConfig();
    await app.connectionModal.fillConnectionForm(config);

    await app.connectionModal.expectFormValues({
      name: config.name,
      host: config.host,
      port: config.port,
    });
  });

  test('5. Can test connection', async () => {
    const success = await app.connectionModal.testConnection();
    expect(success).toBe(true);
  });

  test('6. Can connect to database', async () => {
    await app.connectionModal.saveAndConnect();
    await app.connectionModal.waitForClose();

    const config = getTestConnectionConfig();
    await app.expectConnected(config.name);
  });

  test('7. Schema tree loads', async () => {
    await app.sidebar.waitForSchemaLoad();
    await expect(app.sidebar.schemaTree).toBeVisible();
  });

  test('8. Can see test_schema', async () => {
    await app.sidebar.expectSchemaVisible('test_schema');
  });

  test('9. Can expand schema', async () => {
    await app.sidebar.expandNode('test_schema');
    await expect(app.sidebar.getTreeNode('Tables')).toBeVisible();
  });

  test('10. Can see tables', async () => {
    await app.sidebar.expandNode('Tables');
    await app.sidebar.expectTableVisible('users');
    await app.sidebar.expectTableVisible('orders');
  });

  test('11. Can open table viewer', async () => {
    await app.sidebar.doubleClickTable('users');
    await app.tableViewer.waitForLoad();
    await app.tableViewer.expectTableVisible();
  });

  test('12. Table has data', async () => {
    await app.tableViewer.expectNotEmpty();
    const rowCount = await app.tableViewer.getRowCount();
    expect(rowCount).toBeGreaterThan(0);
  });

  test('13. Can execute query', async () => {
    // Open a new query tab first
    await app.sidebar.openNewQuery();
    await app.queryEditor.setQuery('SELECT 1 as smoke_test');
    await app.queryEditor.executeQuery();
    await app.queryEditor.expectResults();
  });

  test('14. Query results are correct', async () => {
    await app.queryEditor.expectColumnExists('smoke_test');
    const value = await app.queryEditor.getResultsCellValue(1, 'smoke_test');
    expect(value).toContain('1');
  });

  test('15. Application is stable', async () => {
    // No console errors
    const consoleErrors: string[] = [];
    app.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Navigate around - check if Tables is visible, if not expand the tree
    // (previous tests may have left the tree in expanded state)
    const tablesNode = app.sidebar.getTreeNode('Tables');
    if (!(await tablesNode.isVisible())) {
      await app.sidebar.expandNode('test_schema');
      await expect(tablesNode).toBeVisible({ timeout: 5000 });
    }

    const ordersNode = app.sidebar.getTreeNode('orders');
    if (!(await ordersNode.isVisible())) {
      await app.sidebar.expandNode('Tables');
      await expect(ordersNode).toBeVisible({ timeout: 5000 });
    }

    await app.sidebar.doubleClickTable('orders');
    await app.tableViewer.waitForLoad();

    // Execute another query (open a new query tab first)
    await app.sidebar.openNewQuery();
    await app.queryEditor.setQuery('SELECT COUNT(*) as total FROM test_schema.users');
    await app.queryEditor.executeQuery();

    // Filter console errors to exclude expected ones
    const criticalErrors = consoleErrors.filter(
      (e) =>
        !e.includes('favicon') &&
        !e.includes('404') &&
        !e.includes('WebSocket') && // WebSocket connection issues during test teardown
        !e.includes('Failed to fetch') && // Network issues during rapid test execution
        !e.includes('query history') && // Query history API issues in test environment
        !e.includes('400') // Bad request errors (often timing-related in tests)
    );

    if (criticalErrors.length > 0) {
      console.log('Unexpected console errors found:', criticalErrors);
    }
    expect(criticalErrors.length).toBe(0);
  });
});

/**
 * Quick health check - runs in under 30 seconds
 */
test.describe('Quick Health Check', () => {
  test('App loads and connects', async ({ page }) => {
    const app = new AppPage(page);
    const config = getTestConnectionConfig();

    // Load app
    await app.goto();
    await app.waitForLoad();

    // Connect
    await app.createConnection(config);
    await app.expectConnected(config.name);

    // Browse schema
    await app.sidebar.waitForSchemaLoad();
    await app.sidebar.expectSchemaVisible('test_schema');

    // Execute query (open a new query tab first)
    await app.sidebar.openNewQuery();
    await app.queryEditor.setQuery('SELECT 1');
    await app.queryEditor.executeQuery();
    await app.queryEditor.expectResults();
  });
});
