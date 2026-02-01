# Web Frontend Architecture

This document provides a comprehensive overview of the RotorLife React frontend architecture, including component structure, state management, and data flow.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Project Structure](#project-structure)
4. [State Management](#state-management)
5. [Component Architecture](#component-architecture)
6. [API Layer](#api-layer)
7. [Authentication](#authentication)
8. [Routing & Navigation](#routing--navigation)
9. [Styling](#styling)
10. [Type System](#type-system)

---

## Overview

The RotorLife frontend is a React single-page application built with:

- **React 18** - UI library with functional components and hooks
- **TypeScript** - Strict type checking
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Utility-first styling
- **Context API** - Global state (auth only)
- **Local component state** - UI and data state

### Design Principles

1. **No external state management** - React Context + local state is sufficient
2. **Colocation** - Keep state close to where it's used
3. **Type safety** - All data structures have TypeScript interfaces
4. **Functional components** - No class components
5. **Custom hooks** - Extract reusable logic

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           RotorLife Web Frontend                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                              App.tsx                                 │   │
│  │                         (Root Component)                             │   │
│  │                                                                      │   │
│  │  ┌─────────────┐  ┌──────────────────────────────────────────────┐  │   │
│  │  │ AuthContext │  │              Section State                    │  │   │
│  │  │  Provider   │  │  activeSection, items, aircraft, inventory   │  │   │
│  │  └─────────────┘  └──────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│         │                              │                                    │
│         ▼                              ▼                                    │
│  ┌─────────────┐    ┌─────────────────────────────────────────────────┐   │
│  │  Sidebar    │    │                Main Content                      │   │
│  │             │    │                                                  │   │
│  │ • Dashboard │    │  ┌───────────┐ ┌──────────┐ ┌──────────────┐   │   │
│  │ • News Feed │    │  │ Dashboard │ │ FeedList │ │ AircraftList │   │   │
│  │ • Shop      │    │  └───────────┘ └──────────┘ └──────────────┘   │   │
│  │ • My Gear   │    │                                                  │   │
│  │ • Aircraft  │    │  ┌───────────┐ ┌──────────┐ ┌──────────────┐   │   │
│  └─────────────┘    │  │ ShopSection│ │Inventory │ │   Modals     │   │   │
│                     │  └───────────┘ └──────────┘ └──────────────┘   │   │
│                     └─────────────────────────────────────────────────┘   │
│                                        │                                    │
│  ┌─────────────────────────────────────┴───────────────────────────────┐   │
│  │                           API Layer                                  │   │
│  │                                                                      │   │
│  │  ┌─────────┐  ┌──────────────┐  ┌─────────────┐  ┌────────────┐   │   │
│  │  │ api.ts  │  │ equipmentApi │  │ aircraftApi │  │  authApi   │   │   │
│  │  │ (feeds) │  │   .ts        │  │    .ts      │  │    .ts     │   │   │
│  │  └─────────┘  └──────────────┘  └─────────────┘  └────────────┘   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                        │                                    │
└────────────────────────────────────────┼────────────────────────────────────┘
                                         │
                                         ▼
                          ┌──────────────────────────┐
                          │     Go Backend API       │
                          │     (port 8080)          │
                          └──────────────────────────┘
```

---

## Project Structure

```
web/src/
├── main.tsx                 # Entry point, renders App
├── App.tsx                  # Root component, routing, state
├── index.css                # Global styles, Tailwind imports
│
├── api.ts                   # News feed API calls
├── authApi.ts               # Authentication API calls
├── equipmentApi.ts          # Equipment/inventory API calls
├── aircraftApi.ts           # Aircraft API calls
│
├── types.ts                 # News feed types
├── authTypes.ts             # Auth types
├── equipmentTypes.ts        # Equipment/inventory types
├── aircraftTypes.ts         # Aircraft types
│
├── hooks.ts                 # Shared custom hooks
├── hooks/
│   └── useAuth.ts           # Auth hook (uses AuthContext)
│
├── contexts/
│   └── AuthContext.tsx      # Auth state provider
│
└── components/
    ├── index.ts             # Component exports
    │
    ├── Dashboard.tsx        # Logged-in homepage
    ├── Sidebar.tsx          # News source filters
    ├── EquipmentSidebar.tsx # Main navigation sidebar
    ├── TopBar.tsx           # Search and filter controls
    │
    ├── FeedCard.tsx         # News item card
    ├── FeedList.tsx         # News feed list
    ├── ItemDetail.tsx       # News item modal
    │
    ├── ShopSection.tsx      # Equipment shop
    ├── EquipmentCard.tsx    # Equipment item card
    │
    ├── InventoryCard.tsx    # Inventory item card
    ├── AddInventoryModal.tsx# Add/edit inventory modal
    │
    ├── AircraftCard.tsx     # Aircraft card
    ├── AircraftList.tsx     # Aircraft list
    ├── AircraftForm.tsx     # Add/edit aircraft modal
    ├── AircraftDetail.tsx   # Aircraft detail modal
    │
    ├── LoginPage.tsx        # Login modal
    ├── SignupPage.tsx       # Signup modal
    └── AuthCallback.tsx     # OAuth callback handler
```

---

## State Management

### Philosophy

We use a **minimal state management** approach:

| State Type | Solution | Example |
|------------|----------|---------|
| Auth state | React Context | User, tokens, isAuthenticated |
| Server data | Local state + useEffect | Feed items, aircraft, inventory |
| UI state | Local useState | Modals, active section, filters |
| Persisted | localStorage | Filter preferences |

### Auth Context

The only global state is authentication:

```typescript
// contexts/AuthContext.tsx
interface AuthContextType extends AuthState {
  signup: (params: SignupParams) => Promise<void>;
  login: (params: LoginParams) => Promise<void>;
  loginWithGoogle: (params: GoogleLoginParams) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}
```

### App-Level State

All other state lives in `App.tsx`:

```typescript
// Section navigation
const [activeSection, setActiveSection] = useState<AppSection>('news');

// News feed
const [items, setItems] = useState<FeedItem[]>([]);
const [sources, setSources] = useState<SourceInfo[]>([]);

// Inventory
const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
const [inventorySummary, setInventorySummary] = useState<InventorySummary | null>(null);

// Aircraft
const [aircraftItems, setAircraftItems] = useState<Aircraft[]>([]);
```

### Custom Hooks

Reusable logic is extracted into hooks:

```typescript
// hooks.ts
export function useFilters()     // Filter state with localStorage persistence
export function useDebounce()    // Debounced values for search

// hooks/useAuth.ts
export function useAuth()        // Access auth context
```

---

## Component Architecture

### Component Types

| Type | Purpose | Example |
|------|---------|---------|
| **Page** | Full section content | Dashboard, ShopSection |
| **List** | Renders collections | FeedList, AircraftList |
| **Card** | Single item display | FeedCard, AircraftCard |
| **Modal** | Overlay dialogs | ItemDetail, AircraftForm |
| **Layout** | Structure | EquipmentSidebar, TopBar |
| **Form** | User input | LoginPage, AddInventoryModal |

### Component Patterns

**Props-driven components:**
```typescript
interface FeedListProps {
  items: FeedItem[];
  sources: SourceInfo[];
  isLoading: boolean;
  error: string | null;
  onItemClick: (item: FeedItem) => void;
}
```

**Callback props for actions:**
```typescript
interface AircraftListProps {
  aircraft: Aircraft[];
  onSelect: (aircraft: Aircraft) => void;
  onEdit: (aircraft: Aircraft) => void;
  onDelete: (aircraft: Aircraft) => void;
}
```

**Loading and error states:**
```typescript
if (isLoading) return <SkeletonLoader />;
if (error) return <ErrorDisplay message={error} />;
if (items.length === 0) return <EmptyState />;
return <ItemList items={items} />;
```

---

## API Layer

### Structure

Each domain has its own API module:

| Module | Endpoints | Auth Required |
|--------|-----------|---------------|
| `api.ts` | `/api/items`, `/api/sources`, `/api/refresh` | No |
| `authApi.ts` | `/api/auth/*` | No |
| `equipmentApi.ts` | `/api/equipment/*`, `/api/inventory/*` | Partial |
| `aircraftApi.ts` | `/api/aircraft/*` | Yes |

### API Pattern

```typescript
// Common pattern for all API calls
const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options?.headers,
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || `HTTP ${response.status}`);
  }

  return response.json();
}
```

### Token Management

```typescript
// Tokens stored in localStorage
function getAccessToken(): string | null {
  return localStorage.getItem('access_token');
}

function storeTokens(tokens: AuthTokens): void {
  localStorage.setItem('access_token', tokens.accessToken);
  localStorage.setItem('refresh_token', tokens.refreshToken);
}

function clearStoredTokens(): void {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
}
```

---

## Authentication

### Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  App Load   │────▶│ Check tokens │────▶│ Validate    │
└─────────────┘     └──────────────┘     │ with server │
                                         └──────┬──────┘
                                                │
                    ┌───────────────────────────┴───────────────────────┐
                    │                                                   │
                    ▼                                                   ▼
            ┌───────────────┐                               ┌───────────────┐
            │  Valid token  │                               │ Invalid/None  │
            │               │                               │               │
            │ AUTH_SUCCESS  │                               │  AUTH_LOGOUT  │
            └───────────────┘                               └───────────────┘
```

### Auth State Transitions

```typescript
type AuthAction =
  | { type: 'AUTH_START' }           // Login/signup initiated
  | { type: 'AUTH_SUCCESS'; ... }    // Successfully authenticated
  | { type: 'AUTH_ERROR'; ... }      // Auth failed
  | { type: 'AUTH_LOGOUT' }          // User logged out
  | { type: 'REFRESH_TOKENS'; ... }  // Tokens refreshed
  | { type: 'CLEAR_ERROR' };         // Clear error state
```

### Protected Sections

```typescript
const handleSectionChange = useCallback((section: AppSection) => {
  // Dashboard, inventory, and aircraft require authentication
  if ((section === 'dashboard' || section === 'inventory' || section === 'aircraft') 
      && !isAuthenticated) {
    setAuthModal('login');
    return;
  }
  setActiveSection(section);
}, [isAuthenticated]);
```

---

## Routing & Navigation

### Section-Based Navigation

Instead of URL routing, we use section state:

```typescript
type AppSection = 'dashboard' | 'news' | 'equipment' | 'inventory' | 'aircraft';

// Sidebar triggers section changes
<EquipmentSidebar
  activeSection={activeSection}
  onSectionChange={handleSectionChange}
  // ...
/>

// Main content renders based on active section
{activeSection === 'dashboard' && <Dashboard ... />}
{activeSection === 'news' && <FeedList ... />}
{activeSection === 'equipment' && <ShopSection />}
{activeSection === 'inventory' && <InventoryList ... />}
{activeSection === 'aircraft' && <AircraftList ... />}
```

### Auth-Based Homepage

```typescript
// On auth state change, set appropriate homepage
useEffect(() => {
  if (authLoading) return;
  
  // Detect logout: redirect to news
  if (wasAuthenticated && !isAuthenticated) {
    setActiveSection('news');
  }
  
  // Detect login: redirect to dashboard
  if (!wasAuthenticated && isAuthenticated) {
    setActiveSection('dashboard');
  }
}, [isAuthenticated, authLoading]);
```

---

## Styling

### Tailwind CSS

All styling uses Tailwind utility classes:

```tsx
<button
  className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors ${
    isActive
      ? 'bg-primary-600/20 text-primary-400'
      : 'text-slate-400 hover:text-white hover:bg-slate-800'
  }`}
>
```

### Design Tokens

Custom colors defined in `tailwind.config.js`:

| Token | Usage |
|-------|-------|
| `primary-*` | Brand color, CTAs, active states |
| `slate-*` | Backgrounds, borders, text |
| `green-*` | Success states |
| `red-*` | Error states |
| `yellow-*` | Warning states |

### Dark Theme

The app uses a dark theme exclusively:

- Background: `bg-slate-900`
- Cards: `bg-slate-800`
- Borders: `border-slate-700`
- Text: `text-white`, `text-slate-400`

---

## Type System

### Type Files

| File | Contents |
|------|----------|
| `types.ts` | Feed items, sources, filters |
| `authTypes.ts` | User, tokens, auth state |
| `equipmentTypes.ts` | Equipment, inventory, categories |
| `aircraftTypes.ts` | Aircraft, components, ELRS |

### Key Types

```typescript
// News Feed
interface FeedItem {
  id: string;
  title: string;
  url: string;
  source: string;
  sourceType: SourceType;
  publishedAt?: string;
  // ...
}

// User
interface User {
  id: string;
  email: string;
  displayName: string;
  avatarUrl?: string;
  // ...
}

// Aircraft
interface Aircraft {
  id: string;
  userId: string;
  name: string;
  type: AircraftType;
  // ...
}

// Inventory
interface InventoryItem {
  id: string;
  name: string;
  category: EquipmentCategory;
  condition: ItemCondition;
  quantity: number;
  // ...
}
```

### Strict Mode

TypeScript strict mode is enabled:

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true
  }
}
```

---

## Key Patterns

### Loading States

```typescript
// Skeleton loaders during fetch
if (isLoading && items.length === 0) {
  return (
    <div className="space-y-4">
      {[...Array(6)].map((_, i) => (
        <div key={i} className="animate-pulse bg-slate-800 rounded-xl h-24" />
      ))}
    </div>
  );
}
```

### Empty States

```typescript
if (items.length === 0) {
  return (
    <EmptyState
      icon={<Icon />}
      title="No Items Yet"
      description="Add your first item to get started"
      actionLabel="Add Item"
      onAction={onAdd}
    />
  );
}
```

### Optimistic Updates

```typescript
// Update local state immediately, revert on error
const handleDelete = async (item: InventoryItem) => {
  const previousItems = inventoryItems;
  setInventoryItems(prev => prev.filter(i => i.id !== item.id));
  
  try {
    await deleteInventoryItem(item.id);
  } catch (err) {
    setInventoryItems(previousItems);
    // Show error toast
  }
};
```

---

## Development

### Dev Server

```bash
cd web
npm install
npm run dev
# Opens http://localhost:5173
```

### Build

```bash
npm run build
# Output in dist/
```

### Type Check

```bash
npx tsc --noEmit
```

### Lint

```bash
npm run lint
```

---

*Last updated: February 2026*
