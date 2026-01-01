import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Query Editor Tests
 * Uses a SINGLE shared connection to avoid PostgreSQL connection exhaustion
 */
test.describe('Query Editor', () => {
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

  // Open a new query tab before each test
  test.beforeEach(async () => {
    await app.sidebar.openNewQuery();
    await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
  });

  test.describe('Basic Query Execution', () => {
    test('should execute simple SELECT query', async () => {
      await app.queryEditor.setQuery('SELECT 1 as result');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('result');
    });

    test('should execute query with Ctrl+Enter', async () => {
      await app.queryEditor.setQuery('SELECT 2 as value');
      await app.queryEditor.executeWithKeyboard();

      await app.queryEditor.expectResults();
    });

    test('should query test_schema tables', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users LIMIT 5');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('id');
      await app.queryEditor.expectColumnExists('name');
      await app.queryEditor.expectColumnExists('email');
    });

    test('should display multiple rows in results', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      // We have 5 test users
      const rows = await app.queryEditor.resultsTableRows.count();
      expect(rows).toBe(5);
    });

    test('should display row count', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectRowCount(5);
    });
  });

  test.describe('Query Results', () => {
    test('should get correct column names', async () => {
      await app.queryEditor.setQuery(
        'SELECT id, name, email FROM test_schema.users LIMIT 1'
      );
      await app.queryEditor.executeQuery();

      const columns = await app.queryEditor.getResultsColumnNames();
      expect(columns).toContain('id');
      expect(columns).toContain('name');
      expect(columns).toContain('email');
    });

    test('should display correct cell values', async () => {
      await app.queryEditor.setQuery(
        "SELECT name, email FROM test_schema.users WHERE name = 'Alice Johnson'"
      );
      await app.queryEditor.executeQuery();

      const name = await app.queryEditor.getResultsCellValue(1, 'name');
      expect(name).toContain('Alice Johnson');

      const email = await app.queryEditor.getResultsCellValue(1, 'email');
      expect(email).toContain('alice@example.com');
    });

    test('should show execution time', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test_schema.users');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectExecutionTime();
    });
  });

  test.describe('Complex Queries', () => {
    test('should execute JOIN query', async () => {
      await app.queryEditor.setQuery(`
        SELECT u.name, o.total, o.status
        FROM test_schema.users u
        JOIN test_schema.orders o ON u.id = o.user_id
        LIMIT 5
      `);
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('name');
      await app.queryEditor.expectColumnExists('total');
      await app.queryEditor.expectColumnExists('status');
    });

    test('should execute aggregate query', async () => {
      await app.queryEditor.setQuery(`
        SELECT status, COUNT(*) as order_count
        FROM test_schema.orders
        GROUP BY status
      `);
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('status');
      await app.queryEditor.expectColumnExists('order_count');
    });

    test('should query view', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test_schema.user_order_summary');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('order_count');
      await app.queryEditor.expectColumnExists('total_spent');
    });
  });

  test.describe('Error Handling', () => {
    test('should display SQL syntax error', async () => {
      await app.queryEditor.setQuery('SELEC * FORM users');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectError();
    });

    test('should display error for non-existent table', async () => {
      await app.queryEditor.setQuery('SELECT * FROM nonexistent_table');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectError('does not exist');
    });

    test('should display error for non-existent column', async () => {
      await app.queryEditor.setQuery(
        'SELECT nonexistent_column FROM test_schema.users'
      );
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectError();
    });

    test('should recover from error and execute valid query', async () => {
      // First execute invalid query
      await app.queryEditor.setQuery('INVALID SQL');
      await app.queryEditor.executeQuery();
      await app.queryEditor.expectError();

      // Then execute valid query
      await app.queryEditor.setQuery('SELECT 1 as success');
      await app.queryEditor.executeQuery();
      await app.queryEditor.expectResults();
      await app.queryEditor.expectNoError();
    });
  });

  test.describe('Editor Features', () => {
    test('should clear query editor', async () => {
      await app.queryEditor.setQuery('SELECT * FROM test');
      await app.queryEditor.clearQuery();

      const text = await app.queryEditor.getQueryText();
      expect(text.trim()).toBe('');
    });

    test('should append to existing query', async () => {
      await app.queryEditor.setQuery('SELECT *');
      await app.queryEditor.appendQuery(' FROM test_schema.users');

      const text = await app.queryEditor.getQueryText();
      expect(text).toContain('SELECT *');
      expect(text).toContain('FROM test_schema.users');
    });

    test('should focus editor', async () => {
      await app.queryEditor.focus();
      await expect(app.queryEditor.codeContent).toBeFocused();
    });
  });

  test.describe('Empty Results', () => {
    test('should show no results message for empty query result', async () => {
      await app.queryEditor.setQuery(
        "SELECT * FROM test_schema.users WHERE name = 'Nonexistent Person'"
      );
      await app.queryEditor.executeQuery();

      // Should show results table but with 0 rows
      const rowCount = await app.queryEditor.resultsTableRows.count();
      expect(rowCount).toBe(0);
    });
  });

  test.describe('Query Tab Isolation', () => {
    test('should open new query tab with correct SQL from context menu', async () => {
      // First, add some custom text to the current query editor
      await app.queryEditor.setQuery('-- This is my custom query');

      // Now right-click on a table and open in query
      await app.sidebar.expandNode('test_schema');
      await app.sidebar.expandNode('Tables');
      await app.sidebar.rightClickTable('users');

      // Click "Open in Query" from context menu
      await app.sidebar.clickContextMenuItem('Open in Query');

      // Wait for the new query tab to be active
      await app.queryEditor.codeEditor.waitFor({ state: 'visible', timeout: 5000 });
      await app.page.waitForTimeout(300);

      // The new query should have SELECT statement, NOT the previous custom text
      const queryText = await app.queryEditor.getQueryText();
      expect(queryText).toContain('SELECT *');
      expect(queryText).toContain('test_schema');
      expect(queryText).toContain('users');
      expect(queryText).not.toContain('This is my custom query');
    });
  });

  test.describe('Special Values', () => {
    test('should handle NULL values in results', async () => {
      await app.queryEditor.setQuery('SELECT NULL as null_value');
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
    });

    test('should handle timestamp values', async () => {
      await app.queryEditor.setQuery(
        'SELECT created_at FROM test_schema.users LIMIT 1'
      );
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      await app.queryEditor.expectColumnExists('created_at');
    });

    test('should handle decimal values', async () => {
      await app.queryEditor.setQuery(
        'SELECT price FROM test_schema.products LIMIT 1'
      );
      await app.queryEditor.executeQuery();

      await app.queryEditor.expectResults();
      const price = await app.queryEditor.getResultsCellValue(1, 'price');
      expect(price).toMatch(/\d+\.\d+/);
    });
  });
});
