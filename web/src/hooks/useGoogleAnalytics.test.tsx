import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'

// Mock import.meta.env before importing the module
const mockEnv = {
  VITE_GA_MEASUREMENT_ID: 'G-TEST123456',
}

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useLocation: vi.fn(() => ({ pathname: '/', search: '' })),
  }
})

// We need to dynamically import to control the env mock
let useGoogleAnalytics: typeof import('./useGoogleAnalytics').useGoogleAnalytics
let trackPageView: typeof import('./useGoogleAnalytics').trackPageView
let trackEvent: typeof import('./useGoogleAnalytics').trackEvent

describe('useGoogleAnalytics', () => {
  let originalDataLayer: unknown[] | undefined
  let originalGtag: ((...args: unknown[]) => void) | undefined
  let appendChildSpy: ReturnType<typeof vi.spyOn>

  beforeEach(async () => {
    // Store originals
    originalDataLayer = window.dataLayer
    originalGtag = window.gtag

    // Reset window properties
    delete (window as { dataLayer?: unknown[] }).dataLayer
    delete (window as { gtag?: (...args: unknown[]) => void }).gtag

    // Mock document.head.appendChild to prevent actual script injection
    appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      // Simulate script load after a short delay
      if (node instanceof HTMLScriptElement && node.onload) {
        setTimeout(() => {
          node.onload?.(new Event('load'))
        }, 10)
      }
      return node
    })

    // Clear module cache and re-import with fresh state
    vi.resetModules()

    // Mock the env
    vi.stubEnv('VITE_GA_MEASUREMENT_ID', mockEnv.VITE_GA_MEASUREMENT_ID)

    // Re-import the module
    const module = await import('./useGoogleAnalytics')
    useGoogleAnalytics = module.useGoogleAnalytics
    trackPageView = module.trackPageView
    trackEvent = module.trackEvent
  })

  afterEach(() => {
    // Restore originals
    if (originalDataLayer !== undefined) {
      window.dataLayer = originalDataLayer
    }
    if (originalGtag !== undefined) {
      window.gtag = originalGtag
    }
    appendChildSpy.mockRestore()
    vi.unstubAllEnvs()
    vi.clearAllMocks()
  })

  const wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter>{children}</MemoryRouter>
  )

  describe('initialization', () => {
    it('initializes GA and creates gtag function on mount', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      // gtag should be defined
      expect(window.gtag).toBeDefined()
      expect(typeof window.gtag).toBe('function')
    })

    it('creates dataLayer array on initialization', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      expect(window.dataLayer).toBeDefined()
      expect(Array.isArray(window.dataLayer)).toBe(true)
    })

    it('adds gtag script to document head', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      expect(appendChildSpy).toHaveBeenCalled()
      const scriptArg = appendChildSpy.mock.calls[0]?.[0] as HTMLScriptElement
      expect(scriptArg.tagName).toBe('SCRIPT')
      expect(scriptArg.src).toContain('googletagmanager.com/gtag/js')
      expect(scriptArg.src).toContain('G-TEST123456')
    })

    it('configures GA with send_page_view disabled', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      // Find config call in dataLayer
      const configCall = window.dataLayer?.find(
        (entry) => Array.isArray(entry) && entry[0] === 'config'
      ) as unknown[] | undefined

      expect(configCall).toBeDefined()
      expect(configCall?.[1]).toBe('G-TEST123456')
      expect(configCall?.[2]).toEqual({ send_page_view: false })
    })

    it('does not re-initialize if called multiple times', async () => {
      const { rerender } = renderHook(() => useGoogleAnalytics(), { wrapper })

      const initialCallCount = appendChildSpy.mock.calls.length

      rerender()
      rerender()

      // Should still only have one script added
      expect(appendChildSpy.mock.calls.length).toBe(initialCallCount)
    })
  })

  describe('page view tracking', () => {
    it('tracks page view after route change with delay', async () => {
      vi.useFakeTimers()

      renderHook(() => useGoogleAnalytics(), { wrapper })

      // Fast-forward past the 100ms delay
      await act(async () => {
        vi.advanceTimersByTime(150)
      })

      // Check for page_view event in dataLayer
      const pageViewCall = window.dataLayer?.find(
        (entry) => Array.isArray(entry) && entry[0] === 'event' && entry[1] === 'page_view'
      ) as unknown[] | undefined

      expect(pageViewCall).toBeDefined()
      expect(pageViewCall?.[2]).toMatchObject({
        page_path: '/',
      })

      vi.useRealTimers()
    })

    it('includes page title in page view tracking', async () => {
      vi.useFakeTimers()
      document.title = 'Test Page Title'

      renderHook(() => useGoogleAnalytics(), { wrapper })

      await act(async () => {
        vi.advanceTimersByTime(150)
      })

      const pageViewCall = window.dataLayer?.find(
        (entry) => Array.isArray(entry) && entry[0] === 'event' && entry[1] === 'page_view'
      ) as unknown[] | undefined

      expect(pageViewCall?.[2]).toMatchObject({
        page_title: 'Test Page Title',
      })

      vi.useRealTimers()
    })
  })

  describe('return values', () => {
    it('returns trackEvent function', () => {
      const { result } = renderHook(() => useGoogleAnalytics(), { wrapper })

      expect(result.current.trackEvent).toBeDefined()
      expect(typeof result.current.trackEvent).toBe('function')
    })

    it('returns trackPageView function', () => {
      const { result } = renderHook(() => useGoogleAnalytics(), { wrapper })

      expect(result.current.trackPageView).toBeDefined()
      expect(typeof result.current.trackPageView).toBe('function')
    })
  })

  describe('trackPageView', () => {
    it('queues page view when GA is not yet ready', async () => {
      // Don't trigger onload - GA stays in "not ready" state
      appendChildSpy.mockImplementation((node: Node) => node)

      renderHook(() => useGoogleAnalytics(), { wrapper })

      await waitFor(() => {
        expect(window.gtag).toBeDefined()
      })

      trackPageView('/custom/path')

      // Event should be in dataLayer (queued via gtag function)
      const pageViewCalls = window.dataLayer?.filter(
        (entry) =>
          Array.isArray(entry) &&
          entry[0] === 'event' &&
          entry[1] === 'page_view' &&
          (entry[2] as Record<string, unknown>)?.page_path === '/custom/path'
      )

      expect(pageViewCalls?.length).toBeGreaterThan(0)
    })

    it('tracks page view with custom title', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      await waitFor(() => {
        expect(window.gtag).toBeDefined()
      })

      trackPageView('/path', 'Custom Title')

      const pageViewCall = window.dataLayer?.find(
        (entry) =>
          Array.isArray(entry) &&
          entry[0] === 'event' &&
          entry[1] === 'page_view' &&
          (entry[2] as Record<string, unknown>)?.page_title === 'Custom Title'
      ) as unknown[] | undefined

      expect(pageViewCall).toBeDefined()
    })
  })

  describe('trackEvent', () => {
    it('queues custom events with name and params', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      await waitFor(() => {
        expect(window.gtag).toBeDefined()
      })

      trackEvent('button_click', { button_id: 'test-btn', value: 42 })

      const eventCall = window.dataLayer?.find(
        (entry) =>
          Array.isArray(entry) && entry[0] === 'event' && entry[1] === 'button_click'
      ) as unknown[] | undefined

      expect(eventCall).toBeDefined()
      expect(eventCall?.[2]).toEqual({ button_id: 'test-btn', value: 42 })
    })

    it('queues events without params', async () => {
      renderHook(() => useGoogleAnalytics(), { wrapper })

      await waitFor(() => {
        expect(window.gtag).toBeDefined()
      })

      trackEvent('simple_event')

      const eventCall = window.dataLayer?.find(
        (entry) =>
          Array.isArray(entry) && entry[0] === 'event' && entry[1] === 'simple_event'
      ) as unknown[] | undefined

      expect(eventCall).toBeDefined()
    })
  })
})

describe('useGoogleAnalytics without GA_MEASUREMENT_ID', () => {
  let appendChildSpy: ReturnType<typeof vi.spyOn>

  beforeEach(async () => {
    // Reset window properties
    delete (window as { dataLayer?: unknown[] }).dataLayer
    delete (window as { gtag?: (...args: unknown[]) => void }).gtag

    appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => node)

    // Clear module cache
    vi.resetModules()

    // Set empty measurement ID
    vi.stubEnv('VITE_GA_MEASUREMENT_ID', '')
  })

  afterEach(() => {
    appendChildSpy.mockRestore()
    vi.unstubAllEnvs()
    vi.clearAllMocks()
  })

  it('does not initialize GA when measurement ID is missing', async () => {
    const module = await import('./useGoogleAnalytics')

    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter>{children}</MemoryRouter>
    )

    renderHook(() => module.useGoogleAnalytics(), { wrapper })

    // Script should not be added
    expect(appendChildSpy).not.toHaveBeenCalled()
  })

  it('trackPageView does nothing when measurement ID is missing', async () => {
    const module = await import('./useGoogleAnalytics')

    const initialDataLayerLength = window.dataLayer?.length ?? 0

    module.trackPageView('/test')

    // dataLayer should not have new entries (or not exist)
    expect(window.dataLayer?.length ?? 0).toBe(initialDataLayerLength)
  })

  it('trackEvent does nothing when measurement ID is missing', async () => {
    const module = await import('./useGoogleAnalytics')

    const initialDataLayerLength = window.dataLayer?.length ?? 0

    module.trackEvent('test_event', { param: 'value' })

    expect(window.dataLayer?.length ?? 0).toBe(initialDataLayerLength)
  })

  it('still returns tracking functions even when disabled', async () => {
    const module = await import('./useGoogleAnalytics')

    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter>{children}</MemoryRouter>
    )

    const { result } = renderHook(() => module.useGoogleAnalytics(), { wrapper })

    // Functions should still be returned (they just won't do anything)
    expect(typeof result.current.trackEvent).toBe('function')
    expect(typeof result.current.trackPageView).toBe('function')
  })
})
