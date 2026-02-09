import { useState } from 'react';
import type { ComponentProps } from 'react';
import type { FeedItem, SourceInfo } from '../types';
import { FeedList } from './FeedList';
import { MobileFloatingControls } from './MobileFloatingControls';
import { TopBar } from './TopBar';

interface NewsPageProps {
  topBarProps: ComponentProps<typeof TopBar>;
  items: FeedItem[];
  sources: SourceInfo[];
  isLoading: boolean;
  isLoadingMore: boolean;
  error: string | null;
  totalCount: number;
  onItemClick: (item: FeedItem) => void;
  onLoadMore: () => void;
}

export function NewsPage({
  topBarProps,
  items,
  sources,
  isLoading,
  isLoadingMore,
  error,
  totalCount,
  onItemClick,
  onLoadMore,
}: NewsPageProps) {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  return (
    <div className="relative flex-1 min-h-0 flex flex-col overflow-hidden">
      <div className="hidden md:block flex-shrink-0">
        <TopBar {...topBarProps} isCollapsed={false} />
      </div>

      <div
        className="flex-1 min-h-0 overflow-y-auto"
        onScroll={(event) => {
          setIsMobileMenuOpen((prev) => (prev ? false : prev));

          // Dismiss keyboard only on touch/coarse-pointer devices and only
          // when a form control inside this scroll region is focused.
          if (typeof window === 'undefined') return;
          if (!window.matchMedia || !window.matchMedia('(pointer: coarse)').matches) return;

          const activeElement = document.activeElement;
          if (!(activeElement instanceof HTMLElement) || activeElement === document.body) return;

          const scrollContainer = event.currentTarget;
          if (!scrollContainer.contains(activeElement)) return;

          const tagName = activeElement.tagName;
          if (tagName === 'INPUT' || tagName === 'TEXTAREA' || tagName === 'SELECT') {
            activeElement.blur();
          }
        }}
      >
        <div className="md:hidden pt-20" />

        <FeedList
          items={items}
          sources={sources}
          isLoading={isLoading || isLoadingMore}
          error={error}
          onItemClick={onItemClick}
          hasMore={items.length < totalCount}
          onLoadMore={onLoadMore}
        />
      </div>

      <MobileFloatingControls
        label="News Feed Controls"
        isOpen={isMobileMenuOpen}
        onToggle={() => setIsMobileMenuOpen((prev) => !prev)}
      >
        <TopBar {...topBarProps} isCollapsed={false} />
      </MobileFloatingControls>
    </div>
  );
}
