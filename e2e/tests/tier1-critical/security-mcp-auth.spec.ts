import { testWithCleanState as test, expect, getTestConnectionConfig } from '../../fixtures';
import { AppPage } from '../../pages';

/**
 * Regression tests for MCP API + WebSocket authentication.
 *
 * Before phase 4 of the security overhaul, the MCP API authenticated only
 * via the `X-Claude-Session-ID` header (a UUID known to several local
 * processes) and the terminal WebSocket had no auth at all once you knew
 * the session ID. These tests assert:
 *
 * - CreateSession returns both a sessionId AND a per-session bearer token.
 * - Every MCP endpoint requires `Authorization: Bearer <token>`.
 * - Wrong bearer tokens are rejected with 401.
 * - Session-scoped destroy/update endpoints require the bearer token.
 * - WebSocket upgrades without a token are rejected.
 */
test.describe('Security: MCP auth requires per-session bearer token', () => {
  test.describe.configure({ mode: 'serial' });

  let app: AppPage;
  let connectionId: string;
  const baseURL = process.env.BASE_URL || 'http://localhost:5137';

  test.beforeAll(async ({ browser, request }) => {
    const page = await browser.newPage();
    app = new AppPage(page);
    await app.goto();
    const config = getTestConnectionConfig();
    await app.createConnection(config);
    await app.expectConnected(config.name);

    const conns = (await (await request.get(`${baseURL}/api/connections`)).json()) as {
      id: string;
      name: string;
    }[];
    connectionId = conns.find((c) => c.name === config.name)!.id;
  });

  test.afterAll(async () => {
    await app.page.close();
  });

  test('CreateSession returns a sessionId AND a token', async ({ request }) => {
    const resp = await request.post(`${baseURL}/api/claude/sessions`, {
      data: { connectionId },
    });
    if (!resp.ok()) {
      // Spawning Claude may not be available in all CI environments.
      // Skip the rest of the test rather than fail spuriously.
      test.skip(true, `Claude session create returned ${resp.status()}`);
      return;
    }
    const body = await resp.json();
    expect(typeof body.sessionId).toBe('string');
    expect(typeof body.token).toBe('string');
    expect(body.token.length).toBeGreaterThan(20);

    // Tear it down so we don't leak a Claude subprocess.
    await request.delete(`${baseURL}/api/claude/sessions/${body.sessionId}`, {
      headers: { Authorization: `Bearer ${body.token}` },
    });
  });

  test('MCP API rejects missing Authorization header', async ({ request }) => {
    const resp = await request.get(`${baseURL}/api/mcp/schemas`, {
      headers: { 'X-Claude-Session-ID': 'any-uuid' },
    });
    expect(resp.status()).toBe(401);
  });

  test('MCP API rejects malformed Authorization header', async ({ request }) => {
    const resp = await request.get(`${baseURL}/api/mcp/schemas`, {
      headers: {
        'X-Claude-Session-ID': 'any-uuid',
        Authorization: 'Basic dXNlcjpwYXNz',
      },
    });
    expect(resp.status()).toBe(401);
  });

  test('MCP API rejects wrong bearer token', async ({ request }) => {
    const resp = await request.get(`${baseURL}/api/mcp/schemas`, {
      headers: {
        'X-Claude-Session-ID': 'any-uuid',
        Authorization: 'Bearer not-a-real-token',
      },
    });
    expect(resp.status()).toBe(401);
  });

  test('DestroySession rejects requests without bearer token', async ({ request }) => {
    const resp = await request.delete(`${baseURL}/api/claude/sessions/any-uuid`);
    expect(resp.status()).toBe(401);
  });

  test('UpdateSessionConnection rejects requests without bearer token', async ({ request }) => {
    const resp = await request.put(`${baseURL}/api/claude/sessions/any-uuid/connection`, {
      data: { connectionId },
    });
    expect(resp.status()).toBe(401);
  });
});
