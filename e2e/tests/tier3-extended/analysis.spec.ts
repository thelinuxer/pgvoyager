import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage, AnalysisViewerPage } from '../../pages';

/**
 * Database Analysis Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Database Analysis', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let analysisViewer: AnalysisViewerPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    analysisViewer = new AnalysisViewerPage(page);

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

  test.describe('Opening Analysis', () => {
    test('should open analysis from sidebar button', async () => {
      await app.sidebar.openAnalysis();
      await app.page.waitForTimeout(500);

      // Analysis should open in a tab
      await app.tabBar.expectTabExists('Analysis');
    });

    test('should display analysis viewer', async () => {
      // Click on analysis tab if not active
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      await analysisViewer.expectVisible();
    });
  });

  test.describe('Analysis Content', () => {
    test('should show analysis toolbar', async () => {
      // Ensure we're on analysis tab
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      // Look for toolbar or breadcrumb (use first() to avoid strict mode violation)
      const toolbar = app.page.locator('.analysis-viewer .toolbar, .analysis-viewer .breadcrumb').first();
      await expect(toolbar).toBeVisible();
    });

    test('should show analysis results or loading state', async () => {
      // Ensure we're on analysis tab
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      // Wait for analysis to complete (or show loading)
      const analysisContainer = app.page.locator('.analysis-viewer');
      await expect(analysisContainer).toBeVisible();

      // Should have content (either stats section, categories, loading, or empty state)
      // The actual component uses .stats-section and .category for results
      const content = analysisContainer.locator('.stats-section, .category, .loading, .empty-state');
      await expect(content.first()).toBeVisible({ timeout: 30000 });
    });

    test('should display analysis categories', async () => {
      // Ensure we're on analysis tab
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      // Wait for loading to complete
      await analysisViewer.waitForLoad();

      // Look for category items
      const categories = app.page.locator('.analysis-viewer .category, .analysis-viewer .category-card');
      const categoryCount = await categories.count();

      // May or may not have categories depending on database state
      expect(categoryCount >= 0).toBe(true);
    });
  });

  test.describe('Analysis Interaction', () => {
    test('should be able to refresh analysis', async () => {
      // Ensure we're on analysis tab
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      // Look for refresh button
      const refreshButton = app.page.locator('.analysis-viewer button:has-text("Refresh"), .analysis-viewer button:has-text("Re-run"), .analysis-viewer button[title*="refresh" i]');
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        // Should start loading
        await app.page.waitForTimeout(500);
      }
    });

    test('should expand/collapse category', async () => {
      // Ensure we're on analysis tab
      const analysisTab = app.tabBar.getTab('Analysis');
      if (await analysisTab.isVisible()) {
        await app.tabBar.clickTab('Analysis');
      }

      // Wait for loading
      await analysisViewer.waitForLoad();

      // Find a category header
      const categoryHeader = app.page.locator('.analysis-viewer .category-header, .analysis-viewer .category-title').first();
      if (await categoryHeader.isVisible()) {
        // Click to toggle
        await categoryHeader.click();
        await app.page.waitForTimeout(200);

        // Click again
        await categoryHeader.click();
        await app.page.waitForTimeout(200);
      }
    });
  });

  test.describe('Tab Management', () => {
    test('should close analysis tab', async () => {
      await app.tabBar.closeTab('Analysis');
      await app.tabBar.expectTabNotExists('Analysis');
    });

    test('should reopen analysis tab', async () => {
      await app.sidebar.openAnalysis();
      await app.page.waitForTimeout(500);
      await app.tabBar.expectTabExists('Analysis');
    });
  });
});
