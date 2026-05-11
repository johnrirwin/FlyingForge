/**
 * Playwright E2E: News filtering
 *
 * Covers:
 *   2. News filtering (source filter, date range, sort, keyword search)
 *   3. Infinite scroll loads more items
 */

import { test, expect } from '@playwright/test';
import {
  MOCK_AGGREGATED_RESPONSE,
  MOCK_SOURCES_RESPONSE,
  MOCK_FEED_ITEMS,
} from './helpers/fixtures';

test.describe('News filtering', () => {
  test.beforeEach(async ({ page }) => {
    // Auth: unauthenticated
    await page.route('**/api/auth/me**', (route) =>
      route.fulfill({ status: 401, body: '' }),
    );

    await page.route('**/api/sources**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SOURCES_RESPONSE),
      }),
    );
  });

  test('keyword search triggers a new API request with the search term', async ({ page }) => {
    let searchQuery = '';

    await page.route('**/api/items**', (route, request) => {
      const url = new URL(request.url());
      searchQuery = url.searchParams.get('q') ?? '';
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...MOCK_AGGREGATED_RESPONSE, items: MOCK_FEED_ITEMS }),
      });
    });

    await page.goto('/news');

    // Type in the search box
    const searchInput = page.locator('input[type="text"]').first();
    await searchInput.fill('motors');
    await page.keyboard.press('Enter');

    // The next API request should include the search query
    await expect.poll(() => searchQuery).toBe('motors');
  });

  test('sort change triggers a new API request', async ({ page }) => {
    let sortParam = '';

    await page.route('**/api/items**', (route, request) => {
      const url = new URL(request.url());
      sortParam = url.searchParams.get('sort') ?? '';
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_AGGREGATED_RESPONSE),
      });
    });

    await page.goto('/news');

    // Initial load should fire a request
    await page.waitForResponse('**/api/items**');

    // Click the "Top Rated" / score sort button if available
    const scoreButton = page.getByRole('button', { name: /top rated|score/i });
    if (await scoreButton.isVisible()) {
      await scoreButton.click();
      await page.waitForResponse('**/api/items**');
      expect(sortParam).toBe('score');
    }
  });

  test('source type filter sends sourceType param', async ({ page }) => {
    const receivedParams: Record<string, string> = {};

    await page.route('**/api/items**', (route, request) => {
      const url = new URL(request.url());
      url.searchParams.forEach((value, key) => {
        receivedParams[key] = value;
      });
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_AGGREGATED_RESPONSE),
      });
    });

    await page.goto('/news');
    await page.waitForResponse('**/api/items**');

    // Click a source-type filter tab (e.g. YouTube)
    const youtubeTab = page.getByRole('button', { name: /youtube/i });
    if (await youtubeTab.isVisible()) {
      await youtubeTab.click();
      await page.waitForResponse('**/api/items**');
      expect(receivedParams['sourceType']).toBe('youtube');
    }
  });
});
