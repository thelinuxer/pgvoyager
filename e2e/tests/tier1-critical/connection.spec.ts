import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Connection Management Tests
 * Tests are organized to minimize PostgreSQL connection usage
 */
test.describe('Connection Management', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.describe('Create Connection - Modal Tests', () => {
    // These tests don't actually connect, so they can run independently
    test.beforeEach(async ({ page }) => {
      app = new AppPage(page);
      await app.goto();
    });

    test('should display welcome screen on first load', async () => {
      await app.expectWelcomeScreen();
      await expect(app.welcomeNewConnectionButton).toBeVisible();
    });

    test('should open connection modal from welcome screen', async () => {
      await app.welcomeNewConnectionButton.click();
      await app.connectionModal.waitForOpen();
      await expect(app.connectionModal.modal).toBeVisible();
    });

    test('should open connection modal from header button', async () => {
      await app.openNewConnectionModal();
      await expect(app.connectionModal.modal).toBeVisible();
    });

    test('should fill connection form with all fields', async () => {
      const config = getTestConnectionConfig();

      await app.openNewConnectionModal();
      await app.connectionModal.fillConnectionForm(config);

      await app.connectionModal.expectFormValues({
        name: config.name,
        host: config.host,
        port: config.port,
        database: config.database,
        username: config.username,
      });
    });

    test('should cancel connection modal without saving', async () => {
      await app.openNewConnectionModal();
      await app.connectionModal.fillConnectionForm({
        name: 'Cancelled Connection',
        host: 'localhost',
        port: 5432,
        database: 'test',
        username: 'test',
        password: 'test',
      });

      await app.connectionModal.cancel();
      await app.connectionModal.waitForClose();

      // Should still be on welcome screen
      await app.expectWelcomeScreen();
    });
  });

  test.describe('Create Connection - Actual Connections', () => {
    // These tests use actual database connections - share a page
    let testPage: AppPage;

    test.beforeAll(async ({ browser }) => {
      const page = await browser.newPage();
      testPage = new AppPage(page);
      await testPage.goto();
    });

    test.afterAll(async () => {
      await testPage.page.close();
    });

    test('should test connection successfully', async () => {
      const config = getTestConnectionConfig();

      await testPage.openNewConnectionModal();
      await testPage.connectionModal.fillConnectionForm(config);

      const success = await testPage.connectionModal.testConnection();
      expect(success).toBe(true);
      await testPage.connectionModal.expectTestSuccess();
      await testPage.connectionModal.cancel(); // Close without connecting
    });

    test('should show error for invalid connection', async () => {
      await testPage.openNewConnectionModal();
      await testPage.connectionModal.fillConnectionForm({
        name: 'Bad Connection',
        host: 'nonexistent-host.invalid',
        port: 5432,
        database: 'nonexistent',
        username: 'nobody',
        password: 'wrongpassword',
      });

      const success = await testPage.connectionModal.testConnection();
      expect(success).toBe(false);
      await testPage.connectionModal.expectTestFailure();
      await testPage.connectionModal.cancel(); // Close the modal
    });

    test('should save connection and connect to database', async () => {
      const config = getTestConnectionConfig();

      await testPage.openNewConnectionModal();
      await testPage.connectionModal.fillConnectionForm(config);
      await testPage.connectionModal.saveAndConnect();

      // Modal should close and we should be connected
      await testPage.connectionModal.waitForClose();
      await testPage.expectConnected(config.name);

      // Schema tree should be visible
      await testPage.sidebar.waitForSchemaLoad();
    });
  });
});

test.describe('Connection Lifecycle', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();

    // Create and connect once
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);
    await app.sidebar.waitForSchemaLoad();
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('should display connection name in dropdown', async () => {
    const config = getTestConnectionConfig();
    await expect(app.connectionDropdown).toContainText(config.name);
  });

  test('should load schema after connecting', async () => {
    // Should see test_schema
    await app.sidebar.expectSchemaVisible('test_schema');
  });

  test('should disconnect from database', async () => {
    await app.disconnectFromDatabase();
    await app.expectDisconnected();
  });

  test('should show welcome screen after disconnect', async () => {
    await app.expectWelcomeScreen();
  });
});

test.describe('Connection Persistence', () => {
  test('should persist connection across reload', async ({ page }) => {
    const app = new AppPage(page);
    const config = getTestConnectionConfig();

    // Create connection
    await app.goto();
    await app.createConnection(config);
    await app.expectConnected(config.name);

    // Reload page
    await app.reload();

    // Connection info should be available (may need to reconnect)
    // The saved connection should appear in the connection dropdown
    await expect(app.connectionDropdown).toBeVisible();
  });
});
