import { act, render } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AUTH_EXPIRED_EVENT } from '../authRouting';
import { AuthProvider } from './AuthContext';

function createResponse(status: number) {
  return new Response(JSON.stringify({}), {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

function createJsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

describe('AuthProvider fetch 401 handling', () => {
  const nativeFetch = window.fetch;

  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  afterEach(() => {
    window.fetch = nativeFetch;
  });

  it('dispatches auth-expired once and clears tokens when authorized requests return 401', async () => {
    const baseFetch = vi.fn(async () => createResponse(401));
    window.fetch = baseFetch as unknown as typeof window.fetch;

    const authExpiredListener = vi.fn();
    window.addEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);

    const view = render(
      <AuthProvider>
        <div>child</div>
      </AuthProvider>,
    );

    localStorage.setItem('access_token', 'token-a');
    localStorage.setItem('refresh_token', 'token-b');

    await act(async () => {
      await expect(window.fetch('/api/inventory', {
        headers: {
          Authorization: 'Bearer token-a',
        },
      })).rejects.toMatchObject({ name: 'SessionExpiredError' });
    });

    await act(async () => {
      await expect(window.fetch('/api/inventory', {
        headers: {
          Authorization: 'Bearer token-a',
        },
      })).rejects.toMatchObject({ name: 'SessionExpiredError' });
    });

    expect(authExpiredListener).toHaveBeenCalledTimes(1);
    expect(localStorage.getItem('access_token')).toBeNull();
    expect(localStorage.getItem('refresh_token')).toBeNull();

    window.removeEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);
    view.unmount();
  });

  it('does not trigger auth-expired flow for unauthenticated requests', async () => {
    const baseFetch = vi.fn(async () => createResponse(401));
    window.fetch = baseFetch as unknown as typeof window.fetch;

    const authExpiredListener = vi.fn();
    window.addEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);

    const view = render(
      <AuthProvider>
        <div>child</div>
      </AuthProvider>,
    );

    await act(async () => {
      await expect(window.fetch('/api/news')).resolves.toBeInstanceOf(Response);
    });
    expect(authExpiredListener).not.toHaveBeenCalled();

    window.removeEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);
    view.unmount();
  });

  it('refreshes tokens and retries authorized requests when access token expires', async () => {
    localStorage.setItem('access_token', 'token-a');
    localStorage.setItem('refresh_token', 'token-b');

    const baseFetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.toString()
          : input.url;

      const pathname = new URL(url, 'http://localhost').pathname;

      if (pathname === '/api/auth/me') {
        return createJsonResponse(200, { id: 'user-1' });
      }

      if (pathname === '/api/auth/refresh') {
        return createJsonResponse(200, {
          accessToken: 'token-new',
          refreshToken: 'refresh-new',
          tokenType: 'Bearer',
          expiresIn: 900,
        });
      }

      if (pathname === '/api/inventory') {
        const authHeader = input instanceof Request ? input.headers.get('Authorization') : null;
        if (authHeader === 'Bearer token-new') {
          return createJsonResponse(200, { ok: true });
        }
        return createResponse(401);
      }

      return createJsonResponse(200, {});
    });

    window.fetch = baseFetch as unknown as typeof window.fetch;

    const authExpiredListener = vi.fn();
    window.addEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);

    const view = render(
      <AuthProvider>
        <div>child</div>
      </AuthProvider>,
    );

    await act(async () => {
      await expect(window.fetch('/api/inventory', {
        headers: {
          Authorization: 'Bearer token-a',
        },
      })).resolves.toBeInstanceOf(Response);
    });

    expect(authExpiredListener).not.toHaveBeenCalled();
    expect(localStorage.getItem('access_token')).toBe('token-new');
    expect(localStorage.getItem('refresh_token')).toBe('refresh-new');

    window.removeEventListener(AUTH_EXPIRED_EVENT, authExpiredListener as EventListener);
    view.unmount();
  });

  it('restores the original fetch when the last AuthProvider unmounts', () => {
    const baseFetch = vi.fn(async () => createResponse(200));
    window.fetch = baseFetch as unknown as typeof window.fetch;

    const first = render(
      <AuthProvider>
        <div>first</div>
      </AuthProvider>,
    );

    const wrappedFetch = window.fetch;
    expect(wrappedFetch).not.toBe(baseFetch);

    const second = render(
      <AuthProvider>
        <div>second</div>
      </AuthProvider>,
    );

    expect(window.fetch).toBe(wrappedFetch);

    first.unmount();
    expect(window.fetch).toBe(wrappedFetch);

    second.unmount();
    expect(window.fetch).toBe(baseFetch);
  });
});
