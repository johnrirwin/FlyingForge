import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { render } from '../test/test-utils'
import { Sidebar } from './Sidebar'
import type { SourceInfo } from '../types'

function createSource(overrides: Partial<SourceInfo> = {}): SourceInfo {
  return {
    id: 'source-1',
    name: 'Test Source',
    url: 'https://example.com',
    sourceType: 'news',
    description: 'Test description',
    feedType: 'rss',
    enabled: true,
    ...overrides,
  }
}

describe('Sidebar', () => {
  const defaultProps = {
    sources: [
      createSource({ id: 'news-1', name: 'News Source 1', sourceType: 'news' }),
      createSource({ id: 'news-2', name: 'News Source 2', sourceType: 'news' }),
      createSource({ id: 'community-1', name: 'Community Source', sourceType: 'community' }),
    ],
    selectedSources: [],
    sourceType: 'all' as const,
    onToggleSource: vi.fn(),
    onSourceTypeChange: vi.fn(),
    isLoading: false,
  }

  it('renders the FlyingForge heading', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByRole('heading', { name: /FlyingForge/i })).toBeInTheDocument()
  })

  it('renders source type toggle buttons', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('All')).toBeInTheDocument()
    expect(screen.getByText('News')).toBeInTheDocument()
    expect(screen.getByText('Community')).toBeInTheDocument()
  })

  it('shows source count summary', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('2 news • 1 community')).toBeInTheDocument()
  })

  it('renders all sources when sourceType is all', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('News Source 1')).toBeInTheDocument()
    expect(screen.getByText('News Source 2')).toBeInTheDocument()
    expect(screen.getByText('Community Source')).toBeInTheDocument()
  })

  it('filters sources by type', () => {
    render(<Sidebar {...defaultProps} sourceType="news" />)
    expect(screen.getByText('News Source 1')).toBeInTheDocument()
    expect(screen.getByText('News Source 2')).toBeInTheDocument()
    expect(screen.queryByText('Community Source')).not.toBeInTheDocument()
  })

  it('calls onSourceTypeChange when type button clicked', () => {
    const onSourceTypeChange = vi.fn()
    render(<Sidebar {...defaultProps} onSourceTypeChange={onSourceTypeChange} />)

    fireEvent.click(screen.getByText('News'))
    expect(onSourceTypeChange).toHaveBeenCalledWith('news')
  })

  it('calls onToggleSource when source clicked', () => {
    const onToggleSource = vi.fn()
    render(<Sidebar {...defaultProps} onToggleSource={onToggleSource} />)

    fireEvent.click(screen.getByText('News Source 1'))
    expect(onToggleSource).toHaveBeenCalledWith('news-1')
  })

  it('shows loading skeletons when isLoading', () => {
    const { container } = render(<Sidebar {...defaultProps} isLoading={true} />)

    // Should show skeleton elements instead of sources
    const skeletons = container.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it('highlights selected source', () => {
    render(<Sidebar {...defaultProps} selectedSources={['news-1']} />)

    const sourceButton = screen.getByText('News Source 1').closest('button')
    expect(sourceButton).toHaveClass('bg-primary-600/20')
  })

  it('highlights active source type button', () => {
    render(<Sidebar {...defaultProps} sourceType="community" />)

    const communityButton = screen.getByText('Community')
    expect(communityButton).toHaveClass('bg-primary-600')
  })
})
