import { Page, APIRequestContext, expect } from '@playwright/test';

/**
 * API helper utilities for E2E tests
 */

/**
 * Get the base URL for API calls
 */
export function getApiBaseUrl(): string {
  return process.env.BASE_URL || 'http://localhost:5137';
}

/**
 * Make an API request using Playwright's request context
 */
export async function apiRequest(
  request: APIRequestContext,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  endpoint: string,
  data?: unknown
): Promise<unknown> {
  const url = `${getApiBaseUrl()}${endpoint}`;

  let response;
  switch (method) {
    case 'GET':
      response = await request.get(url);
      break;
    case 'POST':
      response = await request.post(url, { data });
      break;
    case 'PUT':
      response = await request.put(url, { data });
      break;
    case 'DELETE':
      response = await request.delete(url);
      break;
  }

  if (!response.ok()) {
    throw new Error(`API request failed: ${response.status()} ${response.statusText()}`);
  }

  return response.json();
}

/**
 * Execute a SQL query through the API
 */
export async function executeQuery(
  request: APIRequestContext,
  connectionId: string,
  sql: string
): Promise<unknown> {
  return apiRequest(request, 'POST', '/api/query', {
    connectionId,
    sql,
  });
}

/**
 * Get list of connections through the API
 */
export async function getConnections(request: APIRequestContext): Promise<unknown[]> {
  const result = (await apiRequest(request, 'GET', '/api/connections')) as { connections: unknown[] };
  return result.connections || [];
}

/**
 * Create a new connection through the API
 */
export async function createConnection(
  request: APIRequestContext,
  config: {
    name: string;
    host: string;
    port: number;
    database: string;
    username: string;
    password: string;
    sslMode?: string;
  }
): Promise<unknown> {
  return apiRequest(request, 'POST', '/api/connections', config);
}

/**
 * Delete a connection through the API
 */
export async function deleteConnection(
  request: APIRequestContext,
  connectionId: string
): Promise<void> {
  await apiRequest(request, 'DELETE', `/api/connections/${connectionId}`);
}

/**
 * Get schema information through the API
 */
export async function getSchema(
  request: APIRequestContext,
  connectionId: string
): Promise<unknown> {
  return apiRequest(request, 'GET', `/api/connections/${connectionId}/schema`);
}

/**
 * Get table data through the API
 */
export async function getTableData(
  request: APIRequestContext,
  connectionId: string,
  schema: string,
  table: string,
  options: { limit?: number; offset?: number } = {}
): Promise<unknown> {
  const { limit = 50, offset = 0 } = options;
  return apiRequest(
    request,
    'GET',
    `/api/connections/${connectionId}/tables/${schema}.${table}/data?limit=${limit}&offset=${offset}`
  );
}

/**
 * Wait for API to be ready
 */
export async function waitForApi(request: APIRequestContext, timeout = 30000): Promise<void> {
  const startTime = Date.now();
  const baseUrl = getApiBaseUrl();

  while (Date.now() - startTime < timeout) {
    try {
      const response = await request.get(`${baseUrl}/api/health`);
      if (response.ok()) {
        return;
      }
    } catch {
      // API not ready yet
    }

    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  throw new Error('API did not become ready within timeout');
}

/**
 * Intercept and mock API responses
 */
export async function mockApiResponse(
  page: Page,
  urlPattern: string | RegExp,
  response: {
    status?: number;
    body?: unknown;
    headers?: Record<string, string>;
  }
): Promise<void> {
  await page.route(urlPattern, (route) => {
    route.fulfill({
      status: response.status || 200,
      contentType: 'application/json',
      body: JSON.stringify(response.body || {}),
      headers: response.headers,
    });
  });
}

/**
 * Capture API requests for verification
 */
export function captureApiRequests(page: Page, urlPattern: string | RegExp): {
  requests: { url: string; method: string; data?: unknown }[];
  clear: () => void;
} {
  const requests: { url: string; method: string; data?: unknown }[] = [];

  page.on('request', (request) => {
    const url = request.url();
    const matches =
      typeof urlPattern === 'string' ? url.includes(urlPattern) : urlPattern.test(url);

    if (matches) {
      requests.push({
        url,
        method: request.method(),
        data: request.postDataJSON(),
      });
    }
  });

  return {
    requests,
    clear: () => {
      requests.length = 0;
    },
  };
}
