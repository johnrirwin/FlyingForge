/**
 * Playwright auth helpers.
 *
 * Use `setupAuthenticatedUser` to inject a fake JWT into localStorage and
 * stub the `/api/auth/me` endpoint so the AuthProvider accepts the session.
 */

import type { Page } from '@playwright/test';

export const MOCK_USER = {
  id: 'user-1',
  email: 'pilot@example.com',
  displayName: 'Test Pilot',
  status: 'active',
  emailVerified: true,
  isAdmin: false,
  isContentAdmin: false,
  isGearAdmin: false,
  createdAt: '2026-01-01T00:00:00Z',
};

export const MOCK_TOKENS = {
  accessToken: 'fake-access-token',
  refreshToken: 'fake-refresh-token',
  tokenType: 'Bearer',
  expiresIn: 3600,
};

/**
 * Seed localStorage with auth tokens before navigation, then stub `/api/auth/me`
 * so the AuthProvider resolves the user without a real backend.
 */
export async function setupAuthenticatedUser(page: Page) {
  // Stub the me endpoint before the page loads
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    }),
  );

  // Set tokens in localStorage before navigation
  await page.addInitScript((tokens) => {
    localStorage.setItem('auth_tokens', JSON.stringify(tokens));
  }, MOCK_TOKENS);
}

/**
 * Stub a GET endpoint with a JSON body.
 */
export async function stubGet(page: Page, urlPattern: string, body: unknown) {
  await page.route(`**${urlPattern}**`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    }),
  );
}
