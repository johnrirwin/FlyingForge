import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { render } from '../test/test-utils'
import { FeedCard } from './FeedCard'
import type { FeedItem, SourceInfo } from '../types'

// Helper to create test feed items
function createFeedItem(overrides: Partial<FeedItem> = {}): FeedItem {
  return {
    id: 'test-id-1',
    title: 'Test Feed Item Title',
    url: 'https://example.com/article',
    source: 'test-source',
    sourceType: 'rss',
    tags: ['fpv', 'drone'],
    ...overrides,
  }
}

function createSourceInfo(overrides: Partial<SourceInfo> = {}): SourceInfo {
  return {
    id: 'source-1',
    name: 'Test Source',
    url: 'https://example.com',
    sourceType: 'rss',
    description: 'Test source description',
    feedType: 'rss',
    enabled: true,
    ...overrides,
  }
}

describe('FeedCard', () => {
  it('renders title correctly', () => {
    const item = createFeedItem({ title: 'FPV Drone Racing Championship' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('FPV Drone Racing Championship')).toBeInTheDocument()
  })

  it('renders source badge', () => {
    const item = createFeedItem({ source: 'rotorbuilds' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('rotorbuilds')).toBeInTheDocument()
  })

  it('renders source name from sourceInfo when provided', () => {
    const item = createFeedItem({ source: 'rb' })
    const source = createSourceInfo({ name: 'Rotor Builds' })
    const onClick = vi.fn()

    render(<FeedCard item={item} source={source} onClick={onClick} />)

    expect(screen.getByText('Rotor Builds')).toBeInTheDocument()
  })

  it('renders author when provided', () => {
    const item = createFeedItem({ author: 'TestPilot123' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('by TestPilot123')).toBeInTheDocument()
  })

  it('renders summary when provided', () => {
    const item = createFeedItem({ summary: 'This is a test summary for the article.' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('This is a test summary for the article.')).toBeInTheDocument()
  })

  it('strips html from summary', () => {
    const item = createFeedItem({ summary: '<p>News <strong>summary</strong> &amp; details</p>' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('News summary & details')).toBeInTheDocument()
    expect(screen.queryByText(/<strong>/)).not.toBeInTheDocument()
  })

  it('renders score when provided', () => {
    const item = createFeedItem({ score: 42 })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('renders tags', () => {
    const item = createFeedItem({ tags: ['fpv', 'racing', 'betaflight'] })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    expect(screen.getByText('fpv')).toBeInTheDocument()
    expect(screen.getByText('racing')).toBeInTheDocument()
    expect(screen.getByText('betaflight')).toBeInTheDocument()
  })

  it('shows overflow indicator for many tags', () => {
    const item = createFeedItem({
      tags: ['fpv', 'racing', 'betaflight', 'freestyle', 'cinematic', 'long-range'],
    })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    // Should show first 4 tags and "+2"
    expect(screen.getByText('fpv')).toBeInTheDocument()
    expect(screen.getByText('racing')).toBeInTheDocument()
    expect(screen.getByText('betaflight')).toBeInTheDocument()
    expect(screen.getByText('freestyle')).toBeInTheDocument()
    expect(screen.getByText('+2')).toBeInTheDocument()
    expect(screen.queryByText('cinematic')).not.toBeInTheDocument()
  })

  it('calls onClick when clicked', () => {
    const item = createFeedItem()
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    const article = screen.getByRole('article')
    fireEvent.click(article)

    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('renders rss source with blue badge styling', () => {
    const item = createFeedItem({ sourceType: 'rss' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    const badge = screen.getByText('test-source')
    expect(badge).toHaveClass('bg-blue-500/20', 'text-blue-400')
  })

  it('renders youtube source with red badge styling', () => {
    const item = createFeedItem({ sourceType: 'youtube' })
    const onClick = vi.fn()

    render(<FeedCard item={item} onClick={onClick} />)

    const badge = screen.getByText('test-source')
    expect(badge).toHaveClass('bg-red-500/20', 'text-red-400')
  })

  it('renders image when media is provided', () => {
    const item = createFeedItem({
      media: { imageUrl: 'https://example.com/image.jpg' },
    })
    const onClick = vi.fn()

    const { container } = render(<FeedCard item={item} onClick={onClick} />)

    // Image has alt="" so it has role="presentation", use querySelector
    const img = container.querySelector('img')
    expect(img).toBeInTheDocument()
    expect(img).toHaveAttribute('src', 'https://example.com/image.jpg')
  })

  it('does not render image when no media is provided', () => {
    const item = createFeedItem({ media: undefined })
    const onClick = vi.fn()

    const { container } = render(<FeedCard item={item} onClick={onClick} />)

    expect(container.querySelector('img')).not.toBeInTheDocument()
  })
})
