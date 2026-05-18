import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Regression tests for SQL injection in DDL handlers.
 *
 * The backend previously interpolated user-supplied `col.Type`, `col.Default`,
 * `onDelete`, `onUpdate`, and CHECK `expression` straight into DDL. A crafted
 * JSON body could break out of the SQL fragment and execute arbitrary
 * statements. These tests assert each path now rejects hostile input with a
 * 400 and that the schema is unchanged afterward.
 */
test.describe('Security: DDL injection rejection', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let connectionId: string;
  const baseURL = process.env.BASE_URL || 'http://localhost:5137';
  const probeTable = `e2e_sec_probe_${Date.now()}`;

  test.beforeAll(async ({ browser, request }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);

    // Fetch the connection ID via the API for direct DDL calls below.
    const resp = await request.get(`${baseURL}/api/connections`);
    expect(resp.ok()).toBeTruthy();
    const conns = (await resp.json()) as { id: string; name: string }[];
    const match = conns.find((c) => c.name === config.name);
    if (!match) throw new Error('test connection not found');
    connectionId = match.id;

    // Create a real probe table that the injection attempts would target
    // if they succeeded. The tests below assert it still exists at the end.
    const create = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema`,
      {
        data: {
          name: probeTable,
          columns: [
            { name: 'id', type: 'int', nullable: false, primaryKey: true },
          ],
        },
      }
    );
    expect(create.ok()).toBeTruthy();
  });

  test.afterAll(async ({ request }) => {
    if (connectionId) {
      await request.delete(
        `${baseURL}/api/data/${connectionId}/tables/test_schema/${probeTable}`,
        { data: { cascade: true } }
      );
    }
    await app.page.close();
  });

  test('CreateTable rejects injection in column type', async ({ request }) => {
    const resp = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema`,
      {
        data: {
          name: `evil_${Date.now()}`,
          columns: [
            {
              name: 'id',
              type: `int); DROP TABLE test_schema.${probeTable}; --`,
              nullable: false,
              primaryKey: true,
            },
          ],
        },
      }
    );
    expect(resp.status()).toBe(400);
    const body = await resp.json();
    expect(JSON.stringify(body).toLowerCase()).toContain('type');
  });

  test('CreateTable rejects injection in column default', async ({ request }) => {
    const resp = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema`,
      {
        data: {
          name: `evil2_${Date.now()}`,
          columns: [
            {
              name: 'id',
              type: 'int',
              nullable: false,
              primaryKey: true,
              default: `1); DROP TABLE test_schema.${probeTable}; SELECT (1`,
            },
          ],
        },
      }
    );
    expect(resp.status()).toBe(400);
  });

  test('AddConstraint rejects injection in onDelete', async ({ request }) => {
    const resp = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema/${probeTable}/constraints`,
      {
        data: {
          type: 'fk',
          columns: ['id'],
          refTable: 'users',
          refColumns: ['id'],
          onDelete: `CASCADE; DROP TABLE test_schema.${probeTable}; --`,
        },
      }
    );
    expect(resp.status()).toBe(400);
  });

  test('AddConstraint rejects injection in onUpdate', async ({ request }) => {
    const resp = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema/${probeTable}/constraints`,
      {
        data: {
          type: 'fk',
          columns: ['id'],
          refTable: 'users',
          refColumns: ['id'],
          onDelete: 'CASCADE',
          onUpdate: 'NOPE',
        },
      }
    );
    expect(resp.status()).toBe(400);
  });

  test('AddConstraint rejects injection in CHECK expression', async ({ request }) => {
    const resp = await request.post(
      `${baseURL}/api/data/${connectionId}/tables/test_schema/${probeTable}/constraints`,
      {
        data: {
          type: 'check',
          expression: `id > 0); DROP TABLE test_schema.${probeTable}; --`,
        },
      }
    );
    expect(resp.status()).toBe(400);
  });

  test('Probe table still exists after every injection attempt', async ({ request }) => {
    // If any of the rejections leaked through, the probe table would be gone.
    const resp = await request.get(`${baseURL}/api/data/${connectionId}/tables/test_schema/${probeTable}/count`);
    expect(resp.ok()).toBeTruthy();
  });
});
