import { Page, expect } from '@playwright/test';

/**
 * Wait for a network request to complete
 */
export async function waitForApiCall(
  page: Page,
  urlPattern: string | RegExp,
  timeout = 10000
): Promise<void> {
  await page.waitForResponse(
    (response) => {
      const url = response.url();
      if (typeof urlPattern === 'string') {
        return url.includes(urlPattern);
      }
      return urlPattern.test(url);
    },
    { timeout }
  );
}

/**
 * Wait for all pending network requests to complete
 */
export async function waitForNetworkIdle(page: Page, timeout = 5000): Promise<void> {
  await page.waitForLoadState('networkidle', { timeout });
}

/**
 * Take a screenshot with a descriptive name
 */
export async function takeScreenshot(page: Page, name: string): Promise<void> {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  await page.screenshot({
    path: `test-results/screenshots/${name}-${timestamp}.png`,
    fullPage: true,
  });
}

/**
 * Retry an action until it succeeds or times out
 */
export async function retry<T>(
  action: () => Promise<T>,
  options: { maxAttempts?: number; delay?: number } = {}
): Promise<T> {
  const { maxAttempts = 3, delay = 1000 } = options;

  let lastError: Error | undefined;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await action();
    } catch (error) {
      lastError = error as Error;
      if (attempt < maxAttempts) {
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
  }

  throw lastError;
}

/**
 * Generate a random string for test data
 */
export function randomString(length = 8): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

/**
 * Generate a unique test identifier
 */
export function uniqueTestId(prefix = 'test'): string {
  return `${prefix}-${Date.now()}-${randomString(4)}`;
}

/**
 * Wait for toast message to appear and optionally verify text
 */
export async function waitForToast(
  page: Page,
  expectedText?: string | RegExp,
  timeout = 5000
): Promise<void> {
  const toast = page.locator('.toast, [class*="toast"], .notification');
  await expect(toast).toBeVisible({ timeout });

  if (expectedText) {
    if (typeof expectedText === 'string') {
      await expect(toast).toContainText(expectedText);
    } else {
      await expect(toast).toHaveText(expectedText);
    }
  }
}

/**
 * Wait for loading indicator to disappear
 */
export async function waitForLoadingComplete(page: Page, timeout = 30000): Promise<void> {
  const loading = page.locator('.loading, .spinner, [class*="loading"]');

  try {
    // Wait for loading to appear (briefly)
    await loading.waitFor({ state: 'visible', timeout: 1000 });
  } catch {
    // Loading might already be gone or not appear
  }

  // Wait for loading to disappear
  await expect(loading).not.toBeVisible({ timeout });
}

/**
 * Check if an element is in the viewport
 */
export async function isInViewport(page: Page, selector: string): Promise<boolean> {
  return page.evaluate((sel) => {
    const element = document.querySelector(sel);
    if (!element) return false;

    const rect = element.getBoundingClientRect();
    return (
      rect.top >= 0 &&
      rect.left >= 0 &&
      rect.bottom <= window.innerHeight &&
      rect.right <= window.innerWidth
    );
  }, selector);
}

/**
 * Scroll element into view
 */
export async function scrollIntoView(page: Page, selector: string): Promise<void> {
  await page.evaluate((sel) => {
    const element = document.querySelector(sel);
    element?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }, selector);
}

/**
 * Get all console errors from the page
 */
export function setupConsoleErrorCapture(page: Page): string[] {
  const errors: string[] = [];

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });

  page.on('pageerror', (error) => {
    errors.push(error.message);
  });

  return errors;
}

/**
 * Assert no console errors occurred
 */
export function assertNoConsoleErrors(
  errors: string[],
  ignorePatterns: (string | RegExp)[] = []
): void {
  const filteredErrors = errors.filter((error) => {
    return !ignorePatterns.some((pattern) => {
      if (typeof pattern === 'string') {
        return error.includes(pattern);
      }
      return pattern.test(error);
    });
  });

  expect(filteredErrors).toHaveLength(0);
}
