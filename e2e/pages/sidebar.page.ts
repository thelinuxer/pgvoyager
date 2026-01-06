import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Page object for the Sidebar (Schema Browser)
 */
export class SidebarPage extends BasePage {
  // Main sidebar container
  get sidebar(): Locator {
    return this.page.locator('[data-testid="sidebar"]');
  }

  // Schema tree
  get schemaTree(): Locator {
    return this.page.locator('[data-testid="schema-tree"]');
  }

  // Search
  get searchInput(): Locator {
    return this.sidebar.locator('.search-input, input[placeholder*="filter" i], input[placeholder*="search" i]');
  }

  get searchClearButton(): Locator {
    return this.sidebar.locator('.search-clear, [class*="clear"]');
  }

  // Loading state
  get loadingIndicator(): Locator {
    return this.sidebar.locator('.loading, .spinner, [class*="loading"]');
  }

  // Empty/error states
  get emptyState(): Locator {
    return this.sidebar.locator('.empty, [class*="empty"]');
  }

  get errorState(): Locator {
    return this.sidebar.locator('.error, [class*="error"]');
  }

  get noResultsMessage(): Locator {
    return this.sidebar.locator('.no-results');
  }

  // Action buttons - using data-testid
  get newQueryButton(): Locator {
    return this.page.locator('[data-testid="btn-new-query"]');
  }

  get savedQueriesButton(): Locator {
    return this.sidebar.locator('[title*="Saved Queries" i], [aria-label*="Saved Queries" i]');
  }

  get queryHistoryButton(): Locator {
    // Target the button specifically in the sidebar-actions div
    return this.sidebar.locator('.sidebar-actions button[title="Query History"]');
  }

  get analyzeButton(): Locator {
    return this.sidebar.locator('[title*="Analyze" i], [aria-label*="Analyze" i]');
  }

  get refreshSchemaButton(): Locator {
    return this.page.locator('[data-testid="btn-refresh-schema"]');
  }

  // Context menu - be specific to avoid matching backdrop
  get contextMenu(): Locator {
    return this.page.locator('.context-menu:not(.context-menu-backdrop)');
  }

  // Tree navigation helpers - use data-testid for reliable selection
  getTreeNode(name: string): Locator {
    return this.page.locator(`[data-testid="tree-button-${name}"]`);
  }

  getTreeItem(name: string): Locator {
    return this.page.locator(`[data-testid="tree-item-${name}"]`);
  }

  getSchemaNode(schemaName: string): Locator {
    return this.getTreeNode(schemaName);
  }

  getTableNode(tableName: string): Locator {
    return this.getTreeNode(tableName);
  }

  getViewNode(viewName: string): Locator {
    return this.getTreeNode(viewName);
  }

  getFunctionNode(functionName: string): Locator {
    return this.getTreeNode(functionName);
  }

  getSequenceNode(sequenceName: string): Locator {
    return this.getTreeNode(sequenceName);
  }

  getTypeNode(typeName: string): Locator {
    return this.getTreeNode(typeName);
  }

  // Context menu items
  getContextMenuItem(label: string): Locator {
    // Context menu items are buttons with class .context-menu-item
    // Escape special regex characters in the label
    const escapedLabel = label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return this.contextMenu.locator('.context-menu-item', { hasText: new RegExp(escapedLabel, 'i') }).or(
      this.contextMenu.getByRole('button', { name: new RegExp(escapedLabel, 'i') })
    );
  }

  // Actions
  async waitForSchemaLoad(timeout = 30000): Promise<void> {
    // Wait for loading to finish
    await expect(this.loadingIndicator).not.toBeVisible({ timeout });
    // Wait for tree to appear
    await expect(this.schemaTree).toBeVisible({ timeout });
  }

  async searchFor(query: string): Promise<void> {
    await this.fillInput(this.searchInput, query);
    // Wait for filter to apply
    await this.page.waitForTimeout(300);
  }

  async clearSearch(): Promise<void> {
    const clearButton = this.searchClearButton;
    if (await this.isVisible(clearButton)) {
      await clearButton.click();
    } else {
      await this.searchInput.clear();
      await this.pressEscape();
    }
  }

  async expandNode(nodeName: string): Promise<void> {
    const node = this.getTreeNode(nodeName);
    // Wait for node to be visible first
    await node.waitFor({ state: 'visible', timeout: 10000 });

    // Check if already expanded by looking for the expanded chevron
    const chevron = node.locator('.tree-chevron.expanded');
    const isExpanded = await chevron.count() > 0;

    if (!isExpanded) {
      await node.click();
      // Wait for expansion animation
      await this.page.waitForTimeout(500);
    }
  }

  async collapseNode(nodeName: string): Promise<void> {
    const node = this.getTreeNode(nodeName);
    // Check if currently expanded
    const chevron = node.locator('.tree-chevron.expanded');
    const isExpanded = await chevron.count() > 0;

    if (isExpanded) {
      await node.click();
      await this.page.waitForTimeout(200);
    }
  }

  async toggleNode(nodeName: string): Promise<void> {
    const node = this.getTreeNode(nodeName);
    await node.click();
    await this.page.waitForTimeout(200);
  }

  async openTable(tableName: string): Promise<void> {
    await this.getTreeNode(tableName).click();
  }

  async doubleClickTable(tableName: string): Promise<void> {
    await this.getTreeNode(tableName).dblclick();
  }

  async rightClickTable(tableName: string): Promise<void> {
    await this.getTreeNode(tableName).click({ button: 'right' });
    await expect(this.contextMenu).toBeVisible();
  }

  async rightClickSchema(schemaName: string): Promise<void> {
    await this.getSchemaNode(schemaName).click({ button: 'right' });
    await expect(this.contextMenu).toBeVisible();
  }

  async clickContextMenuItem(label: string): Promise<void> {
    await this.getContextMenuItem(label).click();
    // Context menu should close
    await expect(this.contextMenu).not.toBeVisible();
  }

  async openNewQuery(): Promise<void> {
    await this.newQueryButton.click();
  }

  async openSavedQueries(): Promise<void> {
    await this.savedQueriesButton.click();
  }

  async openQueryHistory(): Promise<void> {
    // Ensure the button is visible and clickable
    const button = this.queryHistoryButton;
    await expect(button).toBeVisible();
    await expect(button).toBeEnabled();
    // Click the button
    await button.click();
    // Wait for panel animation
    await this.page.waitForTimeout(500);
  }

  async openAnalysis(): Promise<void> {
    await this.analyzeButton.click();
  }

  async refreshSchema(): Promise<void> {
    await this.refreshSchemaButton.click();
    // Wait for loading to start and then finish
    await this.page.waitForTimeout(100);
    await this.waitForSchemaLoad();
  }

  // Navigate to specific objects
  async navigateToTable(schemaName: string, tableName: string): Promise<void> {
    await this.expandNode(schemaName);
    await this.expandNode('Tables');
    await this.openTable(tableName);
  }

  async navigateToView(schemaName: string, viewName: string): Promise<void> {
    await this.expandNode(schemaName);
    await this.expandNode('Views');
    await this.getTreeNode(viewName).click();
  }

  // Assertions
  async expectSchemaVisible(schemaName: string): Promise<void> {
    await expect(this.getSchemaNode(schemaName)).toBeVisible();
  }

  async expectTableVisible(tableName: string): Promise<void> {
    await expect(this.getTableNode(tableName)).toBeVisible();
  }

  async expectNodeExpanded(nodeName: string): Promise<void> {
    const node = this.getTreeNode(nodeName);
    await expect(node.locator('[class*="expanded"], .expanded')).toBeVisible();
  }

  async expectNodeCollapsed(nodeName: string): Promise<void> {
    const node = this.getTreeNode(nodeName);
    await expect(node.locator('[class*="expanded"], .expanded')).not.toBeVisible();
  }

  async expectSearchResults(minCount: number): Promise<void> {
    const items = this.schemaTree.locator('.tree-item, [class*="tree-item"]');
    await expect(items).toHaveCount(minCount, { timeout: 5000 });
  }

  async expectNoSearchResults(): Promise<void> {
    await expect(this.noResultsMessage).toBeVisible();
  }

  async expectContextMenuVisible(): Promise<void> {
    await expect(this.contextMenu).toBeVisible();
  }

  async expectContextMenuHidden(): Promise<void> {
    await expect(this.contextMenu).not.toBeVisible();
  }

  // Filter modal helpers
  get filterModal(): Locator {
    return this.page.locator('.filter-modal');
  }

  get filterModalTitle(): Locator {
    return this.filterModal.locator('.modal-header h3');
  }

  get filterTableName(): Locator {
    return this.filterModal.locator('.filter-table-name');
  }

  get filterConditions(): Locator {
    return this.filterModal.locator('.filter-section').first().locator('.filter-condition');
  }

  get orderByConditions(): Locator {
    return this.filterModal.locator('.filter-section').nth(1).locator('.filter-condition');
  }

  get filterLogicSelect(): Locator {
    return this.filterModal.locator('.logic-select');
  }

  get limitInput(): Locator {
    return this.filterModal.locator('.filter-row input[type="number"]');
  }

  get addConditionButton(): Locator {
    return this.filterModal.locator('.filter-section').first().locator('.add-btn');
  }

  get addOrderByButton(): Locator {
    return this.filterModal.locator('.filter-section').nth(1).locator('.add-btn');
  }

  get applyFilterButton(): Locator {
    return this.filterModal.locator('.modal-footer .btn-primary');
  }

  get cancelFilterButton(): Locator {
    return this.filterModal.locator('.modal-footer .btn-secondary');
  }

  async expectFilterModalVisible(): Promise<void> {
    await expect(this.filterModal).toBeVisible({ timeout: 5000 });
  }

  async expectFilterModalHidden(): Promise<void> {
    await expect(this.filterModal).not.toBeVisible();
  }

  async setFilterColumn(index: number, columnName: string): Promise<void> {
    const condition = this.filterConditions.nth(index);
    const columnSelect = condition.locator('.filter-col-select');
    await columnSelect.selectOption(columnName);
  }

  async setFilterOperator(index: number, operator: string): Promise<void> {
    const condition = this.filterConditions.nth(index);
    const operatorSelect = condition.locator('.filter-op-select');
    await operatorSelect.selectOption(operator);
  }

  async setFilterValue(index: number, value: string): Promise<void> {
    const condition = this.filterConditions.nth(index);
    const valueInput = condition.locator('.filter-value-input');
    await valueInput.fill(value);
  }

  async removeFilterCondition(index: number): Promise<void> {
    const condition = this.filterConditions.nth(index);
    const removeButton = condition.locator('.btn-icon');
    await removeButton.click();
  }

  async addFilterCondition(): Promise<void> {
    await this.addConditionButton.click();
  }

  async setFilterLogic(logic: 'AND' | 'OR'): Promise<void> {
    await this.filterLogicSelect.selectOption(logic);
  }

  async setOrderByColumn(index: number, columnName: string): Promise<void> {
    const condition = this.orderByConditions.nth(index);
    const columnSelect = condition.locator('select').first();
    await columnSelect.selectOption(columnName);
  }

  async setOrderByDirection(index: number, direction: 'ASC' | 'DESC'): Promise<void> {
    const condition = this.orderByConditions.nth(index);
    const directionSelect = condition.locator('.filter-dir-select');
    await directionSelect.selectOption(direction);
  }

  async addOrderByCondition(): Promise<void> {
    await this.addOrderByButton.click();
  }

  async removeOrderByCondition(index: number): Promise<void> {
    const condition = this.orderByConditions.nth(index);
    const removeButton = condition.locator('.btn-icon');
    await removeButton.click();
  }

  async setLimit(limit: number): Promise<void> {
    await this.limitInput.fill(limit.toString());
  }

  async applyFilter(): Promise<void> {
    await this.applyFilterButton.click();
    await this.expectFilterModalHidden();
  }

  async cancelFilter(): Promise<void> {
    await this.cancelFilterButton.click();
    await this.expectFilterModalHidden();
  }
}
