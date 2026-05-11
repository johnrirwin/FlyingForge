/**
 * Playwright E2E: Social / Pilots page
 *
 * Covers:
 *  18. Social/Pilots page loads; clicking a pilot opens the profile modal
 *  19. Pilot profile modal closes on Escape key
 */

import { test, expect } from '@playwright/test';
import { setupAuthenticatedUser } from './helpers/auth';
import {
  MOCK_AGGREGATED_RESPONSE,
  MOCK_SOURCES_RESPONSE,
} from './helpers/fixtures';

const MOCK_PILOT = {
  id: 'pilot-1',
  callSign: 'SkyFox',
  displayName: 'Alex Fox',
  aircraftCount: 3,
  buildCount: 2,
  followerCount: 10,
  isFollowing: false,
};

test.describe('Social / Pilots', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedUser(page);

    await page.route('**/api/items**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_AGGREGATED_RESPONSE) }),
    );
    await page.route('**/api/sources**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SOURCES_RESPONSE) }),
    );
    await page.route('**/api/aircraft**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ aircraft: [], totalCount: 0 }) }),
    );
    await page.route('**/api/inventory/summary**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ totalItems: 0, totalValue: 0, byCategory: {} }) }),
    );
    await page.route('**/api/inventory**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [], totalCount: 0 }) }),
    );

    // Discover pilots returns our mock pilot
    await page.route('**/api/pilots/discover**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ pilots: [MOCK_PILOT], totalCount: 1 }),
      }),
    );
    await page.route('**/api/pilots**', (route) => {
      const url = route.request().url();
      if (url.includes('/discover')) {
        route.continue();
      } else {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) });
      }
    });
    await page.route('**/api/social/following**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) }),
    );
    await page.route('**/api/social/followers**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ pilots: [], totalCount: 0 }) }),
    );
  });

  test('social page loads and displays pilot cards', async ({ page }) => {
    await page.goto('/social');

    await expect(page.getByText('SkyFox')).toBeVisible();
  });

  test('clicking a pilot card opens the profile modal', async ({ page }) => {
    // Stub pilot profile API
    await page.route(`**/api/pilots/pilot-1**`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ pilot: { ...MOCK_PILOT, builds: [], aircraft: [] } }),
      }),
    );

    await page.goto('/social');

    // Click the pilot card
    await page.getByRole('button', { name: /skyfox/i }).click();

    // The pilot profile modal should open
    const modal = page.getByRole('dialog', { name: /pilot profile/i });
    await expect(modal).toBeVisible();
  });

  test('pilot profile modal closes on Escape key', async ({ page }) => {
    await page.route(`**/api/pilots/pilot-1**`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ pilot: { ...MOCK_PILOT, builds: [], aircraft: [] } }),
      }),
    );

    await page.goto('/social');

    // Open modal
    await page.getByRole('button', { name: /skyfox/i }).click();
    await expect(page.getByRole('dialog', { name: /pilot profile/i })).toBeVisible();

    // Close with Escape
    await page.keyboard.press('Escape');

    await expect(page.getByRole('dialog', { name: /pilot profile/i })).toHaveCount(0);
  });

  test('pilot profile modal close button dismisses the modal', async ({ page }) => {
    await page.route(`**/api/pilots/pilot-1**`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ pilot: { ...MOCK_PILOT, builds: [], aircraft: [] } }),
      }),
    );

    await page.goto('/social');

    await page.getByRole('button', { name: /skyfox/i }).click();

    const modal = page.getByRole('dialog', { name: /pilot profile/i });
    await expect(modal).toBeVisible();

    // Click close button
    await modal.getByRole('button', { name: /close pilot profile modal/i }).click();

    await expect(page.getByRole('dialog', { name: /pilot profile/i })).toHaveCount(0);
  });
});
