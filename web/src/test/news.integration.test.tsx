/**
 * Integration tests: News feed
 * Covers:
 *   - News feed renders items (scenario 1)
 *   - Clicking an item fires onItemClick (scenario 4)
 *   - Error state displayed when API fails (scenario 23)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from './test-utils';
import { NewsPage } from '../components/NewsPage';
import type { FeedItem, SourceInfo } from '../types';

const source: SourceInfo = {
  id: 'src-1',
  name: 'FPV Blog',
  url: 'https://fpvblog.example.com',
  sourceType: 'rss',
  description: 'FPV news',
  feedType: 'rss',
  enabled: true,
};

const item1: FeedItem = {
  id: 'item-1',
  title: 'Best FPV Motors of 2026',
  url: 'https://fpvblog.example.com/motors',
  source: 'src-1',
  sourceType: 'rss',
  publishedAt: '2026-03-01T12:00:00Z',
  summary: 'A roundup of the best motors for freestyle and racing.',
  tags: ['motors'],
};

const item2: FeedItem = {
  id: 'item-2',
  title: 'New DJI Goggles Review',
  url: 'https://fpvblog.example.com/goggles',
  source: 'src-1',
  sourceType: 'rss',
  publishedAt: '2026-03-02T12:00:00Z',
  summary: 'We test the latest DJI goggles.',
  tags: ['goggles'],
};

function makeTopBarProps() {
  return {
    query: '',
    onQueryChange: vi.fn(),
    onSearch: vi.fn(),
    fromDate: '',
    toDate: '',
    onFromDateChange: vi.fn(),
    onToDateChange: vi.fn(),
    sort: 'newest' as const,
    onSortChange: vi.fn(),
    sourceType: 'all' as const,
    onSourceTypeChange: vi.fn(),
    totalCount: 2,
  };
}

describe('NewsPage – news feed integration', () => {
  it('renders feed items with their titles', () => {
    render(
      <NewsPage
        topBarProps={makeTopBarProps()}
        items={[item1, item2]}
        sources={[source]}
        isLoading={false}
        isLoadingMore={false}
        error={null}
        totalCount={2}
        onItemClick={vi.fn()}
        onLoadMore={vi.fn()}
      />,
    );

    expect(screen.getByText('Best FPV Motors of 2026')).toBeInTheDocument();
    expect(screen.getByText('New DJI Goggles Review')).toBeInTheDocument();
  });

  it('calls onItemClick when a feed card is clicked', () => {
    const onItemClick = vi.fn();

    render(
      <NewsPage
        topBarProps={makeTopBarProps()}
        items={[item1]}
        sources={[source]}
        isLoading={false}
        isLoadingMore={false}
        error={null}
        totalCount={1}
        onItemClick={onItemClick}
        onLoadMore={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText('Best FPV Motors of 2026'));
    expect(onItemClick).toHaveBeenCalledWith(item1);
  });

  it('shows error state when API fails', () => {
    render(
      <NewsPage
        topBarProps={makeTopBarProps()}
        items={[]}
        sources={[]}
        isLoading={false}
        isLoadingMore={false}
        error="Failed to load items"
        totalCount={0}
        onItemClick={vi.fn()}
        onLoadMore={vi.fn()}
      />,
    );

    expect(screen.getByText('Failed to Load Feed')).toBeInTheDocument();
    expect(screen.getByText('Failed to load items')).toBeInTheDocument();
  });

  it('shows loading skeleton when isLoading and no items', () => {
    const { container } = render(
      <NewsPage
        topBarProps={makeTopBarProps()}
        items={[]}
        sources={[]}
        isLoading={true}
        isLoadingMore={false}
        error={null}
        totalCount={0}
        onItemClick={vi.fn()}
        onLoadMore={vi.fn()}
      />,
    );

    // Skeleton cards have animate-pulse class
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0);
  });

  it('does not call onItemClick when not clicked', () => {
    const onItemClick = vi.fn();

    render(
      <NewsPage
        topBarProps={makeTopBarProps()}
        items={[item1, item2]}
        sources={[source]}
        isLoading={false}
        isLoadingMore={false}
        error={null}
        totalCount={2}
        onItemClick={onItemClick}
        onLoadMore={vi.fn()}
      />,
    );

    expect(onItemClick).not.toHaveBeenCalled();
  });
});
