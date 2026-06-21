/** @jest-environment jsdom */
import APIClient from './api_client';

// Helpers ---------------------------------------------------------------

function base64url(obj: Record<string, unknown>): string {
  const json = JSON.stringify(obj);
  // btoa works on latin1; our payloads here are ASCII so this is fine.
  return window
    .btoa(json)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}

function makeToken(payload: Record<string, unknown>): string {
  const header = base64url({ alg: 'HS512', typ: 'JWT' });
  return `${header}.${base64url(payload)}.signature-not-verified-on-client`;
}

function mockOkFetch() {
  // jsdom doesn't expose a global `Response`, and APIClient.fetch only reads
  // `status`, `ok` and `json()`, so a lightweight stub is enough here.
  const fn = jest.fn(async () => ({
    status: 200,
    ok: true,
    json: async () => ({ ok: true }),
  }));
  // @ts-expect-error overriding the test global with a minimal stub
  window.fetch = fn;
  return fn;
}

describe('APIClient.isTokenExpired', () => {
  let client: APIClient;

  beforeEach(() => {
    client = new APIClient();
  });

  test('decodes a payload whose base64url contains multiple "-"/"_" chars', () => {
    // This payload encodes to a segment with several `-`/`_` characters.
    // The previous implementation only replaced the first of each, so atob()
    // received an invalid string (or decoded garbage) and expiry was wrong.
    const future = Math.floor(Date.now() / 1000) + 3600;
    const token = makeToken({
      userId: 12345,
      tenantId: 42,
      role: 'owner',
      exp: future,
      scope: 'a>?>?>?b~ff>>??>>?? padding to force url-unsafe chars',
    });

    // Sanity check: the payload segment really does contain >1 special char.
    const segment = token.split('.')[1];
    expect((segment.match(/[-_]/g) || []).length).toBeGreaterThan(1);

    expect(client.isTokenExpired(token)).toBe(false);
  });

  test('returns true for an expired token', () => {
    const past = Math.floor(Date.now() / 1000) - 10;
    expect(client.isTokenExpired(makeToken({ exp: past }))).toBe(true);
  });

  test('treats a token without exp as expired instead of throwing', () => {
    expect(client.isTokenExpired(makeToken({ userId: 1 }))).toBe(true);
  });

  test('treats an unparseable token as expired instead of throwing', () => {
    expect(client.isTokenExpired('not-a-jwt')).toBe(true);
    expect(client.isTokenExpired('only.two')).toBe(true);
  });
});

describe('APIClient verb helpers forward all arguments', () => {
  let client: APIClient;
  let fetchMock: jest.Mock;

  beforeEach(() => {
    client = new APIClient();
    client.setJwtChecker(() => null);
    client.setSiteIdCheck(() => ({ siteId: null }));
    client.setOnUpdateJwt(() => {});
    fetchMock = mockOkFetch();
  });

  test('put() forwards headers and abort signal to fetch', async () => {
    const controller = new AbortController();
    await client.put(
      '/projects/1',
      { name: 'x' },
      { clean: true },
      { 'X-Custom': 'yes' },
      controller.signal,
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe('PUT');
    expect((init.headers as Headers).get('X-Custom')).toBe('yes');
    expect(init.signal).toBe(controller.signal);
  });

  test('delete() forwards headers and abort signal to fetch', async () => {
    const controller = new AbortController();
    await client.delete(
      '/projects/1',
      undefined,
      { clean: true },
      { 'X-Custom': 'del' },
      controller.signal,
    );

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe('DELETE');
    expect((init.headers as Headers).get('X-Custom')).toBe('del');
    expect(init.signal).toBe(controller.signal);
  });

  test('patch() forwards headers and abort signal to fetch', async () => {
    const controller = new AbortController();
    await client.patch(
      '/projects/1',
      { name: 'y' },
      { clean: true },
      { 'X-Custom': 'patch' },
      controller.signal,
    );

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe('PATCH');
    expect((init.headers as Headers).get('X-Custom')).toBe('patch');
    expect(init.signal).toBe(controller.signal);
  });
});
