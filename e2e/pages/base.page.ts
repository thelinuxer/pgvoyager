import { Page, Locator, expect } from '@playwright/test';

/**
 * Base page class with common utilities for all page objects
 */
export abstract class BasePage {
  constructor(protected readonly page: Page) {}

  // Common loading indicators
  protected get loadingSpinner(): Locator {
    return this.page.locator('.loading, .spinner, [class*="loading"]');
  }

  protected get toastNotification(): Locator {
    return this.page.locator('[role="alert"], .toast, .notification');
  }

  // Wait utilities
  async waitForLoad(): Promise<void> {
    await this.page.waitForLoadState('networkidle');
  }

  async waitForDomContentLoaded(): Promise<void> {
    await this.page.waitForLoadState('domcontentloaded');
  }

  async waitForNoSpinners(timeout = 10000): Promise<void> {
    await expect(this.loadingSpinner).toHaveCount(0, { timeout });
  }

  async waitForToast(text?: string, timeout = 5000): Promise<void> {
    if (text) {
      await expect(this.toastNotification).toContainText(text, { timeout });
    } else {
      await expect(this.toastNotification).toBeVisible({ timeout });
    }
  }

  async waitForToastDismiss(timeout = 10000): Promise<void> {
    await expect(this.toastNotification).not.toBeVisible({ timeout });
  }

  // Screenshot utility
  async screenshot(name: string): Promise<void> {
    await this.page.screenshot({
      path: `test-results/screenshots/${name}.png`,
      fullPage: true,
    });
  }

  // Keyboard shortcuts
  async pressCtrlEnter(): Promise<void> {
    await this.page.keyboard.press('Control+Enter');
  }

  async pressCtrlS(): Promise<void> {
    await this.page.keyboard.press('Control+s');
  }

  async pressCtrlBacktick(): Promise<void> {
    await this.page.keyboard.press('Control+`');
  }

  async pressEscape(): Promise<void> {
    await this.page.keyboard.press('Escape');
  }

  async pressEnter(): Promise<void> {
    await this.page.keyboard.press('Enter');
  }

  async pressTab(): Promise<void> {
    await this.page.keyboard.press('Tab');
  }

  // Focus management
  async focusElement(locator: Locator): Promise<void> {
    await locator.focus();
  }

  // Scroll utilities
  async scrollToTop(): Promise<void> {
    await this.page.evaluate(() => window.scrollTo(0, 0));
  }

  async scrollToBottom(): Promise<void> {
    await this.page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  }

  async scrollIntoView(locator: Locator): Promise<void> {
    await locator.scrollIntoViewIfNeeded();
  }

  // Wait for specific network request
  async waitForApiResponse(urlPattern: string | RegExp): Promise<void> {
    await this.page.waitForResponse(urlPattern);
  }

  // Click with retry
  async clickWithRetry(locator: Locator, maxRetries = 3): Promise<void> {
    for (let i = 0; i < maxRetries; i++) {
      try {
        await locator.click({ timeout: 5000 });
        return;
      } catch (error) {
        if (i === maxRetries - 1) throw error;
        await this.page.waitForTimeout(500);
      }
    }
  }

  // Fill with clear
  async fillInput(locator: Locator, value: string): Promise<void> {
    await locator.clear();
    await locator.fill(value);
  }

  // Get text content safely
  async getText(locator: Locator): Promise<string> {
    return (await locator.textContent()) || '';
  }

  // Check if element is visible
  async isVisible(locator: Locator): Promise<boolean> {
    try {
      await expect(locator).toBeVisible({ timeout: 1000 });
      return true;
    } catch {
      return false;
    }
  }
}
