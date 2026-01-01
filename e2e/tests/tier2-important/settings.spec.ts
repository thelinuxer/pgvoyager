import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage, SettingsModal } from '../../pages';

/**
 * Settings Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Settings', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let settingsModal: SettingsModal;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    settingsModal = new SettingsModal(page);

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

  test.describe('Opening Settings', () => {
    test('should open settings modal from header button', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();
    });

    test('should close settings with close button', async () => {
      // Ensure modal is open
      if (!(await settingsModal.modalContent.isVisible())) {
        await app.openSettings();
      }

      await settingsModal.close();
      await settingsModal.expectClosed();
    });

    test('should close settings by pressing Escape', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      await app.page.keyboard.press('Escape');
      await settingsModal.expectClosed();
    });

    test('should close settings by clicking backdrop', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // Click on backdrop
      await settingsModal.modal.click({ position: { x: 10, y: 10 } });
      await settingsModal.expectClosed();
    });
  });

  test.describe('Theme Settings', () => {
    test('should display theme section', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      await expect(settingsModal.themeSection).toBeVisible();
      await settingsModal.close();
    });

    test('should display theme options', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // Should have at least one theme card
      const themeCount = await settingsModal.themeCards.count();
      expect(themeCount).toBeGreaterThan(0);

      await settingsModal.close();
    });

    test('should have a selected theme', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // One theme should be selected
      const selectedTheme = settingsModal.themeCards.filter({ has: app.page.locator('.selected, .selected-indicator') });
      await expect(selectedTheme.first()).toBeVisible();

      await settingsModal.close();
    });

    test('should change theme when clicking theme card', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // Get count of themes
      const themeCount = await settingsModal.themeCards.count();
      if (themeCount > 1) {
        // Find a non-selected theme by checking each card
        let targetIndex = -1;
        for (let i = 0; i < themeCount; i++) {
          const card = settingsModal.themeCards.nth(i);
          const hasIndicator = await card.locator('.selected-indicator').count() > 0;
          if (!hasIndicator) {
            targetIndex = i;
            break;
          }
        }

        if (targetIndex >= 0) {
          // Get the theme name before clicking
          const targetCard = settingsModal.themeCards.nth(targetIndex);
          const themeName = await targetCard.locator('.theme-name').textContent();

          await targetCard.click();
          await app.page.waitForTimeout(500);

          // Verify the theme changed - use a fresh locator to check for selected state
          const updatedCard = settingsModal.themeCards.nth(targetIndex);
          const hasSelectedIndicator = await updatedCard.locator('.selected-indicator').count() > 0;
          const hasSelectedClass = await updatedCard.evaluate(el => el.classList.contains('selected'));

          expect(hasSelectedIndicator || hasSelectedClass).toBe(true);
        }
      }

      await settingsModal.close();
    });
  });

  test.describe('Icon Style Settings', () => {
    test('should display icon style section', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      await expect(settingsModal.iconStyleSection).toBeVisible();
      await settingsModal.close();
    });

    test('should display icon style options', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // Should have at least one icon library card
      const iconLibraryCount = await settingsModal.iconLibraryCards.count();
      expect(iconLibraryCount).toBeGreaterThan(0);

      await settingsModal.close();
    });

    test('should have a selected icon style', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // One icon library should be selected
      const selectedLibrary = settingsModal.iconLibraryCards.filter({
        has: app.page.locator('.selected, .selected-indicator'),
      });
      await expect(selectedLibrary.first()).toBeVisible();

      await settingsModal.close();
    });

    test('should change icon style when clicking card', async () => {
      await app.openSettings();
      await settingsModal.expectVisible();

      // Get count of icon libraries
      const libraryCount = await settingsModal.iconLibraryCards.count();
      if (libraryCount > 1) {
        // Find a non-selected library by checking each card
        let targetIndex = -1;
        for (let i = 0; i < libraryCount; i++) {
          const card = settingsModal.iconLibraryCards.nth(i);
          const hasIndicator = await card.locator('.selected-indicator').count() > 0;
          if (!hasIndicator) {
            targetIndex = i;
            break;
          }
        }

        if (targetIndex >= 0) {
          const targetCard = settingsModal.iconLibraryCards.nth(targetIndex);
          await targetCard.click();
          await app.page.waitForTimeout(300);

          // Verify the library changed - use a fresh locator to check for selected state
          const updatedCard = settingsModal.iconLibraryCards.nth(targetIndex);
          const hasSelectedIndicator = await updatedCard.locator('.selected-indicator').count() > 0;
          const hasSelectedClass = await updatedCard.evaluate(el => el.classList.contains('selected'));

          expect(hasSelectedIndicator || hasSelectedClass).toBe(true);
        }
      }

      await settingsModal.close();
    });
  });

  test.describe('Settings Persistence', () => {
    test('should persist theme after reload', async () => {
      // Open settings and note current theme
      await app.openSettings();
      await settingsModal.expectVisible();

      // Get currently selected theme name
      const selectedThemeCard = settingsModal.themeCards.filter({
        has: app.page.locator('.selected-indicator'),
      }).first();
      const themeName = await selectedThemeCard.locator('.theme-name').textContent();

      await settingsModal.close();

      // Reload page
      await app.reload();
      // Wait for app to load (but don't require schema - connection might not auto-reconnect)
      await app.waitForLoad();
      await app.page.waitForTimeout(1000);

      // Open settings again
      await app.openSettings();
      await settingsModal.expectVisible();

      // Same theme should be selected
      if (themeName) {
        await settingsModal.expectThemeSelected(themeName.trim());
      }

      await settingsModal.close();
    });
  });
});
