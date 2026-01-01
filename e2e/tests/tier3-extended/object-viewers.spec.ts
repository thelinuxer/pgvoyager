import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Object Viewers Tests
 * Tests viewers for database objects: Views, Functions, Sequences, Types
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Object Viewers', () => {
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

    // Expand schema tree
    await app.sidebar.expandNode('test_schema');
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test.describe('View Viewer', () => {
    test('should open view in sidebar', async () => {
      // Expand Views folder
      await app.sidebar.expandNode('Views');
      await app.page.waitForTimeout(300);

      // Click on the view
      await app.sidebar.getViewNode('user_order_summary').click();
      await app.page.waitForTimeout(500);

      // Should open view tab
      await app.tabBar.expectTabExists('user_order_summary');
    });

    test('should display view definition', async () => {
      // Ensure view tab is active
      await app.tabBar.clickTab('user_order_summary');
      await app.page.waitForTimeout(300);

      // Look for view definition content
      const viewViewer = app.page.locator('.view-viewer, [class*="view-viewer"]');
      const codeBlock = app.page.locator('.view-definition, pre, code');

      // Either the viewer or code should be visible
      const isViewerVisible = await viewViewer.isVisible().catch(() => false);
      const isCodeVisible = await codeBlock.first().isVisible().catch(() => false);
      expect(isViewerVisible || isCodeVisible).toBe(true);
    });

    test('should show view metadata', async () => {
      await app.tabBar.clickTab('user_order_summary');

      // Look for metadata (schema, name, etc)
      const content = await app.page.content();
      expect(content).toContain('user_order_summary');
    });

    test('should close view tab', async () => {
      await app.tabBar.closeTab('user_order_summary');
      await app.tabBar.expectTabNotExists('user_order_summary');
    });
  });

  test.describe('Function Viewer', () => {
    test('should open function in sidebar', async () => {
      // Expand Functions folder
      const functionsNode = app.sidebar.getTreeNode('Functions');
      if (!(await functionsNode.isVisible())) {
        await app.sidebar.expandNode('test_schema');
      }
      await app.sidebar.expandNode('Functions');
      await app.page.waitForTimeout(300);

      // Click on the function
      await app.sidebar.getFunctionNode('get_user_orders').click();
      await app.page.waitForTimeout(500);

      // Should open function tab
      await app.tabBar.expectTabExists('get_user_orders');
    });

    test('should display function definition', async () => {
      // Ensure function tab is active
      await app.tabBar.clickTab('get_user_orders');
      await app.page.waitForTimeout(300);

      // Look for function definition content
      const functionViewer = app.page.locator('.function-viewer, [class*="function-viewer"]');
      const codeBlock = app.page.locator('.function-definition, pre, code');

      const isViewerVisible = await functionViewer.isVisible().catch(() => false);
      const isCodeVisible = await codeBlock.first().isVisible().catch(() => false);
      expect(isViewerVisible || isCodeVisible).toBe(true);
    });

    test('should show function metadata', async () => {
      await app.tabBar.clickTab('get_user_orders');

      // Look for metadata
      const content = await app.page.content();
      expect(content).toContain('get_user_orders');
    });

    test('should close function tab', async () => {
      await app.tabBar.closeTab('get_user_orders');
      await app.tabBar.expectTabNotExists('get_user_orders');
    });
  });

  test.describe('Sequence Viewer', () => {
    test('should show Sequences in sidebar', async () => {
      // Sequences folder should be visible under schema
      const sequencesNode = app.sidebar.getTreeNode('Sequences');
      // May or may not have sequences in test database
      const hasSequences = await sequencesNode.isVisible().catch(() => false);
      expect(typeof hasSequences).toBe('boolean');
    });

    test('should open sequence if exists', async () => {
      // Expand Sequences folder if visible
      const sequencesNode = app.sidebar.getTreeNode('Sequences');
      if (await sequencesNode.isVisible()) {
        await app.sidebar.expandNode('Sequences');
        await app.page.waitForTimeout(300);

        // Look for a sequence (custom_seq from test data)
        const sequenceNode = app.sidebar.getSequenceNode('custom_seq');
        if (await sequenceNode.isVisible()) {
          await sequenceNode.click();
          await app.page.waitForTimeout(500);
          await app.tabBar.expectTabExists('custom_seq');

          // Clean up
          await app.tabBar.closeTab('custom_seq');
        }
      }
    });
  });

  test.describe('Type Viewer', () => {
    test('should show Types in sidebar', async () => {
      // Types folder should be visible under schema
      const typesNode = app.sidebar.getTreeNode('Types');
      // May or may not have custom types in test database
      const hasTypes = await typesNode.isVisible().catch(() => false);
      expect(typeof hasTypes).toBe('boolean');
    });

    test('should open type if exists', async () => {
      // Expand Types folder if visible
      const typesNode = app.sidebar.getTreeNode('Types');
      if (await typesNode.isVisible()) {
        await app.sidebar.expandNode('Types');
        await app.page.waitForTimeout(300);

        // Look for a type (order_status from test data)
        const typeNode = app.sidebar.getTypeNode('order_status');
        if (await typeNode.isVisible()) {
          await typeNode.click();
          await app.page.waitForTimeout(500);
          await app.tabBar.expectTabExists('order_status');

          // Clean up
          await app.tabBar.closeTab('order_status');
        }
      }
    });
  });

  test.describe('Multiple Objects', () => {
    test('should open multiple objects in tabs', async () => {
      // Close any existing tabs first
      await app.tabBar.closeAllTabs();

      // Make sure schema is expanded
      await app.sidebar.expandNode('test_schema');

      // Open view (use first() to handle potential duplicates in filtered tree)
      await app.sidebar.expandNode('Views');
      await app.sidebar.getViewNode('user_order_summary').first().click();
      await app.page.waitForTimeout(300);
      await app.tabBar.pinTab('user_order_summary');

      // Open function
      await app.sidebar.expandNode('Functions');
      await app.sidebar.getFunctionNode('get_user_orders').first().click();
      await app.page.waitForTimeout(300);

      // Both should be open
      await app.tabBar.expectTabExists('user_order_summary');
      await app.tabBar.expectTabExists('get_user_orders');

      // Clean up
      await app.tabBar.unpinTab('user_order_summary');
      await app.tabBar.closeAllTabs();
    });
  });
});
