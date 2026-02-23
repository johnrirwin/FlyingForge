import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { render } from '../test/test-utils'
import { ItemDetail } from './ItemDetail'
import type { FeedItem } from '../types'

function createFeedItem(overrides: Partial<FeedItem> = {}): FeedItem {
  return {
    id: 'item-1',
    title: 'Test item',
    url: 'https://example.com/original',
    source: 'test-source',
    sourceType: 'rss',
    tags: ['fpv'],
    ...overrides,
  }
}

describe('ItemDetail', () => {
  it('strips html from summary and content text', () => {
    const item = createFeedItem({
      summary: '<p>Summary with <strong>HTML</strong> &amp; entities</p>',
      contentText: '<div>Full <em>content</em> body</div>',
    })

    render(<ItemDetail item={item} onClose={vi.fn()} />)

    expect(screen.getByText('Summary with HTML & entities')).toBeInTheDocument()
    expect(screen.getByText('Full content body')).toBeInTheDocument()
  })

  it('shows "Read Original" for non-video items', () => {
    const item = createFeedItem()

    render(<ItemDetail item={item} onClose={vi.fn()} />)

    expect(screen.getByRole('link', { name: 'Read Original' })).toHaveAttribute('href', item.url)
  })

  it('shows "View Video" for YouTube items', () => {
    const item = createFeedItem({ sourceType: 'youtube' })

    render(<ItemDetail item={item} onClose={vi.fn()} />)

    expect(screen.getByRole('link', { name: 'View Video' })).toHaveAttribute('href', item.url)
  })

  it('shows "View Video" for items marked as video media', () => {
    const item = createFeedItem({
      media: { type: 'video', videoUrl: 'https://example.com/video' },
    })

    render(<ItemDetail item={item} onClose={vi.fn()} />)

    expect(screen.getByRole('link', { name: 'View Video' })).toHaveAttribute('href', item.url)
  })
})
