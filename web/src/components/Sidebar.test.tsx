import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { render } from '../test/test-utils'
import { Sidebar } from './Sidebar'
import type { EquipmentSearchParams, InventorySummary, EquipmentCategory, ItemCondition } from '../equipmentTypes'
import type { User } from '../authTypes'

// Mock user for authenticated tests
const mockUser: User = {
  id: 'user-1',
  email: 'test@example.com',
  displayName: 'Test User',
  avatarUrl: 'https://example.com/avatar.jpg',
  status: 'active',
  emailVerified: true,
  createdAt: '2025-01-01T00:00:00Z',
}

// Default search params
const defaultSearchParams: EquipmentSearchParams = {
  query: '',
  category: undefined,
  minPrice: undefined,
  maxPrice: undefined,
  inStockOnly: false,
  seller: undefined,
}

// Default inventory summary - must include all categories and conditions
const mockInventorySummary: InventorySummary = {
  totalItems: 15,
  totalValue: 2500,
  byCategory: {
    frames: 3,
    motors: 8,
    propellers: 4,
    vtx: 0,
    flight_controllers: 0,
    esc: 0,
    stacks: 0,
    receivers: 0,
    cameras: 0,
    antennas: 0,
    accessories: 0,
  } as Record<EquipmentCategory, number>,
  byCondition: {
    new: 5,
    used: 10,
    broken: 0,
    spare: 0,
  } as Record<ItemCondition, number>,
}

// Helper to create default props
function createDefaultProps(overrides = {}) {
  return {
    activeSection: 'news' as const,
    onSectionChange: vi.fn(),
    searchParams: defaultSearchParams,
    onSearchChange: vi.fn(),
    sellers: [],
    inventorySummary: null,
    inventoryCategory: null,
    inventoryCondition: null,
    onInventoryFilterChange: vi.fn(),
    isAuthenticated: false,
    user: null,
    authLoading: false,
    onSignIn: vi.fn(),
    onSignOut: vi.fn(),
    ...overrides,
  }
}

describe('Sidebar', () => {
  describe('Navigation', () => {
    it('renders navigation items for unauthenticated users', () => {
      render(<Sidebar {...createDefaultProps()} />)
      
      expect(screen.getByText('Home')).toBeInTheDocument()
      expect(screen.getByText('Getting Started')).toBeInTheDocument()
      expect(screen.getByText('News Feed')).toBeInTheDocument()
      expect(screen.getByText('Shop')).toBeInTheDocument()
      expect(screen.getByText('My Gear')).toBeInTheDocument()
      expect(screen.getByText('My Aircraft')).toBeInTheDocument()
      expect(screen.getByText('My Radio')).toBeInTheDocument()
      expect(screen.getByText('Batteries')).toBeInTheDocument()
    })

    it('renders Dashboard instead of Home/Getting Started for authenticated users', () => {
      render(<Sidebar {...createDefaultProps({ isAuthenticated: true, user: mockUser })} />)
      
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
      expect(screen.queryByText('Home')).not.toBeInTheDocument()
      expect(screen.queryByText('Getting Started')).not.toBeInTheDocument()
    })

    it('shows lock icons for auth-required sections when unauthenticated', () => {
      render(<Sidebar {...createDefaultProps()} />)
      
      // Auth-required sections should have a title indicating sign-in is needed
      const myGearButton = screen.getByText('My Gear').closest('button')
      expect(myGearButton).toHaveAttribute('title', 'Sign in to access')
      
      const myAircraftButton = screen.getByText('My Aircraft').closest('button')
      expect(myAircraftButton).toHaveAttribute('title', 'Sign in to access')
    })

    it('does not show lock icons when authenticated', () => {
      render(<Sidebar {...createDefaultProps({ isAuthenticated: true, user: mockUser })} />)
      
      const myGearButton = screen.getByText('My Gear').closest('button')
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

    it('shows item count badge on My Gear when inventory summary exists', () => {
      render(<Sidebar {...createDefaultProps({ 
        isAuthenticated: true, 
        user: mockUser,
        inventorySummary: mockInventorySummary 
      })} />)
      
      // Should show the total items count
      expect(screen.getByText('15')).toBeInTheDocument()
    })
  })

  describe('Inventory Filters', () => {
    it('shows inventory filters only when inventory section is active', () => {
      const { rerender } = render(<Sidebar {...createDefaultProps({ activeSection: 'news' })} />)
      
      expect(screen.queryByText('Condition')).not.toBeInTheDocument()
      expect(screen.queryByText('Categories')).not.toBeInTheDocument()
      
      rerender(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser 
      })} />)
      
      expect(screen.getByText('Condition')).toBeInTheDocument()
      expect(screen.getByText('Categories')).toBeInTheDocument()
    })

    it('shows inventory summary when available', () => {
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
        inventorySummary: mockInventorySummary
      })} />)
      
      expect(screen.getByText('Total Items')).toBeInTheDocument()
      expect(screen.getByText('Total Value')).toBeInTheDocument()
      expect(screen.getByText('$2500')).toBeInTheDocument()
      // 15 appears twice (badge + summary), just verify the summary section exists
      expect(screen.getAllByText('15')).toHaveLength(2)
    })

    it('calls onInventoryFilterChange when condition is changed', () => {
      const onInventoryFilterChange = vi.fn()
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
        onInventoryFilterChange
      })} />)
      
      const conditionSelect = screen.getByRole('combobox')
      fireEvent.change(conditionSelect, { target: { value: 'new' } })
      
      expect(onInventoryFilterChange).toHaveBeenCalledWith(null, 'new')
    })

    it('calls onInventoryFilterChange when category is selected', () => {
      const onInventoryFilterChange = vi.fn()
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
        onInventoryFilterChange
      })} />)
      
      fireEvent.click(screen.getByText('Motors'))
      
      expect(onInventoryFilterChange).toHaveBeenCalledWith('motors', null)
    })

    it('highlights the selected category', () => {
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
        inventoryCategory: 'motors'
      })} />)
      
      const motorsButton = screen.getByText('Motors').closest('button')
      expect(motorsButton).toHaveClass('bg-slate-800')
      expect(motorsButton).toHaveClass('text-white')
    })

    it('shows All Categories option and highlights when no category selected', () => {
      render(<Sidebar {...createDefaultProps({ 
        activeSection: 'inventory',
        isAuthenticated: true,
        user: mockUser,
        inventoryCategory: null
      })} />)
      
      const allCategoriesButton = screen.getByText('All Categories').closest('button')
      expect(allCategoriesButton).toHaveClass('bg-slate-800')
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
})
