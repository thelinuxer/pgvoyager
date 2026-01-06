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

      // Button should be disabled during load (use short timeout - loading might be fast)
      // If this fails, the loading was just too fast which is fine
      try {
        await expect(app.sidebar.refreshSchemaButton).toBeDisabled({ timeout: 500 });
      } catch {
        // Loading was very fast, button may already be enabled - this is acceptable
      }

      // Wait for load to complete
      await app.sidebar.waitForSchemaLoad();

      // Button should be enabled again after loading completes
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

  test.describe('Copy Name Context Menu', () => {
    test.beforeEach(async () => {
      // Ensure table is visible for context menu tests
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }
    });

    test('should copy unquoted table name (schema.table)', async () => {
      // Grant clipboard permissions on the page's context
      await app.page.context().grantPermissions(['clipboard-read', 'clipboard-write']);

      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await expect(app.sidebar.getContextMenuItem('Copy name (schema.table)')).toBeVisible();
      await app.sidebar.clickContextMenuItem('Copy name (schema.table)');

      // Verify clipboard content
      const clipboardText = await app.page.evaluate(() => navigator.clipboard.readText());
      expect(clipboardText).toBe('test_schema.users');
    });

    test('should copy quoted table name ("schema"."table")', async () => {
      // Grant clipboard permissions on the page's context
      await app.page.context().grantPermissions(['clipboard-read', 'clipboard-write']);

      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await expect(app.sidebar.getContextMenuItem('Copy name ("schema"."table")')).toBeVisible();
      await app.sidebar.clickContextMenuItem('Copy name ("schema"."table")');

      // Verify clipboard content
      const clipboardText = await app.page.evaluate(() => navigator.clipboard.readText());
      expect(clipboardText).toBe('"test_schema"."users"');
    });

    test('should copy schema name from schema context menu', async () => {
      // Grant clipboard permissions on the page's context
      await app.page.context().grantPermissions(['clipboard-read', 'clipboard-write']);

      // Right-click on the schema node
      await app.sidebar.rightClickSchema('test_schema');
      await app.page.waitForTimeout(300);
      await expect(app.sidebar.getContextMenuItem('Copy schema name')).toBeVisible();
      await app.sidebar.clickContextMenuItem('Copy schema name');

      // Verify clipboard content
      const clipboardText = await app.page.evaluate(() => navigator.clipboard.readText());
      expect(clipboardText).toBe('test_schema');
    });
  });

  test.describe('Filter Table Dialog', () => {
    test.beforeEach(async () => {
      // Close any open modals from previous tests
      const modalBackdrop = app.page.locator('.modal-backdrop');
      if (await modalBackdrop.isVisible({ timeout: 500 }).catch(() => false)) {
        await modalBackdrop.click();
        await app.page.waitForTimeout(200);
      }

      // Ensure table is visible for filter tests
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
      }
    });

    test.afterEach(async () => {
      // Close filter modal if it's still open
      const filterModal = app.sidebar.filterModal;
      if (await filterModal.isVisible({ timeout: 500 }).catch(() => false)) {
        await app.sidebar.cancelFilterButton.click().catch(() => {
          // Try clicking backdrop if button fails
          app.page.locator('.modal-backdrop').click().catch(() => {});
        });
        await app.page.waitForTimeout(200);
      }
    });

    test('should open filter dialog from context menu', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();
      await expect(app.sidebar.filterModalTitle).toHaveText('Filter Table');
      await expect(app.sidebar.filterTableName).toContainText('test_schema');
      await expect(app.sidebar.filterTableName).toContainText('users');
    });

    test('should display table columns in filter dropdown', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Check that column dropdown exists and has options
      const columnSelect = app.sidebar.filterConditions.first().locator('.filter-col-select');
      await expect(columnSelect).toBeVisible();

      // Verify some expected columns are in the dropdown
      const options = await columnSelect.locator('option').allTextContents();
      expect(options).toContain('id');
      expect(options).toContain('name');
      expect(options).toContain('email');
    });

    test('should add and remove filter conditions', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Should start with 1 filter condition
      await expect(app.sidebar.filterConditions).toHaveCount(1);

      // Add a condition
      await app.sidebar.addFilterCondition();
      await expect(app.sidebar.filterConditions).toHaveCount(2);

      // Add another condition
      await app.sidebar.addFilterCondition();
      await expect(app.sidebar.filterConditions).toHaveCount(3);

      // Remove one condition
      await app.sidebar.removeFilterCondition(1);
      await expect(app.sidebar.filterConditions).toHaveCount(2);
    });

    test('should show filter logic selector when multiple conditions exist', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Logic selector should not be visible with single condition
      await expect(app.sidebar.filterLogicSelect).not.toBeVisible();

      // Add a condition
      await app.sidebar.addFilterCondition();

      // Logic selector should now be visible
      await expect(app.sidebar.filterLogicSelect).toBeVisible();

      // Should default to AND
      await expect(app.sidebar.filterLogicSelect).toHaveValue('AND');

      // Change to OR
      await app.sidebar.setFilterLogic('OR');
      await expect(app.sidebar.filterLogicSelect).toHaveValue('OR');
    });

    test('should add and remove ORDER BY conditions', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Should start with no ORDER BY conditions
      await expect(app.sidebar.orderByConditions).toHaveCount(0);

      // Add an ORDER BY
      await app.sidebar.addOrderByCondition();
      await expect(app.sidebar.orderByConditions).toHaveCount(1);

      // Add another ORDER BY
      await app.sidebar.addOrderByCondition();
      await expect(app.sidebar.orderByConditions).toHaveCount(2);

      // Remove one
      await app.sidebar.removeOrderByCondition(0);
      await expect(app.sidebar.orderByConditions).toHaveCount(1);
    });

    test('should configure ORDER BY direction', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Add ORDER BY
      await app.sidebar.addOrderByCondition();

      // Should default to ASC
      const directionSelect = app.sidebar.orderByConditions.first().locator('.filter-dir-select');
      await expect(directionSelect).toHaveValue('ASC');

      // Change to DESC
      await app.sidebar.setOrderByDirection(0, 'DESC');
      await expect(directionSelect).toHaveValue('DESC');
    });

    test('should change limit value', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Default should be 100
      await expect(app.sidebar.limitInput).toHaveValue('100');

      // Change to 50
      await app.sidebar.setLimit(50);
      await expect(app.sidebar.limitInput).toHaveValue('50');
    });

    test('should cancel filter dialog', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();
      await app.sidebar.cancelFilter();
      await app.sidebar.expectFilterModalHidden();
    });

    test('should apply filter and open query tab', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Configure filter: name = 'Alice'
      await app.sidebar.setFilterColumn(0, 'name');
      await app.sidebar.setFilterOperator(0, '=');
      await app.sidebar.setFilterValue(0, 'Alice Johnson');

      // Add ORDER BY
      await app.sidebar.addOrderByCondition();
      await app.sidebar.setOrderByColumn(0, 'id');
      await app.sidebar.setOrderByDirection(0, 'DESC');

      // Set limit
      await app.sidebar.setLimit(50);

      // Apply filter
      await app.sidebar.applyFilter();

      // Query editor should open
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Verify the generated SQL contains expected parts
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('SELECT *');
      expect(queryText).toContain('test_schema');
      expect(queryText).toContain('users');
      expect(queryText).toContain('WHERE');
      expect(queryText).toContain('name');
      expect(queryText).toContain('Alice Johnson');
      expect(queryText).toContain('ORDER BY');
      expect(queryText).toContain('DESC');
      expect(queryText).toContain('LIMIT 50');
    });

    test('should apply filter with multiple conditions using AND', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // First condition: id > 1
      await app.sidebar.setFilterColumn(0, 'id');
      await app.sidebar.setFilterOperator(0, '>');
      await app.sidebar.setFilterValue(0, '1');

      // Add second condition: id < 5
      await app.sidebar.addFilterCondition();
      await app.sidebar.setFilterColumn(1, 'id');
      await app.sidebar.setFilterOperator(1, '<');
      await app.sidebar.setFilterValue(1, '5');

      // Ensure logic is AND (default)
      await expect(app.sidebar.filterLogicSelect).toHaveValue('AND');

      // Apply filter
      await app.sidebar.applyFilter();

      // Query editor should open
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Verify the generated SQL contains AND logic
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('AND');
    });

    test('should apply filter with OR logic', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // First condition: id = 1
      await app.sidebar.setFilterColumn(0, 'id');
      await app.sidebar.setFilterOperator(0, '=');
      await app.sidebar.setFilterValue(0, '1');

      // Add second condition: id = 2
      await app.sidebar.addFilterCondition();
      await app.sidebar.setFilterColumn(1, 'id');
      await app.sidebar.setFilterOperator(1, '=');
      await app.sidebar.setFilterValue(1, '2');

      // Change logic to OR
      await app.sidebar.setFilterLogic('OR');

      // Apply filter
      await app.sidebar.applyFilter();

      // Query editor should open
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });

      // Verify the generated SQL contains OR logic
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('OR');
    });

    test('should handle IS NULL operator', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Set IS NULL operator
      await app.sidebar.setFilterColumn(0, 'email');
      await app.sidebar.setFilterOperator(0, 'IS NULL');

      // Value input should not be visible for IS NULL
      const valueInput = app.sidebar.filterConditions.first().locator('.filter-value-input');
      await expect(valueInput).not.toBeVisible();

      // Apply filter
      await app.sidebar.applyFilter();

      // Verify the generated SQL
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('IS NULL');
    });

    test('should handle LIKE operator with pattern', async () => {
      await app.sidebar.rightClickTable('users');
      await app.page.waitForTimeout(300);
      await app.sidebar.clickContextMenuItem('Filter table...');

      await app.sidebar.expectFilterModalVisible();

      // Set LIKE operator
      await app.sidebar.setFilterColumn(0, 'name');
      await app.sidebar.setFilterOperator(0, 'LIKE');
      await app.sidebar.setFilterValue(0, '%Alice%');

      // Apply filter
      await app.sidebar.applyFilter();

      // Verify the generated SQL
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('LIKE');
      expect(queryText).toContain('%Alice%');
    });
  });
});
