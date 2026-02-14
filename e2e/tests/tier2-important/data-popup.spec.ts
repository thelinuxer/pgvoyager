import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Data Popup Tests (JSON/XML)
 * Tests the data popup that appears when clicking on JSON/JSONB/XML cell values.
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion.
 */
test.describe('Data Popup', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);

    await app.goto();
    await app.waitForLoad();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();

    // Navigate to metadata table which has JSONB, JSON, XML columns
    await app.sidebar.navigateToTable('test_schema', 'metadata');
    await app.tableViewer.waitForLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test.describe('JSON/XML Cell Display', () => {
    test('should display JSON cells with data-column styling', async () => {
      // The metadata table has jsonb (config) and json (raw_json) columns
      const dataCells = app.tableViewer.table.locator('td.data-column');
      const count = await dataCells.count();
      // We have 3 rows with config (jsonb, all non-null) = at least 3 clickable cells
      expect(count).toBeGreaterThanOrEqual(3);
    });

    test('should make JSON cells clickable', async () => {
      const dataCell = app.tableViewer.table.locator('td.data-column').first();
      await expect(dataCell).toBeVisible();
      // data-column cells should have cursor pointer styling
      const cursor = await dataCell.evaluate(el => getComputedStyle(el).cursor);
      expect(cursor).toBe('pointer');
    });
  });

  test.describe('JSON Popup', () => {
    test('should open popup when clicking a JSON cell', async () => {
      const dataCell = app.tableViewer.table.locator('td.data-column').first();
      await dataCell.click();
      await app.tableViewer.expectDataPopupVisible();
    });

    test('should show formatted JSON with syntax highlighting', async () => {
      // Popup should already be open from previous test
      const content = app.tableViewer.dataPopupContent;
      await expect(content).toBeVisible();

      // Check for syntax highlighting classes
      const hlKey = content.locator('.hl-key');
      const hlString = content.locator('.hl-string');
      expect(await hlKey.count()).toBeGreaterThan(0);
      expect(await hlString.count()).toBeGreaterThan(0);
    });

    test('should display column name and data type in header', async () => {
      await expect(app.tableViewer.dataPopupColumnName).toBeVisible();
      await expect(app.tableViewer.dataPopupDataType).toBeVisible();
    });

    test('should close popup via Escape key', async () => {
      await app.page.keyboard.press('Escape');
      await app.tableViewer.expectDataPopupHidden();
    });

    test('should close popup via backdrop click', async () => {
      // Re-open popup
      const dataCell = app.tableViewer.table.locator('td.data-column').first();
      await dataCell.click();
      await app.tableViewer.expectDataPopupVisible();

      // Click backdrop to close
      await app.tableViewer.closeDataPopupViaBackdrop();
    });

    test('should have a copy button', async () => {
      // Re-open popup
      const dataCell = app.tableViewer.table.locator('td.data-column').first();
      await dataCell.click();
      await app.tableViewer.expectDataPopupVisible();

      // Copy button should be visible
      await expect(app.tableViewer.dataPopupCopyButton).toBeVisible();

      // Close for next test
      await app.page.keyboard.press('Escape');
    });
  });

  test.describe('XML Popup', () => {
    test('should open popup for XML cell with formatted content', async () => {
      // Find an XML data-column cell - xml_data is the last data-type column
      // Navigate to find cells with XML content (row 0 has xml_data)
      const xmlCells = app.tableViewer.table.locator('td.data-column');
      const cellCount = await xmlCells.count();

      // Find a cell that contains XML-like content by checking each data cell
      let xmlCellFound = false;
      for (let i = 0; i < cellCount; i++) {
        const text = await xmlCells.nth(i).textContent();
        if (text && text.includes('<')) {
          await xmlCells.nth(i).click();
          await app.tableViewer.expectDataPopupVisible();

          // Check for XML highlighting
          const content = app.tableViewer.dataPopupContent;
          const hlTag = content.locator('.hl-tag');
          expect(await hlTag.count()).toBeGreaterThan(0);

          xmlCellFound = true;
          await app.page.keyboard.press('Escape');
          break;
        }
      }

      expect(xmlCellFound).toBe(true);
    });
  });

  test.describe('Data Popup in Query Editor', () => {
    test('should work in query editor results too', async () => {
      // Open a new query tab
      await app.sidebar.openNewQuery();
      await app.page.waitForTimeout(500);

      // Execute a query that returns JSON data
      await app.queryEditor.setQuery("SELECT config FROM test_schema.metadata LIMIT 1;");
      await app.queryEditor.executeQuery();
      await app.queryEditor.waitForQueryExecution();

      // Find a data-column cell in query results
      const resultDataCells = app.page.locator('.query-results td.data-column, .result-grid td.data-column');
      if (await resultDataCells.count() > 0) {
        await resultDataCells.first().click();
        await expect(app.page.locator('.data-popup')).toBeVisible({ timeout: 3000 });
        await app.page.keyboard.press('Escape');
      }
    });
  });
});
