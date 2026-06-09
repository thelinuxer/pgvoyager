import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * The Databases panel splits its load into two passes:
 *  1. cheap names + metadata, fetched automatically on connect (so the panel
 *     paints immediately).
 *  2. the expensive `pg_database_size` query, which is NOT run on connect
 *     (it scans every file in each DB dir and would block a pool connection
 *     on servers with many or large databases). It runs only on demand when
 *     the user clicks "Show sizes".
 *
 * These tests verify names render on connect, sizes are not fetched until
 * requested, and the request populates the size badges.
 */
test.describe('Databases panel — on-demand size loading', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('database names render immediately after connect', async () => {
    await expect(app.sidebar.databaseOption('postgres')).toBeVisible({ timeout: 10000 });
    const config = getTestConnectionConfig();
    await expect(app.sidebar.databaseOption(config.database)).toBeVisible({ timeout: 10000 });
  });

  test('sizes are not fetched on connect; "Show sizes" button is offered', async () => {
    const config = getTestConnectionConfig();
    // No size badge until requested.
    await expect(
      app.page.locator(`[data-testid="database-size-${config.database}"]`)
    ).toHaveCount(0);
    await expect(app.sidebar.loadDatabaseSizesButton).toBeVisible({ timeout: 10000 });
  });

  test('clicking "Show sizes" populates size badges on demand', async () => {
    const config = getTestConnectionConfig();
    await app.sidebar.loadDatabaseSizesButton.click();

    const sizeBadge = app.page.locator(`[data-testid="database-size-${config.database}"]`);
    await expect(sizeBadge).toBeVisible({ timeout: 30000 });

    const text = (await sizeBadge.textContent())?.trim() || '';
    // pg_size_pretty output: "8192 bytes", "1024 kB", "5 MB", etc.
    expect(text).toMatch(/^[\d.]+\s*(bytes|kB|MB|GB|TB)$/i);

    // Once loaded, the button is gone.
    await expect(app.sidebar.loadDatabaseSizesButton).toHaveCount(0);
  });

  test('refresh reloads names and resets sizes to on-demand', async () => {
    const config = getTestConnectionConfig();
    await app.sidebar.refreshDatabasesButton.click();

    // Names re-appear quickly.
    await expect(app.sidebar.databaseOption(config.database)).toBeVisible({ timeout: 10000 });
    // Sizes are cleared and the button is offered again.
    await expect(app.sidebar.loadDatabaseSizesButton).toBeVisible({ timeout: 10000 });
    await expect(
      app.page.locator(`[data-testid="database-size-${config.database}"]`)
    ).toHaveCount(0);
  });
});
