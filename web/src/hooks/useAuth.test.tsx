import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { useAuth } from './useAuth'
import { AuthProvider } from '../contexts/AuthContext'

describe('useAuth', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('returns auth context when used within AuthProvider', async () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    // Wait for initial auth check to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Should have all expected properties and methods
    expect(result.current).toHaveProperty('user')
    expect(result.current).toHaveProperty('tokens')
    expect(result.current).toHaveProperty('isAuthenticated')
    expect(result.current).toHaveProperty('error')
    expect(result.current).toHaveProperty('loginWithGoogle')
    expect(result.current).toHaveProperty('logout')
    expect(result.current).toHaveProperty('clearError')

    // Functions should be callable
    expect(typeof result.current.loginWithGoogle).toBe('function')
    expect(typeof result.current.logout).toBe('function')
    expect(typeof result.current.clearError).toBe('function')
  })

  it('starts unauthenticated with no stored tokens', async () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
    expect(result.current.tokens).toBeNull()
  })
})
