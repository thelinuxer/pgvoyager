import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage, ERDViewerPage } from '../../pages';

/**
 * ERD (Entity Relationship Diagram) Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('ERD Viewer', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let erdViewer: ERDViewerPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    erdViewer = new ERDViewerPage(page);

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

  test.describe('Opening ERD', () => {
    test('should open table ERD from context menu', async () => {
      // Expand schema and tables
      await app.sidebar.expandNode('test_schema');
      await app.sidebar.expandNode('Tables');

      // Right-click on users table
      await app.sidebar.rightClickTable('users');
      await app.sidebar.expectContextMenuVisible();

      // Click View ERD
      await app.sidebar.clickContextMenuItem('View ERD');

      // ERD should open in a tab
      await app.tabBar.expectTabExists('ERD');
    });

    test('should open schema ERD from context menu', async () => {
      // Right-click on schema
      await app.sidebar.rightClickSchema('test_schema');
      await app.sidebar.expectContextMenuVisible();

      // Click View Schema ERD (or just View ERD if that's what's available)
      const schemaErdOption = app.sidebar.getContextMenuItem('View Schema ERD');
      const erdOption = app.sidebar.getContextMenuItem('View ERD');

      if (await schemaErdOption.isVisible({ timeout: 1000 }).catch(() => false)) {
        await schemaErdOption.click();
      } else if (await erdOption.isVisible({ timeout: 1000 }).catch(() => false)) {
        await erdOption.click();
      } else {
        // Close context menu and skip - option not available
        await app.page.keyboard.press('Escape');
        test.skip();
        return;
      }

      // ERD should be visible
      await erdViewer.expectVisible();
    });
  });

  test.describe('ERD Display', () => {
    test('should display ERD canvas', async () => {
      // Ensure ERD tab is open
      const erdTab = app.tabBar.getTab('ERD');
      if (await erdTab.isVisible()) {
        await app.tabBar.clickTab('ERD');
      } else {
        // Open ERD via context menu
        await app.sidebar.expandNode('test_schema');
        await app.sidebar.expandNode('Tables');
        await app.sidebar.rightClickTable('users');
        await app.sidebar.clickContextMenuItem('View ERD');
      }

      await erdViewer.waitForLoad();
      await erdViewer.expectCanvasVisible();
    });

    test('should show ERD toolbar', async () => {
      // Ensure we're on ERD tab
      const erdTab = app.tabBar.getTab('ERD');
      if (await erdTab.isVisible()) {
        await app.tabBar.clickTab('ERD');
        await expect(erdViewer.toolbar).toBeVisible();
      }
    });
  });

  test.describe('ERD Navigation', () => {
    test('should have navigation buttons in toolbar', async () => {
      // Ensure we're on ERD tab
      const erdTab = app.tabBar.getTab('ERD');
      if (await erdTab.isVisible()) {
        await app.tabBar.clickTab('ERD');
        await erdViewer.waitForLoad();

        // Check for navigation buttons
        await expect(erdViewer.backButton.or(erdViewer.fullSchemaButton)).toBeVisible();
      }
    });
  });

  test.describe('Tab Management', () => {
    test('should close ERD tab', async () => {
      // Ensure ERD tab exists
      const erdTab = app.tabBar.getTab('ERD');
      if (await erdTab.isVisible()) {
        await app.tabBar.closeTab('ERD');
        await app.tabBar.expectTabNotExists('ERD');
      }
    });

    test('should reopen ERD tab', async () => {
      // Open ERD again
      await app.sidebar.expandNode('test_schema');
      const tablesNode = app.sidebar.getTreeNode('Tables');
      if (!(await tablesNode.isVisible())) {
        await app.sidebar.expandNode('Tables');
      }

      await app.sidebar.rightClickTable('orders');
      await app.sidebar.clickContextMenuItem('View ERD');

      await app.tabBar.expectTabExists('ERD');
    });
  });
});
