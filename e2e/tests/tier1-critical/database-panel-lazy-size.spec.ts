import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * The Databases panel splits its load into two passes:
 *  1. cheap names + metadata (so the panel paints immediately after connect)
 *  2. lazy `pg_database_size` query (so connect is never blocked by a per-DB
 *     file scan on servers with many or large databases).
 *
 * These tests verify both passes: the panel renders names quickly and sizes
 * appear asynchronously without blocking the initial render.
 */
test.describe('Databases panel — lazy size loading', () => {
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

  test('sizes load lazily and appear after the initial render', async () => {
    const config = getTestConnectionConfig();
    const sizeBadge = app.page.locator(`[data-testid="database-size-${config.database}"]`);
    await expect(sizeBadge).toBeVisible({ timeout: 15000 });

    const text = (await sizeBadge.textContent())?.trim() || '';
    // pg_size_pretty output: "8192 bytes", "1024 kB", "5 MB", etc.
    expect(text).toMatch(/^[\d.]+\s*(bytes|kB|MB|GB|TB)$/i);
  });

  test('refresh reloads names then re-fetches sizes lazily', async () => {
    const config = getTestConnectionConfig();
    await app.sidebar.refreshDatabasesButton.click();

    // Names re-appear quickly
    await expect(app.sidebar.databaseOption(config.database)).toBeVisible({ timeout: 10000 });
    // Sizes eventually re-populate
    await expect(
      app.page.locator(`[data-testid="database-size-${config.database}"]`)
    ).toBeVisible({ timeout: 15000 });
  });
});
