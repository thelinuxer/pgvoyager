import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

export interface ConnectionConfig {
  name: string;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  sslMode?: 'disable' | 'prefer' | 'require' | 'verify-ca' | 'verify-full';
}

/**
 * Page object for the Connection Modal
 */
export class ConnectionModalPage extends BasePage {
  // Modal container
  get modal(): Locator {
    return this.page.locator('[data-testid="connection-modal"]');
  }

  get isOpen(): Promise<boolean> {
    return this.isVisible(this.modal);
  }

  // Form fields - using data-testid for reliable selection
  get nameInput(): Locator {
    return this.page.locator('[data-testid="input-name"]');
  }

  get hostInput(): Locator {
    return this.page.locator('[data-testid="input-host"]');
  }

  get portInput(): Locator {
    return this.page.locator('[data-testid="input-port"]');
  }

  get databaseInput(): Locator {
    return this.page.locator('[data-testid="input-database"]');
  }

  get usernameInput(): Locator {
    return this.page.locator('[data-testid="input-username"]');
  }

  get passwordInput(): Locator {
    return this.page.locator('[data-testid="input-password"]');
  }

  get sslModeSelect(): Locator {
    return this.page.locator('[data-testid="select-sslmode"]');
  }

  // Buttons - using data-testid for reliable selection
  get testConnectionButton(): Locator {
    return this.page.locator('[data-testid="btn-test-connection"]');
  }

  get saveButton(): Locator {
    return this.page.locator('[data-testid="btn-save"]');
  }

  get saveAndConnectButton(): Locator {
    return this.page.locator('[data-testid="btn-save"]');
  }

  get cancelButton(): Locator {
    return this.page.locator('[data-testid="btn-cancel"]');
  }

  get closeButton(): Locator {
    return this.page.locator('[data-testid="modal-close"]');
  }

  get deleteButton(): Locator {
    return this.page.locator('[data-testid="btn-delete"]');
  }

  // Status indicators
  get testResultMessage(): Locator {
    return this.modal.locator('.test-result');
  }

  get errorMessage(): Locator {
    return this.modal.locator('.error-message, .test-result.error');
  }

  get successMessage(): Locator {
    return this.modal.locator('.test-result.success');
  }

  // Actions
  async waitForOpen(): Promise<void> {
    await expect(this.modal).toBeVisible({ timeout: 5000 });
  }

  async waitForClose(): Promise<void> {
    await expect(this.modal).not.toBeVisible({ timeout: 5000 });
  }

  async fillConnectionForm(config: ConnectionConfig): Promise<void> {
    await this.fillInput(this.nameInput, config.name);
    await this.fillInput(this.hostInput, config.host);
    await this.fillInput(this.portInput, String(config.port));
    await this.fillInput(this.databaseInput, config.database);
    await this.fillInput(this.usernameInput, config.username);
    await this.fillInput(this.passwordInput, config.password);

    if (config.sslMode) {
      await this.sslModeSelect.selectOption(config.sslMode);
    }
  }

  async testConnection(): Promise<boolean> {
    await this.testConnectionButton.click();

    // Wait for test to complete (button text changes or result appears)
    await this.page.waitForTimeout(500); // Brief wait for request to start

    // Wait for the test to complete
    try {
      await expect(this.testConnectionButton).not.toContainText(/testing/i, { timeout: 15000 });
    } catch {
      // Continue even if timeout
    }

    // Check if there's an error message visible
    const hasError = await this.isVisible(this.errorMessage);
    return !hasError;
  }

  async save(): Promise<void> {
    await this.saveButton.click();
    await this.waitForClose();
  }

  async saveAndConnect(): Promise<void> {
    await this.saveAndConnectButton.click();
    // Wait for modal to close and connection to be established
    await this.waitForClose();
  }

  async cancel(): Promise<void> {
    await this.cancelButton.click();
    await this.waitForClose();
  }

  async close(): Promise<void> {
    await this.closeButton.click();
    await this.waitForClose();
  }

  async deleteConnection(): Promise<void> {
    // Set up dialog handler before clicking delete
    this.page.once('dialog', (dialog) => dialog.accept());
    await this.deleteButton.click();
    await this.waitForClose();
  }

  // Assertions
  async expectTestSuccess(): Promise<void> {
    await expect(this.errorMessage).not.toBeVisible();
  }

  async expectTestFailure(): Promise<void> {
    await expect(this.errorMessage).toBeVisible();
  }

  async expectErrorMessage(text: string): Promise<void> {
    await expect(this.errorMessage).toContainText(text);
  }

  async expectFormValues(config: Partial<ConnectionConfig>): Promise<void> {
    if (config.name) {
      await expect(this.nameInput).toHaveValue(config.name);
    }
    if (config.host) {
      await expect(this.hostInput).toHaveValue(config.host);
    }
    if (config.port) {
      await expect(this.portInput).toHaveValue(String(config.port));
    }
    if (config.database) {
      await expect(this.databaseInput).toHaveValue(config.database);
    }
    if (config.username) {
      await expect(this.usernameInput).toHaveValue(config.username);
    }
  }
}
