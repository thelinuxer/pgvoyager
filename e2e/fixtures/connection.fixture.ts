import { test as base, expect } from '@playwright/test';
import { AppPage } from '../pages';
import { getTestConnectionConfig, cleanPgVoyagerState, resetTestData } from './database';

/**
 * Custom test fixtures for PgVoyager E2E tests
 */

// Extend the base test with custom fixtures
export const test = base.extend<{
  // AppPage instance (always available)
  app: AppPage;
  // Pre-connected app (use for tests that need database connection)
  connectedApp: AppPage;
}>({
  // Basic app fixture - just provides AppPage
  app: async ({ page }, use) => {
    const app = new AppPage(page);
    await use(app);
  },

  // Connected app fixture - navigates to app and establishes connection
  connectedApp: async ({ page }, use) => {
    const app = new AppPage(page);

    // Navigate to app
    await app.goto();

    // Create and connect to test database
    const connectionConfig = getTestConnectionConfig();
    await app.connectToTestDatabase(connectionConfig);

    // Verify connection was successful
    await app.expectConnected(connectionConfig.name);

    await use(app);

    // Cleanup: No need to disconnect as page will close
  },
});

// Re-export expect for convenience
export { expect };

/**
 * Worker-level fixture for test files
 * Note: We don't clean SQLite state as app may be running
 */
export const testWithCleanState = test.extend<{}, { cleanStateWorker: void }>({
  cleanStateWorker: [
    async ({}, use) => {
      console.log('🧪 Starting test file');
      await use();
    },
    { scope: 'worker', auto: true },
  ],
});

/**
 * Test fixture that also resets PostgreSQL test data
 * Use this for CRUD tests that modify the database
 */
export const testWithDataReset = test.extend<{ resetData: void }>({
  resetData: async ({}, use) => {
    // Reset test data before the test
    await resetTestData();
    await use();
    // Optionally reset after test too (for isolation)
    // await resetTestData();
  },
});

/**
 * Combined fixture: clean SQLite state + connected app
 * Most tests should use this
 */
export const connectedTest = testWithCleanState.extend<{
  connectedApp: AppPage;
}>({
  connectedApp: async ({ page }, use) => {
    const app = new AppPage(page);

    // Navigate to app
    await app.goto();

    // Create and connect to test database
    const connectionConfig = getTestConnectionConfig();
    await app.connectToTestDatabase(connectionConfig);

    // Verify connection was successful
    await app.expectConnected(connectionConfig.name);

    await use(app);
  },
});

/**
 * Full fixture for CRUD tests: clean state + connected + data reset
 */
export const crudTest = testWithCleanState.extend<{
  connectedApp: AppPage;
}>({
  connectedApp: async ({ page }, use) => {
    // Reset PostgreSQL test data
    await resetTestData();

    const app = new AppPage(page);

    // Navigate to app
    await app.goto();

    // Create and connect to test database
    const connectionConfig = getTestConnectionConfig();
    await app.connectToTestDatabase(connectionConfig);

    // Verify connection was successful
    await app.expectConnected(connectionConfig.name);

    await use(app);
  },
});
