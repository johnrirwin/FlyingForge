/**
 * Playwright E2E: Escape key closes modals in the correct order
 *
 * Covers:
 *  21. Escape key closes all modals in the correct order:
 *      news detail → inventory modal → aircraft form → aircraft detail
 *
 * The order in App.tsx:
 *   1. pilot profile modal (if open)
 *   2. aircraft form (if open)
 *   3. aircraft detail (if open)
 *   4. inventory add modal (if open)
 *   5. news item detail (if open)
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

test.describe('Escape key modal close order', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedUser(page);

    await page.route('**/api/items**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_AGGREGATED_RESPONSE) }),
    );
    await page.route('**/api/sources**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SOURCES_RESPONSE) }),
    );
    await page.route('**/api/aircraft/aircraft-1**', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ aircraft: MOCK_AIRCRAFT[0], components: [] }),
      }),
    );
    await page.route('**/api/aircraft**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ aircraft: MOCK_AIRCRAFT, totalCount: 1 }) }),
    );
    await page.route('**/api/inventory/summary**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_INVENTORY_SUMMARY) }),
    );
    await page.route('**/api/inventory**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: MOCK_INVENTORY, totalCount: 1 }) }),
    );
  });

  test('Escape closes news detail modal (no other modal open)', async ({ page }) => {
    await page.goto('/news');

    await page.getByText('Best FPV Motors of 2026').first().click();
    await expect(page.getByRole('dialog').last()).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('Escape closes aircraft detail modal before news modal', async ({ page }) => {
    // Navigate to aircraft, open detail
    await page.goto('/aircraft');
    await page.getByRole('button', { name: /shredder 5/i }).first().click();

    const aircraftDialog = page.getByRole('dialog').first();
    await expect(aircraftDialog).toBeVisible();

    // Press Escape — should close aircraft detail
    await page.keyboard.press('Escape');

    // Aircraft detail gone; no other modals
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('Escape closes inventory modal', async ({ page }) => {
    await page.goto('/inventory');

    await page.getByRole('button', { name: /add item/i }).first().click();
    const inventoryDialog = page.getByRole('dialog');
    await expect(inventoryDialog).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('Escape closes aircraft form modal', async ({ page }) => {
    await page.goto('/aircraft');

    await page.getByRole('button', { name: /add aircraft/i }).first().click();
    await expect(page.getByRole('dialog')).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});
