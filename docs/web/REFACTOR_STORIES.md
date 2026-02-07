# FlyingForge Frontend Refactor Stories

Architecture review completed February 6, 2026. Stories organized by priority.

---

## P0 — Critical

### Story 1: Split App.tsx into Feature Routes with Code Splitting

**Effort:** 3-4 days

**Problem:** App.tsx is 957 lines with ~20 useState hooks, all routing logic, and deep prop drilling. This hurts maintainability, bundle size, and developer experience.

**Acceptance Criteria:**
- [ ] App.tsx reduced to <200 lines (router shell only)
- [ ] Each major section lazy-loaded with `React.lazy()`
- [ ] Route-based code splitting verified in build output
- [ ] No functionality regression

**Prompt:**
```
Refactor /web/src/App.tsx to use route-based code splitting. 

1. Create a new /web/src/app/routes.tsx file that defines all routes using React Router
2. Use React.lazy() to lazy-load each major section:
   - NewsSection (FeedList + TopBar + ItemDetail)
   - DashboardSection  
   - InventorySection
   - AircraftSection
   - RadioSection
   - BatterySection
   - SocialSection
   - AdminGearModerationSection
   - ProfileSection
   - GearCatalogSection

3. Extract section-specific state into each feature component (move useState hooks out of App.tsx)
4. Keep only global state in App.tsx: auth, navigation, mobile menu
5. Add Suspense boundaries with loading fallbacks
6. Wrap routes in ErrorBoundary components

Current App.tsx location: /web/src/App.tsx (957 lines)
Preserve all existing functionality and routing behavior.
```

---

### Story 2: Add TanStack Query for Data Layer

**Effort:** 2-3 days

**Problem:** Manual useEffect data fetching with no caching, deduplication, or optimistic updates. Every navigation refetches data unnecessarily.

**Acceptance Criteria:**
- [ ] TanStack Query installed and configured
- [ ] All API calls migrated to useQuery/useMutation hooks
- [ ] Stale-while-revalidate caching working
- [ ] Loading/error states use query status
- [ ] Optimistic updates for mutations (add/edit/delete)

**Prompt:**
```
Add TanStack Query (React Query) to the FlyingForge frontend.

1. Install @tanstack/react-query and @tanstack/react-query-devtools
2. Create /web/src/lib/queryClient.ts with default configuration:
   - staleTime: 5 minutes for read data
   - gcTime: 30 minutes
   - retry: 1 for failed requests

3. Wrap App in QueryClientProvider in main.tsx

4. Create custom query hooks for each domain:
   - /web/src/features/news/useNewsQueries.ts (useNewsItems, useNewsSources)
   - /web/src/features/aircraft/useAircraftQueries.ts
   - /web/src/features/inventory/useInventoryQueries.ts
   - etc.

5. Migrate these existing API patterns:
   - App.tsx lines 158-194 (loadItems) → useQuery
   - App.tsx lines 268-298 (loadInventory) → useQuery
   - All manual loading/error state → query.isLoading, query.error

6. Add useMutation for create/update/delete operations with:
   - Optimistic updates using queryClient.setQueryData
   - Rollback on error
   - Cache invalidation on success

Keep existing API files (api.ts, equipmentApi.ts, etc.) as the fetch layer.
```

---

### Story 3: Add Modal Focus Trap for Accessibility

**Effort:** 1 day

**Problem:** Modals don't trap focus, allowing keyboard users to tab outside the modal. No focus restoration after close.

**Acceptance Criteria:**
- [ ] All modals trap focus within modal bounds
- [ ] Tab cycles through modal elements only
- [ ] Escape key closes modal
- [ ] Focus returns to trigger element on close
- [ ] `role="dialog"` and `aria-modal="true"` added

**Prompt:**
```
Add focus trap and proper ARIA to all modals in FlyingForge.

1. Install @headlessui/react or react-focus-lock

2. Update these modal components:
   - /web/src/components/LoginPage.tsx
   - /web/src/components/AddGearModal.tsx
   - /web/src/components/AddInventoryModal.tsx
   - /web/src/components/CatalogSearchModal.tsx
   - /web/src/components/ItemDetail.tsx (modal view)
   - /web/src/components/FollowListModal.tsx
   - /web/src/components/PublicAircraftModal.tsx

3. For each modal:
   - Wrap content in FocusTrap component
   - Add role="dialog" and aria-modal="true"
   - Add aria-labelledby pointing to modal title
   - Store trigger element ref before open
   - Restore focus to trigger on close
   - Ensure Escape key closes modal

4. Test with keyboard-only navigation
```

---

## P1 — High Priority

### Story 4: Extract Shared UI Components

**Effort:** 1 day

**Problem:** Loading skeletons, error states, and empty states are duplicated across Dashboard.tsx, FeedList.tsx, AircraftList.tsx, and others.

**Acceptance Criteria:**
- [ ] Shared LoadingSpinner component
- [ ] Shared SkeletonCard component with variants
- [ ] Shared ErrorState component
- [ ] Shared EmptyState component
- [ ] All duplications replaced with shared components

**Prompt:**
```
Extract shared UI components from duplicated code in FlyingForge.

1. Create /web/src/shared/components/ directory

2. Create LoadingSpinner.tsx:
   - Reusable spinner with size prop (sm, md, lg)
   - Optional text prop for "Loading..." message
   - Extract from existing spinner patterns

3. Create SkeletonCard.tsx:
   - Props: variant ('feed' | 'aircraft' | 'inventory' | 'equipment')
   - Extract from Dashboard.tsx lines 28-46
   - Extract from FeedList.tsx lines 37-55

4. Create ErrorState.tsx:
   - Props: title, message, onRetry (optional)
   - Red icon, centered layout
   - Extract from FeedList.tsx lines 22-35

5. Create EmptyState.tsx:
   - Props: icon, title, message  
   - Neutral icon, centered layout
   - Extract from FeedList.tsx lines 60-77

6. Replace all existing duplicated patterns with these shared components
7. Export from /web/src/shared/components/index.ts
```

---

### Story 5: Add aria-live Regions for Dynamic Content

**Effort:** 0.5 days

**Problem:** Screen readers don't announce loading states, errors, or success messages.

**Acceptance Criteria:**
- [ ] Loading states announced with aria-live="polite"
- [ ] Error messages announced with role="alert"
- [ ] Success toasts/messages announced
- [ ] Page title updates reflect current section

**Prompt:**
```
Add aria-live regions for screen reader announcements in FlyingForge.

1. Create /web/src/shared/components/Announcer.tsx:
   - Visually hidden region with aria-live="polite"
   - Context for announcing messages from anywhere
   - Auto-clear after announcement

2. Add role="alert" to error messages:
   - AdminGearModeration.tsx error display
   - FeedList.tsx error state
   - Form validation errors

3. Add aria-live="polite" to:
   - Loading state changes ("Loading items...", "12 items loaded")
   - Filter result counts ("4 items found")
   - Success messages ("Item saved")

4. Update document.title on navigation:
   - "News | FlyingForge"
   - "Dashboard | FlyingForge"
   - etc.

5. Test with VoiceOver (macOS) or NVDA
```

---

### Story 6: Virtualize Long Lists

**Effort:** 1-2 days

**Problem:** FeedList, InventoryList render all items. With 100+ items, this causes scroll jank.

**Acceptance Criteria:**
- [ ] FeedList uses virtualization for >50 items
- [ ] InventoryList uses virtualization
- [ ] Smooth scrolling maintained
- [ ] Infinite scroll still works
- [ ] Search/filter performance improved

**Prompt:**
```
Add list virtualization to FlyingForge using react-window.

1. Install react-window and @types/react-window

2. Update /web/src/components/FeedList.tsx:
   - Import VariableSizeList from react-window
   - Measure item heights (cards vary in height)
   - Use VariableSizeList when items.length > 50
   - Keep simple list for small datasets
   - Preserve infinite scroll trigger at bottom

3. Update /web/src/components/InventoryList.tsx:
   - Similar virtualization pattern
   - Account for grid layout on desktop (may need react-window grid)

4. Create /web/src/shared/hooks/useVirtualizedList.ts:
   - Reusable hook for virtualization setup
   - Handle resize events
   - Memoize row renderers

5. Test scroll performance with 200+ items
6. Ensure keyboard navigation still works
```

---

### Story 7: Reorganize to Feature-Based Folder Structure

**Effort:** 1-2 days

**Problem:** 37 flat component files, 16 root-level API/type files. Hard to find related code.

**Acceptance Criteria:**
- [ ] Features grouped in /features folder
- [ ] Shared code in /shared folder
- [ ] All imports updated
- [ ] No broken references
- [ ] Barrel exports maintained

**Prompt:**
```
Reorganize FlyingForge frontend to feature-based folder structure.

Target structure:
/web/src/
├── app/
│   ├── App.tsx
│   ├── routes.tsx
│   └── main.tsx
├── features/
│   ├── auth/
│   │   ├── components/
│   │   │   ├── LoginPage.tsx
│   │   │   └── AuthCallback.tsx
│   │   ├── AuthContext.tsx
│   │   ├── authApi.ts
│   │   ├── authTypes.ts
│   │   └── index.ts
│   ├── news/
│   │   ├── components/
│   │   │   ├── FeedList.tsx
│   │   │   ├── FeedCard.tsx
│   │   │   ├── TopBar.tsx
│   │   │   └── ItemDetail.tsx
│   │   ├── api.ts
│   │   ├── types.ts
│   │   └── index.ts
│   ├── aircraft/
│   ├── inventory/
│   ├── equipment/
│   ├── radio/
│   ├── battery/
│   ├── social/
│   ├── admin/
│   └── profile/
├── shared/
│   ├── components/
│   │   ├── LoadingSpinner.tsx
│   │   ├── ErrorState.tsx
│   │   ├── EmptyState.tsx
│   │   └── index.ts
│   ├── hooks/
│   │   ├── useDebounce.ts
│   │   ├── useInfiniteScroll.ts
│   │   └── index.ts
│   └── types/
└── test/

1. Create folder structure
2. Move files to appropriate locations
3. Update all imports (use IDE refactoring or search/replace)
4. Create barrel exports (index.ts) for each feature
5. Update vite config if needed for path aliases
6. Verify build succeeds with no broken imports
```

---

## P2 — Medium Priority

### Story 8: Expand React.memo Usage

**Effort:** 0.5 days

**Problem:** Only Sidebar uses React.memo. Heavy components re-render unnecessarily.

**Acceptance Criteria:**
- [ ] Dashboard wrapped in memo
- [ ] FeedList, FeedCard wrapped in memo
- [ ] InventoryList, InventoryCard wrapped in memo
- [ ] AircraftList, AircraftCard wrapped in memo
- [ ] Custom comparison functions where needed

**Prompt:**
```
Add React.memo to frequently re-rendered components in FlyingForge.

1. Wrap these components with React.memo():
   - Dashboard.tsx
   - FeedList.tsx
   - FeedCard.tsx
   - InventoryList.tsx
   - InventoryCard.tsx
   - EquipmentCard.tsx
   - AircraftList.tsx
   - AircraftCard.tsx

2. For card components, add custom comparison:
   ```tsx
   export const FeedCard = memo(function FeedCard(props) {
     // ...
   }, (prevProps, nextProps) => {
     return prevProps.item.id === nextProps.item.id 
       && prevProps.item.updatedAt === nextProps.item.updatedAt;
   });
   ```

3. Ensure callback props are stable (wrapped in useCallback at parent)

4. Use React DevTools Profiler to verify fewer re-renders
```

---

### Story 9: Add ARIA Roles for Interactive Widgets

**Effort:** 1 day

**Problem:** Tabs, navigation lack proper ARIA roles. Screen readers can't identify widget types.

**Acceptance Criteria:**
- [ ] Tab components have role="tablist", role="tab", aria-selected
- [ ] Navigation has proper landmarks
- [ ] Form groups have role="group" and aria-labelledby
- [ ] Dropdowns have aria-expanded, aria-haspopup

**Prompt:**
```
Add comprehensive ARIA roles to FlyingForge interactive widgets.

1. Tab patterns (SocialPage.tsx, Dashboard.tsx if applicable):
   - Container: role="tablist"
   - Each tab: role="tab", aria-selected, aria-controls
   - Tab panels: role="tabpanel", aria-labelledby

2. Navigation (Sidebar.tsx):
   - Main nav: <nav aria-label="Main navigation">
   - Current page: aria-current="page"

3. Dropdowns/Selects (TopBar.tsx, AdminGearModeration.tsx):
   - Trigger: aria-haspopup="listbox", aria-expanded
   - Options: role="listbox", role="option"

4. Form groups:
   - Related inputs: role="group", aria-labelledby
   - Required fields: aria-required="true"

5. Cards that are clickable:
   - role="button" or wrap in button element
   - Keyboard accessible (Enter/Space trigger)

6. Test with axe DevTools or Lighthouse accessibility audit
```

---

### Story 10: Add Keyboard Navigation for Lists

**Effort:** 1-2 days

**Problem:** Lists only support mouse interaction. Power users can't navigate with arrow keys.

**Acceptance Criteria:**
- [ ] Arrow up/down moves focus between list items
- [ ] Enter activates focused item
- [ ] Home/End jump to first/last item
- [ ] Focus visible indicator on items
- [ ] Works with virtualized lists

**Prompt:**
```
Add keyboard navigation to list components in FlyingForge.

1. Create /web/src/shared/hooks/useListKeyboardNav.ts:
   ```tsx
   function useListKeyboardNav(options: {
     itemCount: number;
     onSelect: (index: number) => void;
     orientation?: 'vertical' | 'horizontal';
   })
   ```
   - Track focused index
   - Handle ArrowUp/Down, Home, End, Enter
   - Return handlers and focusedIndex

2. Apply to FeedList.tsx:
   - Make items focusable (tabIndex)
   - Apply keyboard hook
   - Show focus indicator ring

3. Apply to InventoryList.tsx, AircraftList.tsx

4. For grid layouts (AircraftList):
   - Also handle ArrowLeft/Right
   - Calculate row/column navigation

5. Handle edge cases:
   - Empty list
   - Single item
   - Virtualized list (scroll focused item into view)

6. Roving tabindex pattern for efficiency
```

---

### Story 11: Add Error Boundaries per Feature

**Effort:** 0.5 days

**Problem:** JavaScript errors crash entire app. No graceful degradation.

**Acceptance Criteria:**
- [ ] ErrorBoundary component created
- [ ] Each feature section wrapped in boundary
- [ ] User-friendly error UI with retry option
- [ ] Errors logged to console (or error service)

**Prompt:**
```
Add Error Boundaries to isolate failures in FlyingForge.

1. Create /web/src/shared/components/ErrorBoundary.tsx:
   ```tsx
   interface Props {
     fallback?: ReactNode;
     onError?: (error: Error, errorInfo: ErrorInfo) => void;
     children: ReactNode;
   }
   ```
   - Class component (required for componentDidCatch)
   - Default fallback with error message and retry button
   - Reset state on retry

2. Create feature-specific boundaries:
   - NewsErrorBoundary
   - InventoryErrorBoundary
   - etc. (or use same component with different fallbacks)

3. Wrap each lazy-loaded route in ErrorBoundary:
   ```tsx
   <ErrorBoundary fallback={<NewsError />}>
     <Suspense fallback={<Loading />}>
       <NewsSection />
     </Suspense>
   </ErrorBoundary>
   ```

4. Log errors (console.error for now, prep for Sentry later)

5. Test by throwing error in a component
```

---

### Story 12: Consolidate Type Files

**Effort:** 0.5 days

**Problem:** 16 type files at root level (authTypes.ts, equipmentTypes.ts, etc.). Should live with their features.

**Acceptance Criteria:**
- [ ] Types moved to feature folders
- [ ] Shared types in /shared/types
- [ ] All imports updated
- [ ] No duplicate type definitions

**Prompt:**
```
Consolidate type definition files into feature folders.

Current root-level type files:
- types.ts → /features/news/types.ts
- authTypes.ts → /features/auth/types.ts
- equipmentTypes.ts → /features/equipment/types.ts
- aircraftTypes.ts → /features/aircraft/types.ts
- radioTypes.ts → /features/radio/types.ts
- batteryTypes.ts → /features/battery/types.ts
- socialTypes.ts → /features/social/types.ts
- fcConfigTypes.ts → /features/aircraft/fcConfigTypes.ts
- gearCatalogTypes.ts → /features/equipment/gearCatalogTypes.ts

1. Move each type file to its feature folder
2. Create /shared/types/index.ts for truly shared types:
   - Pagination types
   - API response wrappers
   - Common utility types

3. Update imports across all files
4. Re-export from feature index.ts for convenience
5. Remove empty files from root
```

---

## Summary

| Priority | Stories | Total Effort |
|----------|---------|--------------|
| P0 | 3 | 6-8 days |
| P1 | 4 | 3.5-5.5 days |
| P2 | 5 | 3.5-5 days |
| **Total** | **12** | **13-18.5 days** |

Recommended order: Stories 1 → 2 → 3 → 7 → 4 → 5 → 6 → 8 → 9 → 10 → 11 → 12
