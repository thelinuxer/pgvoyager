import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Page object for the Saved Queries Panel
 */
export class SavedQueriesPanel extends BasePage {
  get panel(): Locator {
    return this.page.locator('.panel-backdrop:has(h2:has-text("Saved Queries"))');
  }

  get panelContent(): Locator {
    return this.panel.locator('.panel');
  }

  get closeButton(): Locator {
    return this.panelContent.locator('.panel-close');
  }

  get searchInput(): Locator {
    return this.panelContent.locator('.search-wrapper input');
  }

  get currentDbFilterButton(): Locator {
    return this.panelContent.locator('.filter-btn:has-text("Current DB")');
  }

  get allFilterButton(): Locator {
    return this.panelContent.locator('.filter-btn:has-text("All")');
  }

  get queriesList(): Locator {
    return this.panelContent.locator('.queries-list');
  }

  get queryItems(): Locator {
    return this.panelContent.locator('.query-item');
  }

  get emptyState(): Locator {
    return this.panelContent.locator('.empty-state');
  }

  get queriesCount(): Locator {
    return this.panelContent.locator('.queries-count');
  }

  getQueryItem(name: string): Locator {
    return this.queryItems.filter({ hasText: name });
  }

  async close(): Promise<void> {
    await this.closeButton.click();
  }

  async searchFor(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(300);
  }

  async filterByCurrentDb(): Promise<void> {
    await this.currentDbFilterButton.click();
  }

  async filterByAll(): Promise<void> {
    await this.allFilterButton.click();
  }

  async clickQuery(name: string): Promise<void> {
    await this.getQueryItem(name).click();
  }

  async deleteQuery(name: string): Promise<void> {
    const item = this.getQueryItem(name);
    await item.hover();
    await item.locator('.action-btn.danger').click();
    // Handle confirmation dialog
    this.page.once('dialog', dialog => dialog.accept());
  }

  async editQuery(name: string): Promise<void> {
    const item = this.getQueryItem(name);
    await item.hover();
    await item.locator('.action-btn:not(.danger)').click();
  }

  async expectVisible(): Promise<void> {
    await expect(this.panelContent).toBeVisible();
  }

  async expectClosed(): Promise<void> {
    await expect(this.panel).not.toBeVisible();
  }

  async expectQueryCount(count: number): Promise<void> {
    await expect(this.queriesCount).toContainText(`${count}`);
  }

  async expectQueryVisible(name: string): Promise<void> {
    await expect(this.getQueryItem(name)).toBeVisible();
  }

  async expectEmpty(): Promise<void> {
    await expect(this.emptyState).toBeVisible();
  }
}

/**
 * Page object for the Query History Panel
 */
export class QueryHistoryPanel extends BasePage {
  get panel(): Locator {
    // The panel backdrop contains the panel with "Query History" heading
    return this.page.locator('.panel-backdrop:has(h2:has-text("Query History"))');
  }

  get panelContent(): Locator {
    return this.panel.locator('.panel');
  }

  get panelBackdrop(): Locator {
    return this.page.locator('.panel-backdrop');
  }

  get closeButton(): Locator {
    return this.panelContent.locator('.panel-close');
  }

  get searchInput(): Locator {
    return this.panelContent.locator('.search-wrapper input');
  }

  get currentDbFilterButton(): Locator {
    return this.panelContent.locator('.filter-btn:has-text("Current DB")');
  }

  get allFilterButton(): Locator {
    return this.panelContent.locator('.filter-btn:has-text("All")');
  }

  get historyList(): Locator {
    return this.panelContent.locator('.history-list');
  }

  get historyItems(): Locator {
    return this.panelContent.locator('.history-item');
  }

  get emptyState(): Locator {
    return this.panelContent.locator('.empty-state');
  }

  get historyCount(): Locator {
    return this.panelContent.locator('.history-count');
  }

  get clearButton(): Locator {
    return this.panelContent.locator('.panel-footer button:has-text("Clear")');
  }

  getHistoryItem(sql: string): Locator {
    return this.historyItems.filter({ hasText: sql });
  }

  async close(): Promise<void> {
    await this.closeButton.click();
  }

  async searchFor(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(300);
  }

  async filterByCurrentDb(): Promise<void> {
    await this.currentDbFilterButton.click();
  }

  async filterByAll(): Promise<void> {
    await this.allFilterButton.click();
  }

  async clickHistoryEntry(sql: string): Promise<void> {
    await this.getHistoryItem(sql).click();
  }

  async deleteHistoryEntry(sql: string): Promise<void> {
    const item = this.getHistoryItem(sql);
    await item.hover();
    await item.locator('.history-delete').click();
  }

  async clearHistory(): Promise<void> {
    await this.clearButton.click();
  }

  async expectVisible(): Promise<void> {
    // Wait for the panel backdrop to appear first with extended timeout
    await expect(this.panelBackdrop).toBeVisible({ timeout: 5000 });
    // Then verify the panel content is visible
    await expect(this.panelContent).toBeVisible({ timeout: 5000 });
  }

  async expectClosed(): Promise<void> {
    await expect(this.panelBackdrop).not.toBeVisible({ timeout: 5000 });
  }

  async expectHistoryCount(count: number): Promise<void> {
    await expect(this.historyCount).toContainText(`${count}`);
  }

  async expectHistoryVisible(sql: string): Promise<void> {
    await expect(this.getHistoryItem(sql)).toBeVisible();
  }

  async expectEmpty(): Promise<void> {
    await expect(this.emptyState).toBeVisible();
  }
}

/**
 * Page object for the Settings Modal
 */
export class SettingsModal extends BasePage {
  get modal(): Locator {
    return this.page.locator('.modal-overlay:has(h2:has-text("Settings"))');
  }

  get modalContent(): Locator {
    return this.modal.locator('.modal');
  }

  get closeButton(): Locator {
    return this.modalContent.locator('.close-btn');
  }

  get themeSection(): Locator {
    return this.modalContent.locator('.settings-section:has(h3:has-text("Theme"))');
  }

  get iconStyleSection(): Locator {
    return this.modalContent.locator('.settings-section:has(h3:has-text("Icon"))');
  }

  get themeCards(): Locator {
    return this.themeSection.locator('.theme-card');
  }

  get iconLibraryCards(): Locator {
    return this.iconStyleSection.locator('.icon-library-card');
  }

  getThemeCard(themeName: string): Locator {
    return this.themeCards.filter({ hasText: themeName });
  }

  getIconLibraryCard(libraryName: string): Locator {
    return this.iconLibraryCards.filter({ hasText: libraryName });
  }

  async close(): Promise<void> {
    await this.closeButton.click();
  }

  async selectTheme(themeName: string): Promise<void> {
    await this.getThemeCard(themeName).click();
  }

  async selectIconLibrary(libraryName: string): Promise<void> {
    await this.getIconLibraryCard(libraryName).click();
  }

  async expectVisible(): Promise<void> {
    await expect(this.modalContent).toBeVisible();
  }

  async expectClosed(): Promise<void> {
    await expect(this.modal).not.toBeVisible();
  }

  async expectThemeSelected(themeName: string): Promise<void> {
    await expect(this.getThemeCard(themeName)).toHaveClass(/selected/);
  }

  async expectIconLibrarySelected(libraryName: string): Promise<void> {
    await expect(this.getIconLibraryCard(libraryName)).toHaveClass(/selected/);
  }
}

/**
 * Page object for the ERD Viewer
 */
export class ERDViewerPage extends BasePage {
  get erdContainer(): Locator {
    // Be specific to avoid matching .erd-title which is a child element
    return this.page.locator('.erd-viewer').first();
  }

  get canvas(): Locator {
    // Cytoscape uses a .graph-container div
    return this.erdContainer.locator('.graph-container');
  }

  get toolbar(): Locator {
    // Target the main toolbar element, not the left/right subdivisions
    return this.erdContainer.locator('.toolbar').first();
  }

  get backButton(): Locator {
    return this.toolbar.locator('button:has-text("Back"), button[title*="Back"]');
  }

  get forwardButton(): Locator {
    return this.toolbar.locator('button:has-text("Forward"), button[title*="Forward"]');
  }

  get fullSchemaButton(): Locator {
    return this.toolbar.locator('button:has-text("Full Schema"), button[title*="Full"]');
  }

  get exportButton(): Locator {
    return this.toolbar.locator('button:has-text("Export"), button[title*="Export"]');
  }

  get loadingIndicator(): Locator {
    return this.erdContainer.locator('.loading, [class*="loading"]');
  }

  get errorMessage(): Locator {
    return this.erdContainer.locator('.error, [class*="error"]');
  }

  async waitForLoad(): Promise<void> {
    // Wait for the graph container to be visible
    await expect(this.canvas).toBeVisible({ timeout: 15000 });
  }

  async goBack(): Promise<void> {
    await this.backButton.click();
  }

  async goForward(): Promise<void> {
    await this.forwardButton.click();
  }

  async viewFullSchema(): Promise<void> {
    await this.fullSchemaButton.click();
  }

  async export(): Promise<void> {
    await this.exportButton.click();
  }

  async expectVisible(): Promise<void> {
    await expect(this.erdContainer).toBeVisible();
  }

  async expectCanvasVisible(): Promise<void> {
    await expect(this.canvas).toBeVisible();
  }
}

/**
 * Page object for the Analysis Viewer
 */
export class AnalysisViewerPage extends BasePage {
  get analysisContainer(): Locator {
    return this.page.locator('.analysis-viewer, [class*="analysis"]');
  }

  get runButton(): Locator {
    return this.analysisContainer.locator('button:has-text("Run"), button:has-text("Analyze")');
  }

  get issuesList(): Locator {
    return this.analysisContainer.locator('.issues-list, [class*="issues"]');
  }

  get issueItems(): Locator {
    return this.issuesList.locator('.issue-item, [class*="issue-item"]');
  }

  get loadingIndicator(): Locator {
    return this.analysisContainer.locator('.loading, [class*="loading"]');
  }

  get emptyState(): Locator {
    return this.analysisContainer.locator('.empty-state, [class*="empty"]');
  }

  async waitForLoad(): Promise<void> {
    await expect(this.loadingIndicator).not.toBeVisible({ timeout: 30000 });
  }

  async runAnalysis(): Promise<void> {
    await this.runButton.click();
    await this.waitForLoad();
  }

  async expectVisible(): Promise<void> {
    await expect(this.analysisContainer).toBeVisible();
  }

  async expectIssuesVisible(): Promise<void> {
    await expect(this.issuesList).toBeVisible();
  }
}
