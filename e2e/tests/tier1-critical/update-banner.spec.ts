import { test, expect } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:5137';

test.describe('Update banner', () => {
  test('shows "Restart to update" when status is ready', async ({ page }) => {
    // Intercept the status poll and force a ready desktop update.
    await page.route('**/api/update/status', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          edition: 'desktop',
          status: 'ready',
          currentVersion: '0.3.5',
          latestVersion: '9.9.9',
          releaseUrl: 'https://example.com'
        })
      })
    );

    await page.goto(BASE);
    await expect(page.locator('[data-testid="btn-update-restart"]')).toBeVisible({ timeout: 10000 });
  });
});
