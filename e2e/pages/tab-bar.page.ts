import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Page object for the Tab Bar
 */
export class TabBarPage extends BasePage {
  // Tab bar container
  get tabBar(): Locator {
    return this.page.locator('.tab-bar, [class*="tab-bar"]');
  }

  // All tabs
  get tabs(): Locator {
    return this.tabBar.locator('.tab');
  }

  // Active tab
  get activeTab(): Locator {
    return this.tabBar.locator('.tab.active, .tab[class*="active"]');
  }

  // New query button (in tab bar)
  get newTabButton(): Locator {
    return this.tabBar.locator('.tab-actions button, .new-tab');
  }

  // Get specific tab by title (supports partial matching for schema.table format)
  getTab(title: string): Locator {
    // First try exact match, then try contains match (for schema.table format)
    return this.tabBar.locator('.tab').filter({
      has: this.page.locator(`.tab-title:has-text("${title}")`),
    });
  }

  getTabByIndex(index: number): Locator {
    return this.tabs.nth(index);
  }

  getTabCloseButton(title: string): Locator {
    return this.getTab(title).locator('.tab-close, [class*="close"]');
  }

  getTabIcon(title: string): Locator {
    return this.getTab(title).locator('.tab-icon, [class*="icon"]');
  }

  getTabPinIndicator(title: string): Locator {
    return this.getTab(title).locator('.tab-pin, [class*="pin"]');
  }

  // Actions
  async clickTab(title: string): Promise<void> {
    await this.getTab(title).click();
  }

  async closeTab(title: string): Promise<void> {
    // Hover to reveal close button if needed
    await this.getTab(title).hover();
    await this.getTabCloseButton(title).click();
  }

  async closeTabByMiddleClick(title: string): Promise<void> {
    await this.getTab(title).click({ button: 'middle' });
  }

  async doubleClickTab(title: string): Promise<void> {
    await this.getTab(title).dblclick();
  }

  async pinTab(title: string): Promise<void> {
    // Double-click to pin
    await this.doubleClickTab(title);
  }

  async unpinTab(title: string): Promise<void> {
    // Double-click to unpin
    await this.doubleClickTab(title);
  }

  async openNewTab(): Promise<void> {
    await this.newTabButton.click();
  }

  async closeAllTabs(): Promise<void> {
    // Close tabs one by one, always clicking the first available close button
    let count = await this.getTabCount();
    while (count > 0) {
      const closeButton = this.tabBar.locator('.tab .tab-close').first();
      if (await closeButton.isVisible()) {
        await closeButton.click();
        await this.page.waitForTimeout(100);
      }
      count = await this.getTabCount();
    }
  }

  async closeOtherTabs(keepTitle: string): Promise<void> {
    const count = await this.getTabCount();
    for (let i = count - 1; i >= 0; i--) {
      const tab = this.getTabByIndex(i);
      const title = await tab.locator('.tab-title').textContent();
      if (title && !title.includes(keepTitle)) {
        await tab.hover();
        await tab.locator('.tab-close, [class*="close"]').click();
      }
    }
  }

  async getTabCount(): Promise<number> {
    return await this.tabs.count();
  }

  async getActiveTabTitle(): Promise<string> {
    const titleElement = this.activeTab.locator('.tab-title, [class*="tab-title"]');
    return (await titleElement.textContent()) || '';
  }

  async getTabTitles(): Promise<string[]> {
    const titles: string[] = [];
    const count = await this.getTabCount();
    for (let i = 0; i < count; i++) {
      const tab = this.getTabByIndex(i);
      const title = await tab.locator('.tab-title').textContent();
      if (title) titles.push(title.trim());
    }
    return titles;
  }

  // Assertions
  async expectTabExists(title: string): Promise<void> {
    await expect(this.getTab(title)).toBeVisible();
  }

  async expectTabNotExists(title: string): Promise<void> {
    await expect(this.getTab(title)).not.toBeVisible();
  }

  async expectTabActive(title: string): Promise<void> {
    await expect(this.getTab(title)).toHaveClass(/active/);
  }

  async expectTabPinned(title: string): Promise<void> {
    await expect(this.getTabPinIndicator(title)).toBeVisible();
  }

  async expectTabNotPinned(title: string): Promise<void> {
    await expect(this.getTabPinIndicator(title)).not.toBeVisible();
  }

  async expectTabCount(count: number): Promise<void> {
    await expect(this.tabs).toHaveCount(count);
  }

  async expectNoTabs(): Promise<void> {
    await expect(this.tabs).toHaveCount(0);
  }

  async expectActiveTabTitle(title: string): Promise<void> {
    await expect(this.activeTab.locator('.tab-title')).toContainText(title);
  }
}
