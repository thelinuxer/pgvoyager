import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config({ path: path.join(__dirname, '.env') });

const isCI = process.env.CI === 'true' || process.env.CI === '1';
const baseURL = process.env.BASE_URL || 'http://localhost:5137';

export default defineConfig({
  testDir: './tests',

  // Run tests serially to share browser state and avoid connection spam
  fullyParallel: false,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: isCI,

  // Retry on CI only
  retries: isCI ? 2 : 0,

  // Use single worker to avoid multiple browser instances and connection spam
  workers: 1,

  // Reporter configuration
  reporter: [
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['junit', { outputFile: 'test-results/junit.xml' }],
    ['list'],
    ...(isCI ? [['github'] as const] : []),
  ],

  // Shared settings for all projects
  use: {
    baseURL,

    // Collect trace when retrying the failed test
    trace: 'on-first-retry',

    // Capture screenshot on failure
    screenshot: 'only-on-failure',

    // Record video on first retry
    video: 'on-first-retry',

    // Timeouts
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },

  // Global setup and teardown
  globalSetup: path.join(__dirname, 'global-setup.ts'),
  globalTeardown: path.join(__dirname, 'global-teardown.ts'),

  // Configure projects for browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Uncomment to add more browsers in local development
    // {
    //   name: 'firefox',
    //   use: { ...devices['Desktop Firefox'] },
    // },
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  // Run local dev server before starting the tests
  // For local development: start the app manually with `make dev` in a separate terminal
  // For CI: the production binary is built and run with embedded frontend
  webServer: isCI ? {
    command: 'cd .. && PGVOYAGER_MODE=production ./pgvoyager',
    url: baseURL,
    reuseExistingServer: false,
    timeout: 30000,
  } : undefined,  // Local dev: start app manually with `make dev`

  // Output directory for test artifacts
  outputDir: 'test-results/artifacts',

  // Timeout for each test
  timeout: 60000,

  // Expect timeout
  expect: {
    timeout: 10000,
  },
});
