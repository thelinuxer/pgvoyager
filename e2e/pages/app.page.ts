import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';
import { ConnectionModalPage, ConnectionConfig } from './connection-modal.page';
import { SidebarPage } from './sidebar.page';
import { TabBarPage } from './tab-bar.page';
import { QueryEditorPage } from './query-editor.page';
import { TableViewerPage } from './table-viewer.page';

/**
 * Main application page object - entry point for all E2E tests
 */
export class AppPage extends BasePage {
  // Child page objects
  readonly connectionModal: ConnectionModalPage;
  readonly sidebar: SidebarPage;
  readonly tabBar: TabBarPage;
  readonly queryEditor: QueryEditorPage;
  readonly tableViewer: TableViewerPage;

  constructor(page: Page) {
    super(page);
    this.connectionModal = new ConnectionModalPage(page);
    this.sidebar = new SidebarPage(page);
    this.tabBar = new TabBarPage(page);
    this.queryEditor = new QueryEditorPage(page);
    this.tableViewer = new TableViewerPage(page);
  }

  // Header elements
  get header(): Locator {
    return this.page.locator('header, [class*="header"]').first();
  }

  get logo(): Locator {
    return this.header.locator('img[alt*="PgVoyager"], .logo');
  }

  get versionBadge(): Locator {
    return this.header.locator('.version, [class*="version"]');
  }

  get newConnectionButton(): Locator {
    return this.page.getByRole('button', { name: /new connection/i });
  }

  get settingsButton(): Locator {
    return this.page.getByRole('button', { name: /settings/i });
  }

  get resetLayoutButton(): Locator {
    return this.page.getByRole('button', { name: /reset layout/i });
  }

  get connectionDropdown(): Locator {
    return this.page.getByRole('button', { name: /select connection|connection/i }).first();
  }

  get claudeTerminalToggle(): Locator {
    return this.page.locator('[title*="Claude"], [aria-label*="Claude"]');
  }

  get githubLink(): Locator {
    return this.page.getByRole('link', { name: /github/i });
  }

  // Welcome screen elements
  get welcomeScreen(): Locator {
    return this.page.locator('.welcome').first();
  }

  get welcomeLogo(): Locator {
    return this.welcomeScreen.locator('img, .logo');
  }

  get welcomeNewConnectionButton(): Locator {
    return this.welcomeScreen.getByRole('button', { name: /new connection/i });
  }

  // Main content area
  get contentArea(): Locator {
    return this.page.locator('.content-area, main, [class*="content"]');
  }

  // Navigation
  async goto(): Promise<void> {
    await this.page.goto('/');
    await this.waitForLoad();
  }

  async reload(): Promise<void> {
    await this.page.reload();
    await this.waitForLoad();
  }

  // Connection actions
  async openNewConnectionModal(): Promise<void> {
    // Try header button first, then welcome screen button
    const headerButton = this.newConnectionButton;
    const welcomeButton = this.welcomeNewConnectionButton;

    if (await this.isVisible(headerButton)) {
      await headerButton.click();
    } else if (await this.isVisible(welcomeButton)) {
      await welcomeButton.click();
    } else {
      throw new Error('No "New Connection" button found');
    }

    await this.connectionModal.waitForOpen();
  }

  async createConnection(config: ConnectionConfig, maxRetries = 5): Promise<void> {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      await this.openNewConnectionModal();
      await this.connectionModal.fillConnectionForm(config);
      await this.connectionModal.saveAndConnectButton.click();

      // Wait a bit for the connection attempt
      await this.page.waitForTimeout(1500);

      // Check if modal is still visible (indicates an error)
      const modalVisible = await this.connectionModal.modal.isVisible();
      if (!modalVisible) {
        // Connection succeeded, modal closed
        return;
      }

      // Check for connection error (too many clients)
      const errorText = await this.connectionModal.modal.textContent();
      if (errorText?.includes('too many clients')) {
        console.log(`Connection attempt ${attempt} failed: too many clients. Retrying...`);
        // Cancel and wait for connections to be released
        await this.connectionModal.cancel();
        // Exponential backoff: wait longer with each retry
        const waitTime = 5000 * attempt; // 5s, 10s, 15s, 20s, 25s
        await this.page.waitForTimeout(waitTime);
        continue;
      }

      // Some other error - wait longer for modal to close
      try {
        await this.connectionModal.waitForClose();
        return;
      } catch {
        // If still failing, throw the error
        if (attempt === maxRetries) {
          throw new Error(`Failed to create connection after ${maxRetries} attempts. Last error: ${errorText}`);
        }
        await this.connectionModal.cancel();
        await this.page.waitForTimeout(3000);
      }
    }
  }

  async openConnectionDropdown(): Promise<void> {
    await this.connectionDropdown.click();
    await this.page.waitForTimeout(300); // Wait for dropdown animation
  }

  async selectConnection(name: string): Promise<void> {
    await this.openConnectionDropdown();
    await this.page.getByRole('button', { name: new RegExp(name, 'i') }).click();
    await this.sidebar.waitForSchemaLoad();
  }

  async disconnectFromDatabase(): Promise<void> {
    await this.openConnectionDropdown();
    await this.page.getByRole('button', { name: /disconnect/i }).click();
  }

  // Settings
  async openSettings(): Promise<void> {
    await this.settingsButton.click();
    await this.page.locator('.modal').filter({ hasText: /settings/i }).waitFor();
  }

  // Claude Terminal
  async toggleClaudeTerminal(): Promise<void> {
    await this.claudeTerminalToggle.click();
  }

  async openClaudeTerminal(): Promise<void> {
    const isOpen = await this.isClaudeTerminalOpen();
    if (!isOpen) {
      await this.toggleClaudeTerminal();
    }
  }

  async closeClaudeTerminal(): Promise<void> {
    const isOpen = await this.isClaudeTerminalOpen();
    if (isOpen) {
      await this.toggleClaudeTerminal();
    }
  }

  async isClaudeTerminalOpen(): Promise<boolean> {
    return this.isVisible(this.page.locator('.claude-terminal, .terminal-panel'));
  }

  // Layout
  async resetLayout(): Promise<void> {
    await this.resetLayoutButton.click();
  }

  // Assertions
  async expectWelcomeScreen(): Promise<void> {
    await expect(this.welcomeScreen).toBeVisible();
    await expect(this.welcomeLogo).toBeVisible();
  }

  async expectNoWelcomeScreen(): Promise<void> {
    await expect(this.welcomeScreen).not.toBeVisible();
  }

  async expectConnected(connectionName?: string): Promise<void> {
    await expect(this.welcomeScreen).not.toBeVisible({ timeout: 10000 });
    await expect(this.sidebar.schemaTree).toBeVisible({ timeout: 10000 });

    if (connectionName) {
      await expect(this.connectionDropdown).toContainText(connectionName);
    }
  }

  async expectDisconnected(): Promise<void> {
    await expect(this.welcomeScreen).toBeVisible();
  }

  async expectTitle(title: string): Promise<void> {
    await expect(this.page).toHaveTitle(title);
  }

  async expectVersion(version: string): Promise<void> {
    await expect(this.versionBadge).toContainText(version);
  }

  // Utility: Connect to test database, reusing existing connection if available
  async connectToTestDatabase(config: ConnectionConfig): Promise<void> {
    // Check if already connected
    const welcomeVisible = await this.isVisible(this.welcomeScreen);

    if (!welcomeVisible) {
      // Already connected - check if it's the right connection
      const dropdownText = await this.connectionDropdown.textContent();
      if (dropdownText?.includes(config.name)) {
        // Already connected to the right database
        await this.sidebar.waitForSchemaLoad();
        return;
      }

      // Try to find and select existing connection from dropdown
      await this.openConnectionDropdown();
      const existingConnection = this.page.locator('.dropdown-menu, [role="menu"]')
        .getByText(config.name, { exact: false });

      if (await this.isVisible(existingConnection)) {
        await existingConnection.click();
        await this.sidebar.waitForSchemaLoad();
        return;
      }

      // Close dropdown if no matching connection found
      await this.page.keyboard.press('Escape');
    }

    // Create new connection through the UI
    // This is more reliable than API shortcuts
    await this.createConnection(config);
    await this.expectConnected(config.name);
    await this.sidebar.waitForSchemaLoad();
  }
}
