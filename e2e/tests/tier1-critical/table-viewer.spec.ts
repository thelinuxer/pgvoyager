import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Table Viewer Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Table Viewer', () => {
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

    // Navigate to users table for most tests
    await app.sidebar.navigateToTable('test_schema', 'users');
    await app.tableViewer.waitForLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test.describe('Data Display', () => {
    test('should display table data', async () => {
      await app.tableViewer.expectTableVisible();
      await app.tableViewer.expectNotEmpty();
    });

    test('should display correct columns', async () => {
      await app.tableViewer.expectColumnExists('id');
      await app.tableViewer.expectColumnExists('name');
      await app.tableViewer.expectColumnExists('email');
      await app.tableViewer.expectColumnExists('created_at');
    });

    test('should display correct number of rows', async () => {
      // We have 5 test users
      await app.tableViewer.expectMinRowCount(5);
    });

    test('should get column names', async () => {
      const columns = await app.tableViewer.getColumnNames();
      // Column names may include type info, so use partial matching
      expect(columns.some(c => c.includes('id'))).toBe(true);
      expect(columns.some(c => c.includes('name'))).toBe(true);
      expect(columns.some(c => c.includes('email'))).toBe(true);
    });

    test('should get cell value', async () => {
      const name = await app.tableViewer.getCellValue(0, 1);
      expect(name).toBeTruthy();
    });

    test('should get row values', async () => {
      const values = await app.tableViewer.getRowValues(0);
      expect(values.length).toBeGreaterThan(0);
    });
  });

  test.describe('Pagination', () => {
    test('should display pagination controls', async () => {
      await expect(app.tableViewer.pagination).toBeVisible();
    });

    test('should display page info', async () => {
      // Page info should show current page
      const currentPage = await app.tableViewer.getCurrentPage();
      expect(currentPage).toBeGreaterThanOrEqual(1);
    });

    test('should change page size', async () => {
      // Valid page size options: 50, 100, 250, 500, 1000
      await app.tableViewer.setPageSize(50);
      const rowCount = await app.tableViewer.getRowCount();
      expect(rowCount).toBeLessThanOrEqual(50);
    });

    test('should show pagination info', async () => {
      // With 5 users and page size 50+, we should have 1 page
      const totalPages = await app.tableViewer.getTotalPages();
      expect(totalPages).toBeGreaterThanOrEqual(1);
    });
  });

  test.describe('Sorting', () => {
    test('should click column header to sort', async () => {
      // Click on name column to trigger sort
      await app.tableViewer.sortByColumn('name');
      // Data should still be visible after sort
      await app.tableViewer.expectTableVisible();
    });

    test('should click column header multiple times', async () => {
      // Click twice to toggle sort order
      await app.tableViewer.sortByColumn('id');
      await app.tableViewer.sortByColumn('id');
      // Data should still be visible
      await app.tableViewer.expectTableVisible();
    });
  });

  test.describe('Refresh', () => {
    test('should refresh data', async () => {
      await app.tableViewer.refresh();
      await app.tableViewer.expectTableVisible();
      await app.tableViewer.expectNotEmpty();
    });
  });

  test.describe('Different Tables', () => {
    test('should display products table', async () => {
      await app.sidebar.navigateToTable('test_schema', 'products');
      await app.tableViewer.waitForLoad();

      await app.tableViewer.expectColumnExists('name');
      await app.tableViewer.expectColumnExists('price');
      await app.tableViewer.expectColumnExists('stock');
    });

    test('should display orders table with FK column', async () => {
      await app.sidebar.navigateToTable('test_schema', 'orders');
      await app.tableViewer.waitForLoad();

      await app.tableViewer.expectColumnExists('user_id');
      await app.tableViewer.expectColumnExists('total');
      await app.tableViewer.expectColumnExists('status');
    });

    test('should display order_items junction table', async () => {
      await app.sidebar.navigateToTable('test_schema', 'order_items');
      await app.tableViewer.waitForLoad();

      await app.tableViewer.expectColumnExists('order_id');
      await app.tableViewer.expectColumnExists('product_id');
      await app.tableViewer.expectColumnExists('quantity');
    });
  });

  test.describe('Empty Table', () => {
    test('should handle empty result gracefully', async () => {
      // Open a new query tab for this test
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
      await app.queryEditor.setQuery(
        "SELECT * FROM test_schema.users WHERE name = 'NonexistentUser'"
      );
      await app.queryEditor.executeQuery();

      // Should show empty results
      const rowCount = await app.queryEditor.resultsTableRows.count();
      expect(rowCount).toBe(0);
    });
  });

});
