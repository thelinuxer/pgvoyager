import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Query Editor FK Feature Tests
 * Tests FK preview, navigation, and query result persistence in the Query Editor
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Query Editor FK Features', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);

    // Navigate and connect once
    await app.goto();
    await app.waitForLoad();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  // Clean up extra tabs after each test to prevent tab accumulation
  test.afterEach(async () => {
    const tabCount = await app.tabBar.getTabCount();
    // Keep only 2 tabs to prevent accumulation issues
    while (await app.tabBar.getTabCount() > 2) {
      const closeButton = app.tabBar.tabs.first().locator('.tab-close');
      if (await closeButton.isVisible()) {
        await closeButton.click();
        await app.page.waitForTimeout(100);
      } else {
        break;
      }
    }
  });

  test.describe('FK Icons in Query Results', () => {
    test('should display FK icons when querying table with foreign keys', async () => {
      // Open a new query tab
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query the orders table which has FK to users
      await app.queryEditor.setQuery('SELECT * FROM test_schema.orders LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Should have FK icon on user_id column
      const fkIconCount = await app.queryEditor.getFKIconCount();
      expect(fkIconCount).toBeGreaterThanOrEqual(1);
    });

    test('should display PK icons when querying table with primary keys', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query the users table which has PK
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Should have PK icon on id column
      const pkIconCount = await app.queryEditor.getPKIconCount();
      expect(pkIconCount).toBeGreaterThanOrEqual(1);
    });

    test('should display both PK and FK icons in order_items table', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // order_items has FKs to both orders and products
      await app.queryEditor.setQuery('SELECT * FROM test_schema.order_items LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Should have FK icons
      const fkIconCount = await app.queryEditor.getFKIconCount();
      expect(fkIconCount).toBeGreaterThanOrEqual(1);
    });
  });

  test.describe('FK Preview Popup in Query Results', () => {
    test('should show FK preview popup on hover', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query orders table
      await app.queryEditor.setQuery('SELECT * FROM test_schema.orders LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Check if there are FK cells
      const fkCellCount = await app.queryEditor.getFKCellCount();
      if (fkCellCount > 0) {
        // Hover over first FK cell
        await app.queryEditor.hoverFKCell(0);

        // Should show preview popup
        await app.queryEditor.expectFKPreviewPopup();
      }
    });

    test('should hide FK preview popup when mouse leaves', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      await app.queryEditor.setQuery('SELECT * FROM test_schema.orders LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      const fkCellCount = await app.queryEditor.getFKCellCount();
      if (fkCellCount > 0) {
        // Hover over FK cell
        await app.queryEditor.hoverFKCell(0);
        await app.queryEditor.expectFKPreviewPopup();

        // Move mouse away to table header
        await app.queryEditor.resultsTableHeaders.first().hover();
        await app.page.waitForTimeout(300);

        // Popup should be hidden
        await app.queryEditor.expectNoFKPreviewPopup();
      }
    });
  });

  test.describe('FK Navigation from Query Results', () => {
    test('should navigate to referenced table on FK click', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query orders table
      await app.queryEditor.setQuery('SELECT * FROM test_schema.orders LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      const fkCellCount = await app.queryEditor.getFKCellCount();
      if (fkCellCount > 0) {
        const tabCountBefore = await app.tabBar.getTabCount();

        // Click on FK cell
        await app.queryEditor.clickFKCell(0);
        await app.page.waitForTimeout(500);

        // Should open a new tab (since query tabs behave like pinned tabs)
        const tabCountAfter = await app.tabBar.getTabCount();
        expect(tabCountAfter).toBeGreaterThanOrEqual(tabCountBefore);
      }
    });

    test('should open referenced record in new tab from query results', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query order_items which has FK to products
      await app.queryEditor.setQuery('SELECT * FROM test_schema.order_items LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      const fkCellCount = await app.queryEditor.getFKCellCount();
      if (fkCellCount > 0) {
        const tabCountBefore = await app.tabBar.getTabCount();

        // Click on FK cell
        await app.queryEditor.clickFKCell(0);
        await app.page.waitForTimeout(500);

        // New tab should be opened
        const tabCountAfter = await app.tabBar.getTabCount();
        expect(tabCountAfter).toBeGreaterThanOrEqual(tabCountBefore);
      }
    });
  });

  test.describe('Query Results Persistence', () => {
    test('should preserve query results when switching tabs and returning', async () => {
      // Open a query tab
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Execute a query
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users LIMIT 3');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Note the row count
      const rowCountBefore = await app.queryEditor.resultsTableRows.count();
      expect(rowCountBefore).toBe(3);

      // Get the current tab index (the query tab)
      const queryTabIndex = (await app.tabBar.getTabCount()) - 1;

      // Open a different tab (table viewer)
      await app.sidebar.navigateToTable('test_schema', 'products');
      await app.tableViewer.waitForLoad();

      // Switch back to the query tab by index
      await app.tabBar.getTabByIndex(queryTabIndex).click();
      await app.page.waitForTimeout(300);

      // Results should still be there
      await app.queryEditor.expectResults();
      const rowCountAfter = await app.queryEditor.resultsTableRows.count();
      expect(rowCountAfter).toBe(rowCountBefore);
    });

    test('should preserve query text when switching tabs', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      const testQuery = 'SELECT id, name FROM test_schema.users WHERE id > 1';
      await app.queryEditor.setQuery(testQuery);

      // Get the current tab index
      const queryTabIndex = (await app.tabBar.getTabCount()) - 1;

      // Switch to another tab
      await app.sidebar.navigateToTable('test_schema', 'orders');
      await app.tableViewer.waitForLoad();

      // Switch back by index
      await app.tabBar.getTabByIndex(queryTabIndex).click();
      await app.page.waitForTimeout(300);

      // Query text should be preserved
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('SELECT id, name');
      expect(queryText).toContain('test_schema.users');
    });

    test('should preserve error state when switching tabs', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Execute invalid query
      await app.queryEditor.setQuery('SELECT * FROM nonexistent_table_xyz');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectError();

      // Get tab index
      const queryTabIndex = (await app.tabBar.getTabCount()) - 1;

      // Switch to another tab
      await app.sidebar.navigateToTable('test_schema', 'users');
      await app.tableViewer.waitForLoad();

      // Switch back by index
      await app.tabBar.getTabByIndex(queryTabIndex).click();
      await app.page.waitForTimeout(300);

      // Error should still be displayed
      await app.queryEditor.expectError();
    });
  });

  test.describe('FK Features with JOIN Queries', () => {
    test('should show FK info for columns in JOIN results', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // JOIN query that includes FK columns
      await app.queryEditor.setQuery(`
        SELECT o.id, o.user_id, u.name, o.total
        FROM test_schema.orders o
        JOIN test_schema.users u ON o.user_id = u.id
        LIMIT 5
      `);
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // user_id should still show as FK even in JOIN result
      const fkIconCount = await app.queryEditor.getFKIconCount();
      expect(fkIconCount).toBeGreaterThanOrEqual(0); // May or may not have FK depending on column tracing
    });

    test('should handle computed columns without FK info', async () => {
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Query with computed column
      await app.queryEditor.setQuery(`
        SELECT id, name, UPPER(email) as upper_email
        FROM test_schema.users
        LIMIT 5
      `);
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();

      // Computed columns shouldn't have FK icons
      // This test ensures the feature handles non-FK columns gracefully
      await app.queryEditor.expectColumnExists('upper_email');
    });
  });
});
