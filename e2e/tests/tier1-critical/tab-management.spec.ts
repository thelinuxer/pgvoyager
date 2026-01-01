import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Tab Management Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Tab Management', () => {
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

  // Helper to ensure schema tree is expanded
  async function ensureTablesExpanded() {
    const tablesNode = app.sidebar.getTreeNode('Tables');
    if (!(await tablesNode.isVisible())) {
      await app.sidebar.expandNode('test_schema');
      await app.sidebar.expandNode('Tables');
    }
  }

  test.describe('Opening Tabs', () => {
    test('should open table in new tab', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');

      await app.tabBar.expectTabExists('users');
    });

    test('should open multiple tables in tabs when pinned', async () => {
      await ensureTablesExpanded();

      // Open first table and pin it to prevent replacement
      await app.sidebar.doubleClickTable('users');
      await app.page.waitForTimeout(300);
      await app.tabBar.expectTabExists('users');
      await app.tabBar.pinTab('users');

      // Now open second table - since first is pinned, a new tab should open
      await app.sidebar.doubleClickTable('orders');
      await app.page.waitForTimeout(300);
      await app.tabBar.expectTabExists('orders');

      // Should have at least 2 tabs (pinned users + orders)
      const count = await app.tabBar.getTabCount();
      expect(count).toBeGreaterThanOrEqual(2);

      // Cleanup: unpin the users tab for other tests
      await app.tabBar.unpinTab('users');
    });

    test('should not duplicate tab when opening same table', async () => {
      await ensureTablesExpanded();

      await app.sidebar.doubleClickTable('users');
      const countBefore = await app.tabBar.getTabCount();

      await app.sidebar.doubleClickTable('users');
      const countAfter = await app.tabBar.getTabCount();

      expect(countAfter).toBe(countBefore);
    });

    test('should open view in tab', async () => {
      // Ensure tree is expanded
      const viewsNode = app.sidebar.getTreeNode('Views');
      if (!(await viewsNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await app.sidebar.expandNode('Views');

      await app.sidebar.getViewNode('user_order_summary').dblclick();
      await app.tabBar.expectTabExists('user_order_summary');
    });
  });

  test.describe('Closing Tabs', () => {
    test('should close tab with close button', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');

      await app.tabBar.expectTabExists('users');
      await app.tabBar.closeTab('users');

      await app.tabBar.expectTabNotExists('users');
    });

    test('should close tab with middle click', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');

      await app.tabBar.expectTabExists('users');
      await app.tabBar.closeTabByMiddleClick('users');

      await app.tabBar.expectTabNotExists('users');
    });

    test('should close all tabs', async () => {
      await ensureTablesExpanded();

      await app.sidebar.doubleClickTable('users');
      await app.sidebar.doubleClickTable('orders');

      await app.tabBar.closeAllTabs();
      await app.tabBar.expectNoTabs();
    });

  });

  test.describe('Switching Tabs', () => {
    test('should switch to tab by clicking', async () => {
      await ensureTablesExpanded();

      // Open and pin first tab to prevent replacement
      await app.sidebar.doubleClickTable('users');
      await app.tabBar.pinTab('users');

      await app.sidebar.doubleClickTable('orders');

      // Orders tab should be active
      await app.tabBar.expectTabActive('orders');

      // Click users tab
      await app.tabBar.clickTab('users');
      await app.tabBar.expectTabActive('users');

      // Cleanup
      await app.tabBar.unpinTab('users');
    });

    test('should show correct content when switching tabs', async () => {
      await ensureTablesExpanded();

      // Open and pin first tab to prevent replacement
      await app.sidebar.doubleClickTable('users');
      await app.tabBar.pinTab('users');
      await app.tableViewer.waitForLoad();

      await app.sidebar.doubleClickTable('products');
      await app.tableViewer.waitForLoad();

      // Switch back to users
      await app.tabBar.clickTab('users');
      await app.tableViewer.waitForLoad();

      // Should see users data
      await app.tableViewer.expectColumnExists('email');

      // Cleanup: unpin users tab
      await app.tabBar.unpinTab('users');
    });

    test('should get active tab title', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');

      const title = await app.tabBar.getActiveTabTitle();
      expect(title).toContain('users');
    });
  });

  test.describe('Tab Titles', () => {
    test('should get active tab title', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');

      const title = await app.tabBar.getActiveTabTitle();
      expect(title).toContain('users');
    });

    test('should expect active tab title', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('orders');

      await app.tabBar.expectActiveTabTitle('orders');
    });
  });

  test.describe('Tab by Index', () => {
    test('should access tab by index', async () => {
      await ensureTablesExpanded();

      await app.sidebar.doubleClickTable('users');
      await app.sidebar.doubleClickTable('orders');

      const firstTab = app.tabBar.getTabByIndex(0);
      await expect(firstTab).toBeVisible();
    });
  });

  test.describe('Pin Tabs', () => {
    test('should pin tab by double-click', async () => {
      await ensureTablesExpanded();
      await app.sidebar.doubleClickTable('users');
      await app.tabBar.expectTabExists('users');

      // Ensure the tab is active before attempting to pin
      await app.tabBar.clickTab('users');
      await app.tabBar.expectTabActive('users');

      // Ensure tab is NOT pinned first (cleanup from any previous test)
      const pinIndicator = app.tabBar.getTabPinIndicator('users');
      if (await pinIndicator.isVisible()) {
        await app.tabBar.unpinTab('users');
        await app.page.waitForTimeout(300);
      }

      // Double-click the tab to pin it
      await app.tabBar.pinTab('users');
      await app.page.waitForTimeout(500); // Wait for pin state to update

      await app.tabBar.expectTabPinned('users');
    });

    test('should unpin pinned tab', async () => {
      // The users tab may already be pinned from the previous test
      // We need to ensure it's pinned before testing unpin
      await app.tabBar.expectTabExists('users');
      await app.tabBar.clickTab('users');
      await app.tabBar.expectTabActive('users');

      // Tab should be pinned from the previous test
      await app.tabBar.expectTabPinned('users');

      // Now unpin it
      await app.tabBar.unpinTab('users');
      await app.page.waitForTimeout(500);
      await app.tabBar.expectTabNotPinned('users');
    });
  });
});
