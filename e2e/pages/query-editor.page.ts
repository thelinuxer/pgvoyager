import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Page object for the Query Editor
 */
export class QueryEditorPage extends BasePage {
  // Editor container - using data-testid
  get editor(): Locator {
    return this.page.locator('[data-testid="query-editor"]');
  }

  // CodeMirror editor
  get codeEditor(): Locator {
    return this.page.locator('[data-testid="editor-container"] .cm-editor');
  }

  get codeContent(): Locator {
    return this.codeEditor.locator('.cm-content');
  }

  // Toolbar buttons - using data-testid
  get toolbar(): Locator {
    return this.page.locator('[data-testid="editor-toolbar"]');
  }

  get runButton(): Locator {
    return this.page.locator('[data-testid="btn-run-query"]');
  }

  get saveButton(): Locator {
    return this.page.locator('[data-testid="btn-save-query"]');
  }

  get formatButton(): Locator {
    return this.editor.getByRole('button', { name: /format/i });
  }

  get clearButton(): Locator {
    return this.editor.getByRole('button', { name: /clear/i });
  }

  // Results section - using data-testid
  get resultsSection(): Locator {
    return this.page.locator('[data-testid="results-section"]');
  }

  get resultsTable(): Locator {
    return this.page.locator('[data-testid="results-table"] table');
  }

  get resultsTableHeaders(): Locator {
    return this.resultsTable.locator('thead th');
  }

  get resultsTableRows(): Locator {
    // Exclude empty-row and loading-row rows from the count
    return this.resultsTable.locator('tbody tr:not(:has(.empty-row)):not(:has(.loading-row))');
  }

  get resultsHeader(): Locator {
    return this.page.locator('[data-testid="results-header"]');
  }

  get rowCountDisplay(): Locator {
    return this.page.locator('[data-testid="row-count"]');
  }

  get executionTimeDisplay(): Locator {
    return this.page.locator('[data-testid="execution-time"]');
  }

  get exportCsvButton(): Locator {
    return this.page.locator('[data-testid="btn-export-csv"]');
  }

  // Loading and status
  get resultsLoading(): Locator {
    return this.page.locator('[data-testid="results-loading"]');
  }

  get resultsEmpty(): Locator {
    return this.resultsSection.locator('.results-empty');
  }

  get resultsError(): Locator {
    return this.page.locator('[data-testid="results-error"]');
  }

  get errorMessage(): Locator {
    return this.resultsError.locator('pre');
  }

  // Syntax highlighting and autocomplete
  get autocompletePopup(): Locator {
    return this.page.locator('.cm-tooltip-autocomplete');
  }

  get errorHighlight(): Locator {
    return this.codeEditor.locator('.cm-error-highlight');
  }

  // Actions
  async focus(): Promise<void> {
    await this.codeContent.click();
  }

  async setQuery(sql: string): Promise<void> {
    // Wait for editor to be ready
    await this.codeEditor.waitFor({ state: 'visible' });

    // Wait for the test helper to be available (it's set up in handleEditorReady)
    await this.page.waitForFunction(
      () => (window as any).__PGVOYAGER_E2E__?.setQuery,
      { timeout: 5000 }
    );

    // Use the exposed test helper to set the query
    // This properly updates both CodeMirror's visual state AND Svelte's bound variable
    await this.page.evaluate((query) => {
      const win = window as any;
      win.__PGVOYAGER_E2E__.setQuery(query);
    }, sql);

    // Wait for the change to propagate
    await this.page.waitForTimeout(50);

    // Verify the text was set correctly
    await expect(this.codeContent).toContainText(sql.slice(0, 20));
  }

  async appendQuery(sql: string): Promise<void> {
    await this.focus();
    await this.page.keyboard.press('End');
    await this.page.keyboard.type(sql, { delay: 10 });
  }

  async clearQuery(): Promise<void> {
    await this.focus();
    await this.page.keyboard.press('Control+a');
    await this.page.keyboard.press('Delete');
  }

  async getQueryText(): Promise<string> {
    return (await this.codeContent.textContent()) || '';
  }

  async executeQuery(): Promise<void> {
    // Check if the run button is enabled
    const isEnabled = await this.runButton.isEnabled().catch(() => false);

    if (isEnabled) {
      await this.runButton.click();
    } else {
      // If button is disabled (stuck "Executing" state from previous test),
      // use keyboard shortcut which can still work
      await this.focus();
      await this.pressCtrlEnter();
    }
    await this.waitForQueryExecution();
  }

  async executeWithKeyboard(): Promise<void> {
    await this.focus();
    await this.pressCtrlEnter();
    await this.waitForQueryExecution();
  }

  async waitForQueryExecution(timeout = 30000): Promise<void> {
    // Wait for loading to start (button may change text)
    await this.page.waitForTimeout(100);

    // Wait for loading to finish
    try {
      await expect(this.runButton).not.toContainText(/running|executing/i, { timeout });
    } catch {
      // Continue if button text doesn't change
    }

    // Wait for results loading to disappear
    try {
      await expect(this.resultsLoading).not.toBeVisible({ timeout });
    } catch {
      // Continue if no loading indicator
    }
  }

  async saveQuery(): Promise<void> {
    await this.saveButton.click();
  }

  async saveWithKeyboard(): Promise<void> {
    await this.focus();
    await this.pressCtrlS();
  }

  async exportToCsv(): Promise<void> {
    await this.exportCsvButton.click();
  }

  async triggerAutocomplete(): Promise<void> {
    await this.focus();
    await this.page.keyboard.press('Control+Space');
  }

  async selectAutocompleteItem(index: number = 0): Promise<void> {
    for (let i = 0; i < index; i++) {
      await this.page.keyboard.press('ArrowDown');
    }
    await this.pressEnter();
  }

  // Result getters
  async getResultRowCount(): Promise<number> {
    const text = await this.rowCountDisplay.textContent();
    const match = text?.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  async getExecutionTime(): Promise<string> {
    return (await this.executionTimeDisplay.textContent()) || '';
  }

  async getResultsColumnNames(): Promise<string[]> {
    // Get just the column names, not the data types (which are in .col-type)
    const colNames = this.resultsTable.locator('thead th .col-name');
    const names = await colNames.allTextContents();
    return names.map((h) => h.trim());
  }

  async getResultsCellValue(row: number, column: string): Promise<string> {
    const headers = await this.getResultsColumnNames();
    const colIndex = headers.findIndex((h) => h.includes(column));
    if (colIndex === -1) throw new Error(`Column "${column}" not found`);

    const cell = this.resultsTableRows.nth(row - 1).locator(`td:nth-child(${colIndex + 1})`);
    return (await cell.textContent()) || '';
  }

  async getResultsRowValues(row: number): Promise<string[]> {
    const cells = this.resultsTableRows.nth(row - 1).locator('td');
    return await cells.allTextContents();
  }

  async getErrorMessageText(): Promise<string> {
    return (await this.resultsError.textContent()) || '';
  }

  // Assertions
  async expectResults(): Promise<void> {
    await expect(this.resultsTable).toBeVisible();
  }

  async expectNoResults(): Promise<void> {
    await expect(this.resultsEmpty).toBeVisible();
  }

  async expectError(message?: string): Promise<void> {
    await expect(this.resultsError).toBeVisible();
    if (message) {
      await expect(this.resultsError).toContainText(message);
    }
  }

  async expectNoError(): Promise<void> {
    await expect(this.resultsError).not.toBeVisible();
  }

  async expectRowCount(count: number): Promise<void> {
    await expect(this.rowCountDisplay).toContainText(String(count));
  }

  async expectExecutionTime(): Promise<void> {
    await expect(this.executionTimeDisplay).toBeVisible();
    await expect(this.executionTimeDisplay).toContainText(/\d+\s*(ms|s)/);
  }

  async expectColumnExists(columnName: string): Promise<void> {
    await expect(this.resultsTableHeaders.filter({ hasText: columnName })).toBeVisible();
  }

  async expectAutocompleteVisible(): Promise<void> {
    await expect(this.autocompletePopup).toBeVisible();
  }

  async expectAutocompleteHidden(): Promise<void> {
    await expect(this.autocompletePopup).not.toBeVisible();
  }

  async expectSyntaxHighlighting(): Promise<void> {
    // Check for CodeMirror syntax classes
    await expect(this.codeEditor.locator('.cm-keyword, .cm-sql-keyword')).toBeVisible();
  }

  // FK-related locators
  get fkIcons(): Locator {
    return this.resultsTable.locator('[title="Foreign Key"], .fk-icon');
  }

  get pkIcons(): Locator {
    return this.resultsTable.locator('[title="Primary Key"], .pk-icon');
  }

  get fkCells(): Locator {
    return this.resultsTable.locator('td.fk-column');
  }

  get fkPreviewPopup(): Locator {
    return this.page.locator('.fk-popup');
  }

  // FK-related methods
  async getFKIconCount(): Promise<number> {
    return await this.fkIcons.count();
  }

  async getPKIconCount(): Promise<number> {
    return await this.pkIcons.count();
  }

  async getFKCellCount(): Promise<number> {
    return await this.fkCells.count();
  }

  async hoverFKCell(index: number = 0): Promise<void> {
    const cell = this.fkCells.nth(index);
    await cell.hover();
    // Wait for preview popup to appear
    await this.page.waitForTimeout(500);
  }

  async clickFKCell(index: number = 0): Promise<void> {
    const cell = this.fkCells.nth(index);
    await cell.click();
  }

  async getFKCellValue(index: number = 0): Promise<string> {
    const cell = this.fkCells.nth(index);
    return (await cell.textContent()) || '';
  }

  async expectFKIcon(): Promise<void> {
    await expect(this.fkIcons.first()).toBeVisible();
  }

  async expectPKIcon(): Promise<void> {
    await expect(this.pkIcons.first()).toBeVisible();
  }

  async expectFKPreviewPopup(): Promise<void> {
    await expect(this.fkPreviewPopup).toBeVisible({ timeout: 3000 });
  }

  async expectNoFKPreviewPopup(): Promise<void> {
    await expect(this.fkPreviewPopup).not.toBeVisible({ timeout: 3000 });
  }

  async expectFKCellsClickable(): Promise<void> {
    const firstFkCell = this.fkCells.first();
    await expect(firstFkCell).toBeVisible();
    // FK cells should have cursor: pointer style
    const cursor = await firstFkCell.evaluate((el) => getComputedStyle(el).cursor);
    expect(cursor).toBe('pointer');
  }
}
