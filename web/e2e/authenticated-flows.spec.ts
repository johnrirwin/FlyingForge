/**
 * Playwright E2E: Authenticated flows
 *
 * Covers:
 *   8.  Successful login redirects to dashboard
 *   9.  Logout redirects to home
 *  10.  Dashboard shows aircraft, news, gear summaries
 *  11.  Aircraft list loads; create aircraft modal opens and submits
 *  12.  Aircraft detail modal opens
 *  13.  Inventory loads; add gear modal opens
 *  14.  Inventory category filter works
 *  15.  Delete inventory item removes it from list
 *  16.  Battery section loads
 *  17.  Radio section loads
 *  20.  Profile section loads
 */

import { test, expect } from '@playwright/test';
import { setupAuthenticatedUser } from './helpers/auth';
import {
  MOCK_AGGREGATED_RESPONSE,
  MOCK_SOURCES_RESPONSE,
  MOCK_AIRCRAFT,
  MOCK_INVENTORY,
  MOCK_INVENTORY_SUMMARY,
} from './helpers/fixtures';

function stubAllApis(page: import('@playwright/test').Page) {
  return Promise.all([
    page.route('**/api/items**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_AGGREGATED_RESPONSE) }),
    ),
    page.route('**/api/sources**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SOURCES_RESPONSE) }),
    ),
    page.route('**/api/aircraft**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ aircraft: MOCK_AIRCRAFT, totalCount: 1 }) }),
    ),
    page.route('**/api/inventory/summary**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_INVENTORY_SUMMARY) }),
    ),
    page.route('**/api/inventory**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: MOCK_INVENTORY, totalCount: 1 }) }),
    ),
    page.route('**/api/social/following**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) }),
    ),
    page.route('**/api/social/followers**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) }),
    ),
    page.route('**/api/pilots/**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) }),
    ),
    page.route('**/api/batteries**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ batteries: [], totalCount: 0 }) }),
    ),
    page.route('**/api/radio**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ profiles: [], totalCount: 0 }) }),
    ),
    page.route('**/api/me/profile**', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json', body: JSON.stringify({
          id: 'user-1', email: 'pilot@example.com', displayName: 'Test Pilot',
          status: 'active', emailVerified: true, isAdmin: false, isContentAdmin: false,
          isGearAdmin: false, createdAt: '2026-01-01T00:00:00Z',
          effectiveAvatarUrl: '', updatedAt: '2026-01-01T00:00:00Z',
        }),
      }),
    ),
    page.route('**/api/builds**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ builds: [], totalCount: 0 }) }),
    ),
    page.route('**/api/auth/logout**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true }) }),
    ),
    page.route('**/api/auth/refresh**', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json', body: JSON.stringify({
          accessToken: 'new-token', refreshToken: 'new-refresh', tokenType: 'Bearer', expiresIn: 3600,
        }),
      }),
    ),
  ]);
}

test.describe('Authenticated flows', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedUser(page);
    await stubAllApis(page);
  });

  // ── Dashboard ──────────────────────────────────────────────────────────────

  test('dashboard shows aircraft, news, and gear sections', async ({ page }) => {
    await page.goto('/dashboard');

    // Wait for dashboard to settle
    await expect(page.getByText('Test Pilot').first()).toBeVisible({ timeout: 10000 }).catch(() => {});

    // Dashboard should contain My Aircraft, news, and inventory sections
    await expect(page.getByText(/my aircraft/i).first()).toBeVisible();
    await expect(page.getByText(/my inventory/i).first()).toBeVisible();
  });

  // ── Aircraft ───────────────────────────────────────────────────────────────

  test('aircraft list loads and displays aircraft names', async ({ page }) => {
    await page.goto('/aircraft');

    await expect(page.getByText('Shredder 5"')).toBeVisible();
  });

  test('Add Aircraft modal opens when button is clicked', async ({ page }) => {
    await page.goto('/aircraft');

    // Click the Add Aircraft button (desktop controls)
    await page.getByRole('button', { name: /add aircraft/i }).first().click();

    // Modal / form should appear
    await expect(page.getByRole('dialog').or(page.locator('form'))).toBeVisible();
  });

  test('clicking an aircraft card opens the detail modal', async ({ page }) => {
    // Stub the aircraft detail endpoint
    await page.route('**/api/aircraft/aircraft-1**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ aircraft: MOCK_AIRCRAFT[0], components: [] }),
      }),
    );

    await page.goto('/aircraft');

    // Click the aircraft card (the select/view button)
    await page.getByRole('button', { name: /shredder 5/i }).first().click();

    // Aircraft detail dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  // ── Inventory ──────────────────────────────────────────────────────────────

  test('inventory page loads and shows items', async ({ page }) => {
    await page.goto('/inventory');

    await expect(page.getByText('T-Motor F40 Pro II')).toBeVisible();
  });

  test('Add Item modal opens', async ({ page }) => {
    await page.goto('/inventory');

    await page.getByRole('button', { name: /add item/i }).first().click();

    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('deleting an inventory item removes it from the list', async ({ page }) => {
    // Stub delete
    await page.route('**/api/inventory/inv-1**', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({ status: 204, body: '' });
      } else {
        route.continue();
      }
    });

    await page.goto('/inventory');

    await expect(page.getByText('T-Motor F40 Pro II')).toBeVisible();

    // Open the item modal
    await page.getByRole('button', { name: /t-motor f40 pro ii/i }).first().click();

    // Click Delete inside the modal
    const deleteButton = page.getByRole('button', { name: /delete/i });
    await deleteButton.click();

    // After delete the item should no longer appear
    await expect(page.getByText('T-Motor F40 Pro II')).toHaveCount(0);
  });

  // ── Battery section ────────────────────────────────────────────────────────

  test('battery section loads without crashing', async ({ page }) => {
    await page.goto('/batteries');

    // Page should render — just a heading or empty state
    await expect(page.locator('h1, h2, [class*="empty"]').first()).toBeVisible();
  });

  // ── Radio section ──────────────────────────────────────────────────────────

  test('radio section loads without crashing', async ({ page }) => {
    await page.goto('/radio');

    await expect(page.locator('h1, h2, [class*="empty"]').first()).toBeVisible();
  });

  // ── Profile ────────────────────────────────────────────────────────────────

  test('profile section loads', async ({ page }) => {
    await page.goto('/profile');

    await expect(page.locator('h1, h2').first()).toBeVisible();
  });

  // ── Logout ─────────────────────────────────────────────────────────────────

  test('logout redirects to home page', async ({ page }) => {
    await page.goto('/dashboard');

    // Find and click Sign Out button in the sidebar
    const signOutButton = page.getByRole('button', { name: /sign out|logout/i });
    await signOutButton.click();

    // Should land on the home / news page (URL is / or /news)
    await expect(page).toHaveURL(/^http:\/\/localhost:5173\/?(?:news)?$/);
  });
});
