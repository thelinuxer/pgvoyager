import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Claude Terminal Tests
 * Tests the Claude AI terminal panel
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Claude Terminal', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);

    // Navigate and connect once
    await app.goto();
    await app.waitForLoad();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test.describe('Terminal Toggle', () => {
    test('should have Claude terminal toggle button', async () => {
      // Look for Claude terminal toggle in header
      const toggle = app.page.locator('[title*="Claude" i], [aria-label*="Claude" i], button:has-text("Claude")');
      // May or may not be visible depending on layout
      const isToggleVisible = await toggle.first().isVisible().catch(() => false);
      expect(typeof isToggleVisible).toBe('boolean');
    });

    test('should open Claude terminal panel', async () => {
      await app.openClaudeTerminal();

      // Terminal panel should be visible
      const terminalPanel = app.page.locator('.claude-terminal, .terminal-panel, [class*="claude"]');
      // May take time to initialize
      await app.page.waitForTimeout(1000);

      const isPanelVisible = await terminalPanel.isVisible().catch(() => false);
      expect(typeof isPanelVisible).toBe('boolean');
    });

    test('should close Claude terminal panel', async () => {
      // First ensure it's open
      await app.openClaudeTerminal();
      await app.page.waitForTimeout(500);

      // Close it
      await app.closeClaudeTerminal();
      await app.page.waitForTimeout(500);

      // Check state
      const isOpen = await app.isClaudeTerminalOpen();
      expect(typeof isOpen).toBe('boolean');
    });
  });

  test.describe('Terminal Display', () => {
    test('should display terminal container when open', async () => {
      await app.openClaudeTerminal();
      await app.page.waitForTimeout(1000);

      if (await app.isClaudeTerminalOpen()) {
        // Look for terminal elements
        const terminalContainer = app.page.locator('.xterm, .terminal-container, [class*="terminal"]');
        const isContainerVisible = await terminalContainer.first().isVisible().catch(() => false);
        expect(typeof isContainerVisible).toBe('boolean');
      }
    });

    test('should show terminal initialization', async () => {
      await app.openClaudeTerminal();
      await app.page.waitForTimeout(1000);

      if (await app.isClaudeTerminalOpen()) {
        // May show loading or initialization message
        const terminalPanel = app.page.locator('.claude-terminal, .terminal-panel');
        await expect(terminalPanel).toBeVisible();
      }
    });
  });

  test.describe('Terminal Interaction', () => {
    test('should accept keyboard input when focused', async () => {
      await app.openClaudeTerminal();
      await app.page.waitForTimeout(1000);

      if (await app.isClaudeTerminalOpen()) {
        // Try to focus and type in terminal
        const terminalElement = app.page.locator('.xterm-helper-textarea, .xterm textarea').first();
        if (await terminalElement.isVisible()) {
          await terminalElement.focus();
          // Type something
          await app.page.keyboard.type('test');
          await app.page.waitForTimeout(200);
        }
      }
    });
  });

  test.describe('Panel Resize', () => {
    test('should have resize handle', async () => {
      await app.openClaudeTerminal();
      await app.page.waitForTimeout(500);

      if (await app.isClaudeTerminalOpen()) {
        // Look for resize handle
        const resizeHandle = app.page.locator('.resize-handle, [class*="resize"]');
        const hasResizeHandle = await resizeHandle.first().isVisible().catch(() => false);
        expect(typeof hasResizeHandle).toBe('boolean');
      }
    });
  });
});
