import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Temporal column rendering — date/timestamp/time columns must be displayed
 * without ISO `T`/`Z` markers and without zero-time padding on bare dates.
 *
 * Backend serializes Postgres time values as RFC3339 strings (e.g.
 * `"1990-05-15T00:00:00Z"` for a `date`); the DataGrid normalizes these for
 * human display.
 */
test.describe('DataGrid — temporal column formatting', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
    await app.sidebar.navigateToTable('test_schema', 'temporal_test');
    await app.tableViewer.waitForLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('date column renders without zero-time portion', async () => {
    const columns = await app.tableViewer.getColumnNames();
    const idx = columns.findIndex((c) => c.includes('birth_date'));
    expect(idx).toBeGreaterThanOrEqual(0);

    const value = (await app.tableViewer.getCellValue(0, idx)).trim();
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(value).not.toContain('T');
    expect(value).not.toContain('00:00:00');
    expect(value).not.toContain('Z');
  });

  test('timestamp column renders with space separator and no Z', async () => {
    const columns = await app.tableViewer.getColumnNames();
    const idx = columns.findIndex((c) => c === 'event_at' || c.endsWith('event_at'));
    expect(idx).toBeGreaterThanOrEqual(0);

    const value = (await app.tableViewer.getCellValue(0, idx)).trim();
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
    expect(value).not.toContain('T');
    expect(value).not.toContain('Z');
    expect(value).not.toMatch(/\.\d+/);
  });

  test('timestamptz column renders without ISO T separator', async () => {
    const columns = await app.tableViewer.getColumnNames();
    const idx = columns.findIndex((c) => c.includes('event_at_tz'));
    expect(idx).toBeGreaterThanOrEqual(0);

    const value = (await app.tableViewer.getCellValue(0, idx)).trim();
    expect(value).not.toContain('T');
    // Either a bare timestamp (offset normalized to UTC and displayed without Z)
    // or with a "+HH:MM" trailing offset — both acceptable.
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}( [+-]\d{2}:?\d{2})?$/);
  });

  test('time column renders HH:MM:SS', async () => {
    const columns = await app.tableViewer.getColumnNames();
    const idx = columns.findIndex((c) => c === 'event_time' || c.endsWith('event_time'));
    expect(idx).toBeGreaterThanOrEqual(0);

    const value = (await app.tableViewer.getCellValue(0, idx)).trim();
    expect(value).toMatch(/^\d{2}:\d{2}:\d{2}( [+-]\d{2}:?\d{2})?$/);
    expect(value).not.toMatch(/\.\d+/);
  });
});
