import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { render } from '../test/test-utils'
import { Sidebar } from './Sidebar'
import type { User } from '../authTypes'

// Mock user for authenticated tests
const mockUser: User = {
  id: 'user-1',
  email: 'test@example.com',
  displayName: 'Test User',
  avatarUrl: 'https://example.com/avatar.jpg',
  status: 'active',
  emailVerified: true,
  isAdmin: false,
  isContentAdmin: false,
  isGearAdmin: false,
  createdAt: '2025-01-01T00:00:00Z',
}

// Helper to create default props
function createDefaultProps(overrides = {}) {
  return {
    activeSection: 'news' as const,
    onSectionChange: vi.fn(),
    isAuthenticated: false,
    user: null,
    authLoading: false,
    onSignIn: vi.fn(),
    onSignOut: vi.fn(),
    isMobileMenuOpen: false,
    onMobileMenuClose: vi.fn(),
    ...overrides,
  }
}

describe('Sidebar', () => {
  describe('Layout', () => {
    it('keeps sidebar vertically scrollable at tablet/desktop breakpoints', () => {
      const { container } = render(<Sidebar {...createDefaultProps()} />)

      const sidebar = container.querySelector('aside')
      expect(sidebar).toBeInTheDocument()
      expect(sidebar).toHaveClass('overflow-y-auto')
      expect(sidebar?.className).not.toContain('md:overflow-hidden')
    })
  })

  describe('Navigation', () => {
    it('renders navigation items for unauthenticated users', () => {
      render(<Sidebar {...createDefaultProps()} />)
      
      expect(screen.getByText('Home')).toBeInTheDocument()
      expect(screen.getByText('Taking Off')).toBeInTheDocument()
      expect(screen.getByText('News Feed')).toBeInTheDocument()
      expect(screen.getByText('Shop')).toBeInTheDocument()
      expect(screen.getByText('My Inventory')).toBeInTheDocument()
      expect(screen.getByText('My Aircraft')).toBeInTheDocument()
      expect(screen.getByText('My Radio')).toBeInTheDocument()
      expect(screen.getByText('My Batteries')).toBeInTheDocument()
    })

    it('renders Dashboard instead of Home/Taking Off for authenticated users', () => {
      render(<Sidebar {...createDefaultProps({ isAuthenticated: true, user: mockUser })} />)
      
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
      expect(screen.queryByText('Home')).not.toBeInTheDocument()
      expect(screen.queryByText('Taking Off')).not.toBeInTheDocument()
    })

    it('shows lock icons for auth-required sections when unauthenticated', () => {
      render(<Sidebar {...createDefaultProps()} />)
      
      // Auth-required sections should have a title indicating sign-in is needed
      const myGearButton = screen.getByText('My Inventory').closest('button')
      expect(myGearButton).toHaveAttribute('title', 'Sign in to access')
      
      const myAircraftButton = screen.getByText('My Aircraft').closest('button')
      expect(myAircraftButton).toHaveAttribute('title', 'Sign in to access')
    })

    it('does not show lock icons when authenticated', () => {
      render(<Sidebar {...createDefaultProps({ isAuthenticated: true, user: mockUser })} />)
      
      const myGearButton = screen.getByText('My Inventory').closest('button')
      expect(myGearButton).not.toHaveAttribute('title')
    })

    it('calls onSectionChange when navigation item is clicked', () => {
      const onSectionChange = vi.fn()
      render(<Sidebar {...createDefaultProps({ onSectionChange })} />)
      
      fireEvent.click(screen.getByText('News Feed'))
      expect(onSectionChange).toHaveBeenCalledWith('news')
      
      fireEvent.click(screen.getByText('Shop'))
      expect(onSectionChange).toHaveBeenCalledWith('equipment')
    })

    it('highlights the active section', () => {
      render(<Sidebar {...createDefaultProps({ activeSection: 'equipment' })} />)
      
      const shopButton = screen.getByText('Shop').closest('button')
      expect(shopButton).toHaveClass('bg-primary-600/20')
      expect(shopButton).toHaveClass('text-primary-400')
    })
  })

  describe('Inventory navigation', () => {
    it('does not render inventory filter controls inside the sidebar', () => {
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
      })} />)
      
      expect(screen.queryByText('Condition')).not.toBeInTheDocument()
      expect(screen.queryByText('Categories')).not.toBeInTheDocument()
      expect(screen.queryByText('All Categories')).not.toBeInTheDocument()
      expect(screen.queryByText('Total Items')).not.toBeInTheDocument()
      expect(screen.queryByText('Total Value')).not.toBeInTheDocument()
    })
  })

  describe('Authentication', () => {
    it('shows Sign In button when not authenticated', () => {
      render(<Sidebar {...createDefaultProps()} />)
      
      expect(screen.getByText('Sign In')).toBeInTheDocument()
    })

    it('calls onSignIn when Sign In button is clicked', () => {
      const onSignIn = vi.fn()
      render(<Sidebar {...createDefaultProps({ onSignIn })} />)
      
      fireEvent.click(screen.getByText('Sign In'))
      expect(onSignIn).toHaveBeenCalled()
    })

    it('shows user info when authenticated', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser 
      })} />)
      
      expect(screen.getByText('Test User')).toBeInTheDocument()
      expect(screen.getByText('test@example.com')).toBeInTheDocument()
    })

    it('shows user avatar when available', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser 
      })} />)
      
      const avatar = screen.getByAltText('Test User')
      expect(avatar).toHaveAttribute('src', 'https://example.com/avatar.jpg')
    })

    it('shows initial letter when no avatar', () => {
      const userWithoutAvatar = { ...mockUser, avatarUrl: undefined }
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: userWithoutAvatar 
      })} />)
      
      expect(screen.getByText('T')).toBeInTheDocument() // First letter of "Test User"
    })

    it('shows Sign Out button when authenticated', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser 
      })} />)
      
      expect(screen.getByText('Sign Out')).toBeInTheDocument()
    })

    it('calls onSignOut when Sign Out button is clicked', () => {
      const onSignOut = vi.fn()
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser,
        onSignOut 
      })} />)
      
      fireEvent.click(screen.getByText('Sign Out'))
      expect(onSignOut).toHaveBeenCalled()
    })

    it('shows loading spinner when auth is loading', () => {
      const { container } = render(<Sidebar {...createDefaultProps({ authLoading: true })} />)
      
      const spinner = container.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })
  })

  describe('Admin Navigation', () => {
    it('hides admin sections for regular users', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser // mockUser has isAdmin: false
      })} />)
      
      expect(screen.queryByText('Content Moderation')).not.toBeInTheDocument()
      expect(screen.queryByText('User Admin')).not.toBeInTheDocument()
    })

    it('hides admin sections when unauthenticated', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: false, 
        user: null
      })} />)
      
      expect(screen.queryByText('Content Moderation')).not.toBeInTheDocument()
      expect(screen.queryByText('User Admin')).not.toBeInTheDocument()
    })

    it('shows both admin sections for admin users', () => {
      const adminUser: User = { ...mockUser, isAdmin: true }
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: adminUser
      })} />)
      
      expect(screen.getByText('Content Moderation')).toBeInTheDocument()
      expect(screen.getByText('User Admin')).toBeInTheDocument()
    })

    it('shows only Content Moderation for content-admin users', () => {
      const contentAdminUser: User = { ...mockUser, isContentAdmin: true }
      render(<Sidebar {...createDefaultProps({
        isAuthenticated: true,
        user: contentAdminUser,
      })} />)

      expect(screen.getByText('Content Moderation')).toBeInTheDocument()
      expect(screen.queryByText('User Admin')).not.toBeInTheDocument()
    })

    it('navigates to admin-content section when Content Moderation is clicked', () => {
      const adminUser: User = { ...mockUser, isAdmin: true }
      const onSectionChange = vi.fn()
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: adminUser,
        onSectionChange
      })} />)
      
      fireEvent.click(screen.getByText('Content Moderation'))
      expect(onSectionChange).toHaveBeenCalledWith('admin-content')
    })

    it('navigates to admin-users section when User Admin is clicked', () => {
      const adminUser: User = { ...mockUser, isAdmin: true }
      const onSectionChange = vi.fn()
      render(<Sidebar {...createDefaultProps({
        isAuthenticated: true,
        user: adminUser,
        onSectionChange,
      })} />)

      fireEvent.click(screen.getByText('User Admin'))
      expect(onSectionChange).toHaveBeenCalledWith('admin-users')
    })
  })
})
