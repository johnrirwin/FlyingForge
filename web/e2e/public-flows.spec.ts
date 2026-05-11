/**
 * Playwright E2E: Public (unauthenticated) flows
 *
 * Covers:
 *   1. News feed loads and displays items
 *   2. Clicking a news item opens the detail modal
 *   3. Navigating to a protected route redirects to login
 *   6. Getting Started page loads
 *  23. Error state shown when API fails to load news
 */

import { test, expect } from '@playwright/test';
import {
  MOCK_AGGREGATED_RESPONSE,
  MOCK_SOURCES_RESPONSE,
} from './helpers/fixtures';

test.describe('Public flows (unauthenticated)', () => {
  test.beforeEach(async ({ page }) => {
    // Stub news API with mock data
    await page.route('**/api/items**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_AGGREGATED_RESPONSE),
      }),
    );

    await page.route('**/api/sources**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SOURCES_RESPONSE),
      }),
    );

    // Auth: no tokens → unauthenticated
    await page.route('**/api/auth/me**', (route) =>
      route.fulfill({ status: 401, body: '' }),
    );
  });

  test('news feed loads and displays items', async ({ page }) => {
    await page.goto('/news');

    await expect(page.getByText('Best FPV Motors of 2026')).toBeVisible();
    await expect(page.getByText('New DJI Goggles Review')).toBeVisible();
  });

  test('clicking a news item opens the detail modal', async ({ page }) => {
    await page.goto('/news');

    await page.getByText('Best FPV Motors of 2026').first().click();

    // The detail modal overlays the page; the title should appear in the modal too
    const modal = page.locator('[role="dialog"], .fixed.inset-0').last();
    await expect(modal).toBeVisible();
  });

  test('closing news detail modal with Escape key', async ({ page }) => {
    await page.goto('/news');

    // Open modal
    await page.getByText('Best FPV Motors of 2026').first().click();
    await expect(page.locator('[role="dialog"]').last()).toBeVisible();

    // Close with Escape
    await page.keyboard.press('Escape');

    // Modal should be gone
    await expect(page.locator('[role="dialog"]')).toHaveCount(0);
  });

  test('navigating to /dashboard redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/dashboard');

    await expect(page).toHaveURL(/\/login/);
  });

  test('navigating to /inventory redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/inventory');

    await expect(page).toHaveURL(/\/login/);
  });

  test('navigating to /aircraft redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/aircraft');

    await expect(page).toHaveURL(/\/login/);
  });

  test('Getting Started page loads', async ({ page }) => {
    // Stub any API calls the page might make
    await page.route('**/api/**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' }),
    );

    await page.goto('/getting-started');

    // Page should render without error (title or heading visible)
    await expect(page.locator('h1, h2').first()).toBeVisible();
  });

  test('error state shown when news API fails', async ({ page }) => {
    // Override the news stub to return an error
    await page.route('**/api/items**', (route) =>
      route.fulfill({ status: 500, body: JSON.stringify({ message: 'server error' }) }),
    );

    await page.goto('/news');

    await expect(page.getByText('Failed to Load Feed')).toBeVisible();
  });

  test('"/" hotkey focuses the search input on the news page', async ({ page }) => {
    await page.goto('/news');

    await expect(page.getByText('Best FPV Motors of 2026')).toBeVisible();

    // Ensure body has focus (not an input)
    await page.locator('body').click();

    await page.keyboard.press('/');

    // The search input should now be focused
    const searchInput = page.locator('input[type="text"]').first();
    await expect(searchInput).toBeFocused();
  });
});
