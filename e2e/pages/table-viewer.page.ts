import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Page object for the Table Viewer (data grid)
 */
export class TableViewerPage extends BasePage {
  // Main container
  get viewer(): Locator {
    return this.page.locator('.table-viewer, [class*="table-viewer"], .data-viewer');
  }

  // Header section
  get header(): Locator {
    return this.viewer.locator('.viewer-header, [class*="header"]').first();
  }

  get tableName(): Locator {
    return this.header.locator('.table-name, h2, h3');
  }

  get rowCountDisplay(): Locator {
    return this.header.locator('.row-count, [class*="count"]');
  }

  // Toolbar
  get toolbar(): Locator {
    return this.viewer.locator('.toolbar, [class*="toolbar"]');
  }

  get refreshButton(): Locator {
    return this.toolbar.getByRole('button', { name: /refresh/i });
  }

  get exportButton(): Locator {
    return this.toolbar.getByRole('button', { name: /export|csv/i });
  }

  get filterInput(): Locator {
    return this.toolbar.locator('input[type="search"], input[placeholder*="filter" i]');
  }

  get addRowButton(): Locator {
    return this.toolbar.getByRole('button', { name: /add|insert|new/i });
  }

  // Data table
  get dataTable(): Locator {
    return this.viewer.locator('table, .data-table, .ag-root');
  }

  // Alias for dataTable
  get table(): Locator {
    return this.dataTable;
  }

  get tableHeaders(): Locator {
    return this.dataTable.locator('thead th, .ag-header-cell');
  }

  get tableRows(): Locator {
    // Exclude empty-row and loading-row rows from the count
    return this.dataTable.locator('tbody tr:not(:has(.empty-row)):not(:has(.loading-row)), .ag-row');
  }

  get tableCells(): Locator {
    return this.dataTable.locator('tbody td, .ag-cell');
  }

  get selectedRow(): Locator {
    return this.dataTable.locator('tr.selected, .ag-row-selected');
  }

  // Pagination
  get pagination(): Locator {
    return this.viewer.locator('.pagination').first();
  }

  get firstPageButton(): Locator {
    return this.pagination.getByRole('button', { name: /first|<</ });
  }

  get prevPageButton(): Locator {
    return this.pagination.getByRole('button', { name: /prev|</ });
  }

  get nextPageButton(): Locator {
    return this.pagination.getByRole('button', { name: /next|>/ });
  }

  get lastPageButton(): Locator {
    return this.pagination.getByRole('button', { name: /last|>>/ });
  }

  get pageInput(): Locator {
    return this.pagination.locator('input[type="number"], .page-input');
  }

  get pageSizeSelect(): Locator {
    return this.pagination.locator('select, .page-size-select');
  }

  get pageInfo(): Locator {
    return this.pagination.locator('.page-info');
  }

  // Loading and states
  get loadingIndicator(): Locator {
    return this.viewer.locator('.loading, .spinner, [class*="loading"]');
  }

  get emptyState(): Locator {
    return this.viewer.locator('.empty, [class*="empty"], .no-data');
  }

  get errorState(): Locator {
    return this.viewer.locator('.error, [class*="error"]');
  }

  // FK Preview popup
  get fkPreviewPopup(): Locator {
    return this.page.locator('.fk-preview, [class*="fk-preview"], .preview-popup');
  }

  // Context menu - be specific to avoid matching backdrop
  get contextMenu(): Locator {
    return this.page.locator('.context-menu:not(.context-menu-backdrop)');
  }

  // Cell edit overlay
  get cellEditOverlay(): Locator {
    return this.viewer.locator('.cell-edit, [class*="cell-edit"], input.editing');
  }

  // Column-specific locators
  getColumnHeader(columnName: string): Locator {
    return this.tableHeaders.filter({ hasText: columnName });
  }

  getSortIndicator(columnName: string): Locator {
    return this.getColumnHeader(columnName).locator('.sort-icon, [class*="sort"]');
  }

  getCell(row: number, column: string): Locator {
    return this.getRow(row).locator(`td[data-column="${column}"], td`).filter({
      has: this.page.locator(`[data-column="${column}"]`),
    });
  }

  getCellByIndex(row: number, colIndex: number): Locator {
    return this.getRow(row).locator(`td:nth-child(${colIndex + 1})`);
  }

  getRow(index: number): Locator {
    return this.tableRows.nth(index);
  }

  getFkCell(row: number, columnName: string): Locator {
    return this.getRow(row).locator(`td[data-fk="true"], .fk-cell`).filter({
      has: this.page.locator(`[data-column="${columnName}"]`),
    });
  }

  getContextMenuItem(label: string): Locator {
    return this.contextMenu.getByRole('menuitem', { name: new RegExp(label, 'i') });
  }

  // Actions
  async waitForLoad(timeout = 30000): Promise<void> {
    await expect(this.loadingIndicator).not.toBeVisible({ timeout });
    await expect(this.dataTable).toBeVisible({ timeout });
  }

  async refresh(): Promise<void> {
    await this.refreshButton.click();
    await this.waitForLoad();
  }

  async exportToCsv(): Promise<void> {
    await this.exportButton.click();
  }

  async filter(text: string): Promise<void> {
    await this.fillInput(this.filterInput, text);
    await this.page.waitForTimeout(300); // Debounce
    await this.waitForLoad();
  }

  async clearFilter(): Promise<void> {
    await this.filterInput.clear();
    await this.page.waitForTimeout(300);
    await this.waitForLoad();
  }

  // Sorting
  async sortByColumn(columnName: string): Promise<void> {
    await this.getColumnHeader(columnName).click();
    await this.waitForLoad();
  }

  async sortByColumnDescending(columnName: string): Promise<void> {
    await this.sortByColumn(columnName);
    await this.sortByColumn(columnName); // Click again for descending
  }

  // Pagination
  async goToFirstPage(): Promise<void> {
    await this.firstPageButton.click();
    await this.waitForLoad();
  }

  async goToPreviousPage(): Promise<void> {
    await this.prevPageButton.click();
    await this.waitForLoad();
  }

  async goToNextPage(): Promise<void> {
    await this.nextPageButton.click();
    await this.waitForLoad();
  }

  async goToLastPage(): Promise<void> {
    await this.lastPageButton.click();
    await this.waitForLoad();
  }

  async goToPage(page: number): Promise<void> {
    await this.fillInput(this.pageInput, String(page));
    await this.pressEnter();
    await this.waitForLoad();
  }

  async setPageSize(size: number): Promise<void> {
    await this.pageSizeSelect.selectOption(String(size));
    await this.waitForLoad();
  }

  // Row interactions
  async clickRow(index: number): Promise<void> {
    await this.getRow(index).click();
  }

  async doubleClickRow(index: number): Promise<void> {
    await this.getRow(index).dblclick();
  }

  async rightClickRow(index: number): Promise<void> {
    await this.getRow(index).click({ button: 'right' });
    await expect(this.contextMenu).toBeVisible();
  }

  async selectRow(index: number): Promise<void> {
    await this.clickRow(index);
  }

  async selectMultipleRows(indices: number[]): Promise<void> {
    for (let i = 0; i < indices.length; i++) {
      const modifier = i === 0 ? undefined : 'Control';
      await this.getRow(indices[i]).click({
        modifiers: modifier ? [modifier] : undefined,
      });
    }
  }

  // Cell interactions
  async clickCell(row: number, colIndex: number): Promise<void> {
    await this.getCellByIndex(row, colIndex).click();
  }

  async doubleClickCell(row: number, colIndex: number): Promise<void> {
    await this.getCellByIndex(row, colIndex).dblclick();
  }

  async editCell(row: number, colIndex: number, value: string): Promise<void> {
    await this.doubleClickCell(row, colIndex);
    await expect(this.cellEditOverlay).toBeVisible();
    await this.page.keyboard.press('Control+a');
    await this.page.keyboard.type(value);
    await this.pressEnter();
  }

  // FK Preview
  async hoverFkCell(row: number, columnName: string): Promise<void> {
    const cell = this.getFkCell(row, columnName);
    await cell.hover();
    await expect(this.fkPreviewPopup).toBeVisible({ timeout: 3000 });
  }

  async clickFkValue(row: number, columnName: string): Promise<void> {
    const cell = this.getFkCell(row, columnName);
    await cell.click();
  }

  // Context menu actions
  async clickContextMenuItem(label: string): Promise<void> {
    await this.getContextMenuItem(label).click();
    await expect(this.contextMenu).not.toBeVisible();
  }

  async deleteRowViaContextMenu(index: number): Promise<void> {
    await this.rightClickRow(index);
    await this.clickContextMenuItem('Delete');
  }

  async editRowViaContextMenu(index: number): Promise<void> {
    await this.rightClickRow(index);
    await this.clickContextMenuItem('Edit');
  }

  // Data retrieval
  async getRowCount(): Promise<number> {
    return await this.tableRows.count();
  }

  async getColumnNames(): Promise<string[]> {
    // Get just the column names, not the data types (which are in .col-type)
    const colNames = this.table.locator('thead th .col-name');
    const names = await colNames.allTextContents();
    return names.map((h) => h.trim());
  }

  async getCellValue(row: number, colIndex: number): Promise<string> {
    return (await this.getCellByIndex(row, colIndex).textContent()) || '';
  }

  async getRowValues(index: number): Promise<string[]> {
    const cells = this.getRow(index).locator('td');
    return await cells.allTextContents();
  }

  async getCurrentPage(): Promise<number> {
    // Page info is displayed as "Page X of Y"
    const text = await this.pageInfo.textContent();
    const match = text?.match(/Page\s+(\d+)/i);
    return match ? parseInt(match[1], 10) : 1;
  }

  async getTotalPages(): Promise<number> {
    const text = await this.pageInfo.textContent();
    const match = text?.match(/of\s+(\d+)/i);
    return match ? parseInt(match[1], 10) : 1;
  }

  async getTotalRowCount(): Promise<number> {
    const text = await this.rowCountDisplay.textContent();
    const match = text?.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  // Assertions
  async expectTableVisible(): Promise<void> {
    await expect(this.dataTable).toBeVisible();
  }

  async expectRowCount(count: number): Promise<void> {
    await expect(this.tableRows).toHaveCount(count);
  }

  async expectMinRowCount(minCount: number): Promise<void> {
    const count = await this.getRowCount();
    expect(count).toBeGreaterThanOrEqual(minCount);
  }

  async expectColumnExists(columnName: string): Promise<void> {
    await expect(this.getColumnHeader(columnName)).toBeVisible();
  }

  async expectCellValue(row: number, colIndex: number, value: string): Promise<void> {
    await expect(this.getCellByIndex(row, colIndex)).toContainText(value);
  }

  async expectRowSelected(index: number): Promise<void> {
    await expect(this.getRow(index)).toHaveClass(/selected/);
  }

  async expectEmpty(): Promise<void> {
    await expect(this.emptyState).toBeVisible();
  }

  async expectNotEmpty(): Promise<void> {
    await expect(this.emptyState).not.toBeVisible();
    await expect(this.tableRows.first()).toBeVisible();
  }

  async expectError(): Promise<void> {
    await expect(this.errorState).toBeVisible();
  }

  async expectNoError(): Promise<void> {
    await expect(this.errorState).not.toBeVisible();
  }

  async expectSortedAscending(columnName: string): Promise<void> {
    await expect(this.getSortIndicator(columnName)).toHaveClass(/asc|ascending/i);
  }

  async expectSortedDescending(columnName: string): Promise<void> {
    await expect(this.getSortIndicator(columnName)).toHaveClass(/desc|descending/i);
  }

  async expectPage(pageNumber: number): Promise<void> {
    await expect(this.pageInput).toHaveValue(String(pageNumber));
  }

  async expectFkPreviewVisible(): Promise<void> {
    await expect(this.fkPreviewPopup).toBeVisible();
  }

  async expectFkPreviewHidden(): Promise<void> {
    await expect(this.fkPreviewPopup).not.toBeVisible();
  }

  async expectContextMenuVisible(): Promise<void> {
    await expect(this.contextMenu).toBeVisible();
  }

  async expectContextMenuHidden(): Promise<void> {
    await expect(this.contextMenu).not.toBeVisible();
  }
}
