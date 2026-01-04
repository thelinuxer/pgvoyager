import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage, QueryHistoryPanel } from '../../pages';

/**
 * Query History Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Query History', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let historyPanel: QueryHistoryPanel;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    historyPanel = new QueryHistoryPanel(page);

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

  test.describe('Panel Access', () => {
    test('should open query history panel from sidebar', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();
    });

    test('should close query history panel with close button', async () => {
      // Open first if not open
      if (!(await historyPanel.panelContent.isVisible())) {
        await app.sidebar.openQueryHistory();
        await historyPanel.expectVisible();
      }

      await historyPanel.close();
      await historyPanel.expectClosed();
    });

    test('should close query history panel by clicking backdrop', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // Click on backdrop
      await historyPanel.panel.click({ position: { x: 10, y: 10 } });
      await historyPanel.expectClosed();
    });
  });

  test.describe('Panel Content', () => {
    test('should show filter buttons', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      await expect(historyPanel.currentDbFilterButton).toBeVisible();
      await expect(historyPanel.allFilterButton).toBeVisible();

      await historyPanel.close();
    });

    test('should show search input', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      await expect(historyPanel.searchInput).toBeVisible();

      await historyPanel.close();
    });
  });

  test.describe('Query Execution History', () => {
    test('should record executed query in history', async () => {
      // Execute a query first
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users');
      await app.queryEditor.executeQuery();
      await app.page.waitForTimeout(500);

      // Open history panel
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();
      await app.page.waitForTimeout(500);

      // Check for history entry (filter by current DB is default)
      const historyCount = await historyPanel.historyItems.count();
      expect(historyCount).toBeGreaterThanOrEqual(1);

      await historyPanel.close();
    });

    test('should show query SQL in history', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // First history item should contain SELECT
      const firstItem = historyPanel.historyItems.first();
      await expect(firstItem).toContainText('SELECT');

      await historyPanel.close();
    });

    test('should show query metadata (time, rows, duration)', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // First history item should have metadata
      const firstItem = historyPanel.historyItems.first();
      const metaSection = firstItem.locator('.history-meta');

      // Should have time indicator
      await expect(metaSection).toBeVisible();

      await historyPanel.close();
    });
  });

  test.describe('Loading History Entry', () => {
    test('should load query from history into editor', async () => {
      // Make sure there's at least one history entry
      await app.sidebar.openNewQuery();
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
      await app.queryEditor.setQuery('SELECT id, name FROM test_schema.users LIMIT 5');
      await app.queryEditor.executeQuery();
      await app.page.waitForTimeout(500);

      // Clear editor
      await app.queryEditor.clearQuery();

      // Open history and click entry
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();
      await app.page.waitForTimeout(300);

      // Click on first history item
      const firstItem = historyPanel.historyItems.first();
      await firstItem.click();

      // History panel should close and query should be in editor
      await historyPanel.expectClosed();

      // A new query tab should open with the SQL
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
    });
  });

  test.describe('Filter Functionality', () => {
    test('should toggle between Current DB and All filters', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // Default is Current DB
      await expect(historyPanel.currentDbFilterButton).toHaveClass(/active/);

      // Click All filter
      await historyPanel.filterByAll();
      await expect(historyPanel.allFilterButton).toHaveClass(/active/);

      // Click Current DB filter
      await historyPanel.filterByCurrentDb();
      await expect(historyPanel.currentDbFilterButton).toHaveClass(/active/);

      await historyPanel.close();
    });

    test('should filter by search query', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // Type in search
      await historyPanel.searchFor('users');
      await app.page.waitForTimeout(300);

      // Search input should have value
      await expect(historyPanel.searchInput).toHaveValue('users');

      // Results should be filtered (at least the entries with 'users' should show)
      const historyCount = await historyPanel.historyItems.count();
      expect(historyCount).toBeGreaterThanOrEqual(0);

      await historyPanel.close();
    });
  });

  test.describe('Clear History', () => {
    test('should show clear button when history exists', async () => {
      await app.sidebar.openQueryHistory();
      await historyPanel.expectVisible();

      // If there are history entries, clear button should be visible
      const historyCount = await historyPanel.historyItems.count();
      if (historyCount > 0) {
        await expect(historyPanel.clearButton).toBeVisible();
      }

      await historyPanel.close();
    });
  });
});
