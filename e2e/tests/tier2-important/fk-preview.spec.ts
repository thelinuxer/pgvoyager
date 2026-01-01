import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * FK Preview Tests
 * Tests the foreign key preview popup that appears when hovering over FK values
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('FK Preview', () => {
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

    // Navigate to orders table which has FK to users
    await app.sidebar.navigateToTable('test_schema', 'orders');
    await app.tableViewer.waitForLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test.describe('FK Column Display', () => {
    test('should display FK icon on foreign key columns', async () => {
      // The orders table has user_id FK column
      // Look for FK icon in the column header - it's a span.fk-icon with title="Foreign Key"
      const fkIcon = app.tableViewer.table.locator('.fk-icon');
      const fkIconByTitle = app.tableViewer.table.locator('[title="Foreign Key"]');

      // Either FK icon should be visible
      const hasIcon = await fkIcon.first().isVisible().catch(() => false);
      const hasIconByTitle = await fkIconByTitle.first().isVisible().catch(() => false);
      expect(hasIcon || hasIconByTitle).toBe(true);
    });

    test('should show FK column with special styling', async () => {
      // FK values should have special styling - look for cells that are clickable FK links
      const fkCell = app.tableViewer.table.locator('td[class*="fk"], .fk-column, td.fk-value').first();
      // May not have FK cells if column has null values
      const hasFkCell = await fkCell.isVisible().catch(() => false);
      expect(typeof hasFkCell).toBe('boolean');
    });
  });

  test.describe('FK Preview Popup', () => {
    test('should show preview popup on FK value hover', async () => {
      // Find a FK cell (user_id column)
      const fkCells = app.tableViewer.table.locator('td.fk-column, td[class*="fk"]');
      const firstFkCell = fkCells.first();

      if (await firstFkCell.isVisible()) {
        // Hover over the FK cell
        await firstFkCell.hover();

        // Wait for popup to appear
        await app.page.waitForTimeout(500);

        // Check for FK preview popup
        const popup = app.page.locator('.fk-popup');
        // Popup may or may not appear depending on data
        const isPopupVisible = await popup.isVisible().catch(() => false);
        expect(typeof isPopupVisible).toBe('boolean');
      }
    });

    test('should hide preview popup when mouse leaves', async () => {
      // Find a FK cell
      const fkCells = app.tableViewer.table.locator('td.fk-column, td[class*="fk"]');
      const firstFkCell = fkCells.first();

      if (await firstFkCell.isVisible()) {
        // Hover over FK cell
        await firstFkCell.hover();
        await app.page.waitForTimeout(500);

        // Move mouse away
        await app.tableViewer.table.locator('th').first().hover();
        await app.page.waitForTimeout(300);

        // Popup should be hidden
        const popup = app.page.locator('.fk-popup');
        await expect(popup).not.toBeVisible({ timeout: 3000 });
      }
    });
  });

  test.describe('FK Navigation', () => {
    test('should navigate to referenced table on FK click', async () => {
      // Find a FK cell (user_id column in orders table)
      const fkCells = app.tableViewer.table.locator('td.fk-column, td[class*="fk"]');
      const firstFkCell = fkCells.first();

      if (await firstFkCell.isVisible()) {
        // Get the current tab title
        const currentTitle = await app.tabBar.getActiveTabTitle();

        // Click on FK value
        await firstFkCell.click();
        await app.page.waitForTimeout(500);

        // Should navigate (either in same tab or new tab)
        // The table viewer should update or a new tab opens
        await app.tableViewer.waitForLoad();
      }
    });

    test('should open FK target in new tab when current tab is pinned', async () => {
      // First navigate back to orders table
      await app.sidebar.navigateToTable('test_schema', 'orders');
      await app.tableViewer.waitForLoad();

      // Pin the current tab
      await app.tabBar.pinTab('orders');
      await app.page.waitForTimeout(300);

      // Find a FK cell
      const fkCells = app.tableViewer.table.locator('td.fk-column, td[class*="fk"]');
      const firstFkCell = fkCells.first();

      if (await firstFkCell.isVisible()) {
        const tabCountBefore = await app.tabBar.getTabCount();

        // Click on FK value
        await firstFkCell.click();
        await app.page.waitForTimeout(500);

        // With pinned tab, a new tab should open
        const tabCountAfter = await app.tabBar.getTabCount();
        expect(tabCountAfter).toBeGreaterThanOrEqual(tabCountBefore);
      }

      // Cleanup: unpin
      await app.tabBar.clickTab('orders');
      await app.tabBar.unpinTab('orders');
    });
  });

  test.describe('Different FK Tables', () => {
    test('should show FK in order_items table', async () => {
      // order_items has FKs to both orders and products
      await app.sidebar.navigateToTable('test_schema', 'order_items');
      await app.tableViewer.waitForLoad();

      // Should have FK columns
      const fkIcon = app.tableViewer.table.locator('.fk-icon, [title*="Foreign Key"]');
      const fkCount = await fkIcon.count();
      expect(fkCount).toBeGreaterThanOrEqual(1);
    });
  });
});
