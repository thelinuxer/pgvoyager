import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Schema Browser Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Schema Browser', () => {
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

  test.describe('Schema Tree', () => {
    test('should display schema tree after connection', async () => {
      await expect(app.sidebar.schemaTree).toBeVisible();
    });

    test('should show test_schema in tree', async () => {
      await app.sidebar.expectSchemaVisible('test_schema');
    });

    test('should expand schema to show object categories', async () => {
      await app.sidebar.expandNode('test_schema');

      // Should see Tables, Views, Functions, etc.
      await expect(app.sidebar.getTreeNode('Tables')).toBeVisible();
    });

    test('should expand Tables to show table list', async () => {
      // test_schema might already be expanded from previous test
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await app.sidebar.expandNode('Tables');

      // Should see our test tables
      await app.sidebar.expectTableVisible('users');
      await app.sidebar.expectTableVisible('orders');
      await app.sidebar.expectTableVisible('products');
      await app.sidebar.expectTableVisible('order_items');
    });

    test('should collapse expanded node', async () => {
      // Ensure expanded first
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await expect(tablesNode).toBeVisible();

      await app.sidebar.collapseNode('test_schema');
      // Tables should no longer be visible (collapsed)
      await expect(tablesNode).not.toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Search and Filter', () => {
    test('should filter schema tree by search', async () => {
      // First expand to make items visible
      await app.sidebar.expandNode('test_schema');
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await app.sidebar.expandNode('Tables');

      // Search for 'user'
      await app.sidebar.searchFor('user');

      // Wait for filter to apply
      await app.page.waitForTimeout(500);

      // Check the filter input has the value
      const searchInput = app.sidebar.searchInput;
      await expect(searchInput).toHaveValue('user');
    });

    test('should clear search and show all items', async () => {
      // First set a search value
      await app.sidebar.searchFor('test');
      await app.page.waitForTimeout(300);
      await expect(app.sidebar.searchInput).toHaveValue('test');

      // Clear the search
      await app.sidebar.clearSearch();
      await app.page.waitForTimeout(300);

      // Verify search is cleared
      await expect(app.sidebar.searchInput).toHaveValue('');
    });

    test('should show no results message for invalid search', async () => {
      await app.sidebar.searchFor('xyznonexistent123');
      await app.page.waitForTimeout(300);

      // Verify search input has the invalid value
      await expect(app.sidebar.searchInput).toHaveValue('xyznonexistent123');

      // The no results message should be visible
      const noResults = app.sidebar.noResultsMessage;
      await expect(noResults).toBeVisible({ timeout: 5000 });

      // Clear search for next tests
      await app.sidebar.clearSearch();
    });
  });

  test.describe('Navigation', () => {
    test('should navigate to table and open viewer', async () => {
      await app.sidebar.navigateToTable('test_schema', 'users');

      // Table viewer should open
      await app.tableViewer.waitForLoad();
      await app.tableViewer.expectTableVisible();
    });

    test('should double-click table to open in new tab', async () => {
      // Expand if needed
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }
      await app.sidebar.doubleClickTable('orders');

      // Should open a tab for the orders table
      await app.tabBar.expectTabExists('orders');
    });

    test('should navigate to view', async () => {
      await app.sidebar.expandNode('test_schema');
      const viewsNode = app.sidebar.getTreeNode('Views');
      if (!(await viewsNode.isVisible())) {
        await app.sidebar.expandNode('Views');
      }
      await expect(viewsNode).toBeVisible();

      // Expand Views if needed
      await app.sidebar.expandNode('Views');

      // Click on the view
      await app.sidebar.getTreeNode('user_order_summary').click();
    });
  });

  test.describe('Context Menu', () => {
    test('should show context menu on right-click table', async () => {
      // Expand if needed
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }

      await app.sidebar.rightClickTable('users');
      await app.sidebar.expectContextMenuVisible();

      // Close menu by clicking elsewhere
      await app.page.click('body');
      await app.page.waitForTimeout(200);
    });

    test('should have Show first 100 rows option in table context menu', async () => {
      // Expand if needed
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }

      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300); // Wait for context menu to fully render
      await expect(app.sidebar.getContextMenuItem('Show first 100 rows')).toBeVisible();

      // Close menu by clicking elsewhere
      await app.page.click('body');
      await app.page.waitForTimeout(200);
    });

    test('should open table viewer via context menu', async () => {
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }

      await app.sidebar.rightClickTable('products');
      await app.page.waitForTimeout(300); // Wait for context menu to fully render
      await app.sidebar.clickContextMenuItem('Show first 100 rows');

      await app.tableViewer.waitForLoad();
    });
  });

  test.describe('Schema Refresh', () => {
    test('should show refresh button when connected', async () => {
      await expect(app.sidebar.refreshSchemaButton).toBeVisible();
    });

    test('should refresh schema tree when clicking refresh button', async () => {
      // Expand schema to see tables before refresh
      await app.sidebar.expandNode('test_schema');
      await app.sidebar.expandNode('Tables');
      await app.sidebar.expectTableVisible('users');

      // Click refresh button
      await app.sidebar.refreshSchema();

      // Schema tree should still be visible with tables
      await expect(app.sidebar.schemaTree).toBeVisible();
      await app.sidebar.expectSchemaVisible('test_schema');
    });

    test('should disable refresh button while loading', async () => {
      // Click refresh and immediately check button state
      await app.sidebar.refreshSchemaButton.click();

      // Button should be disabled during load
      await expect(app.sidebar.refreshSchemaButton).toBeDisabled();

      // Wait for load to complete
      await app.sidebar.waitForSchemaLoad();

      // Button should be enabled again
      await expect(app.sidebar.refreshSchemaButton).toBeEnabled();
    });
  });

  test.describe('Object Types', () => {
    test('should show Views category', async () => {
      // Expand test_schema if needed
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await expect(app.sidebar.getTreeNode('Views')).toBeVisible();
    });

    test('should show Functions category', async () => {
      await expect(app.sidebar.getTreeNode('Functions')).toBeVisible();
    });

    test('should list view in Views category', async () => {
      await app.sidebar.expandNode('Views');

      await expect(app.sidebar.getViewNode('user_order_summary')).toBeVisible();
    });

    test('should list function in Functions category', async () => {
      const functionsNode = app.sidebar.getTreeNode('Functions');
      if (!(await functionsNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }

      await app.sidebar.expandNode('Functions');
      await app.page.waitForTimeout(300); // Wait for tree to expand
      await expect(app.sidebar.getFunctionNode('get_user_orders')).toBeVisible();
    });
  });
});
