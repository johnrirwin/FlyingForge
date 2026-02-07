# FlyingForge Next.js Migration Plan

Migration assessment completed February 6, 2026.

**Total Estimated Effort:** 6-8 weeks (1-2 developers)

---

## Architecture Overview

### Before (Current)
```
┌─────────────────┐     ┌─────────────────┐
│   Vite + React  │────▶│   Go Backend    │
│   (SPA)         │     │   (API Server)  │
│   Port 5173     │     │   Port 8080     │
└─────────────────┘     └─────────────────┘
```

### After (Next.js)
```
┌─────────────────┐     ┌─────────────────┐
│   Next.js       │────▶│   Go Backend    │
│   (SSR/SSG/CSR) │     │   (API Server)  │
│   Port 3000     │     │   Port 8080     │
└─────────────────┘     └─────────────────┘
```

**Note:** Go backend remains unchanged. Next.js only replaces the frontend build/routing layer.

---

## Phase 1: Project Setup & Configuration

### Story 1.1: Initialize Next.js Project

**Effort:** 2-3 days

**Acceptance Criteria:**
- [ ] Next.js 14+ project initialized with App Router
- [ ] TypeScript configured with strict mode
- [ ] Tailwind CSS configured identically to current
- [ ] ESLint configured
- [ ] Project builds successfully

**Prompt:**
```
Initialize a new Next.js project for FlyingForge frontend migration.

1. Create new Next.js project in /web-next directory:
   ```bash
   npx create-next-app@latest web-next --typescript --tailwind --eslint --app --src-dir
   ```

2. Copy and adapt configuration files:
   - Copy /web/tailwind.config.js → /web-next/tailwind.config.ts
     - Keep all custom colors, fonts, and plugins
   - Copy /web/postcss.config.js → /web-next/postcss.config.mjs
   - Merge /web/tsconfig.json settings into /web-next/tsconfig.json
     - Keep strict mode, path aliases

3. Create /web-next/next.config.ts:
   ```typescript
   import type { NextConfig } from 'next';
   
   const nextConfig: NextConfig = {
     // Proxy API calls to Go backend
     async rewrites() {
       return [
         {
           source: '/api/:path*',
           destination: 'http://localhost:8080/api/:path*',
         },
       ];
     },
     // Image optimization for external domains
     images: {
       remotePatterns: [
         { protocol: 'https', hostname: '*.googleusercontent.com' },
         { protocol: 'https', hostname: 'flyingforge.com' },
       ],
     },
   };
   
   export default nextConfig;
   ```

4. Update /web-next/src/app/globals.css:
   - Copy content from /web/src/index.css
   - Keep all Tailwind directives and custom styles

5. Create environment files:
   - /web-next/.env.local (development)
   - /web-next/.env.production
   
   Convert all VITE_* variables:
   - VITE_API_URL → NEXT_PUBLIC_API_URL
   - VITE_GOOGLE_CLIENT_ID → NEXT_PUBLIC_GOOGLE_CLIENT_ID
   - etc.

6. Verify build succeeds: `npm run build`
```

---

### Story 1.2: Copy Shared Assets and Types

**Effort:** 1 day

**Acceptance Criteria:**
- [ ] All TypeScript types copied
- [ ] All API client files copied
- [ ] Public assets copied
- [ ] No import errors

**Prompt:**
```
Copy shared code from /web to /web-next that doesn't require modification.

1. Copy all type definition files:
   ```
   /web/src/types.ts → /web-next/src/types/news.ts
   /web/src/authTypes.ts → /web-next/src/types/auth.ts
   /web/src/equipmentTypes.ts → /web-next/src/types/equipment.ts
   /web/src/aircraftTypes.ts → /web-next/src/types/aircraft.ts
   /web/src/radioTypes.ts → /web-next/src/types/radio.ts
   /web/src/batteryTypes.ts → /web-next/src/types/battery.ts
   /web/src/socialTypes.ts → /web-next/src/types/social.ts
   /web/src/fcConfigTypes.ts → /web-next/src/types/fcConfig.ts
   /web/src/gearCatalogTypes.ts → /web-next/src/types/gearCatalog.ts
   ```
   
   Create /web-next/src/types/index.ts barrel export.

2. Copy API client files:
   ```
   /web/src/api.ts → /web-next/src/lib/api/news.ts
   /web/src/authApi.ts → /web-next/src/lib/api/auth.ts
   /web/src/equipmentApi.ts → /web-next/src/lib/api/equipment.ts
   /web/src/aircraftApi.ts → /web-next/src/lib/api/aircraft.ts
   /web/src/radioApi.ts → /web-next/src/lib/api/radio.ts
   /web/src/batteryApi.ts → /web-next/src/lib/api/battery.ts
   /web/src/socialApi.ts → /web-next/src/lib/api/social.ts
   /web/src/adminApi.ts → /web-next/src/lib/api/admin.ts
   /web/src/pilotApi.ts → /web-next/src/lib/api/pilot.ts
   /web/src/profileApi.ts → /web-next/src/lib/api/profile.ts
   /web/src/fcConfigApi.ts → /web-next/src/lib/api/fcConfig.ts
   /web/src/gearCatalogApi.ts → /web-next/src/lib/api/gearCatalog.ts
   ```
   
   Update all import paths in API files.

3. Copy public assets:
   ```
   /web/public/* → /web-next/public/*
   ```

4. Copy custom hooks:
   ```
   /web/src/hooks.ts → /web-next/src/hooks/useFilters.ts (split)
   /web/src/hooks/useAuth.ts → /web-next/src/hooks/useAuth.ts
   /web/src/hooks/useGoogleAnalytics.ts → /web-next/src/hooks/useGoogleAnalytics.ts
   ```

5. Update all import paths to use new structure
6. Run `npm run build` to verify no import errors
```

---

### Story 1.3: Update Docker and Infrastructure

**Effort:** 1 day

**Acceptance Criteria:**
- [ ] Dockerfile.web updated for Next.js
- [ ] docker-compose.yml updated
- [ ] nginx.conf updated or removed
- [ ] Terraform ECS task definitions updated

**Prompt:**
```
Update Docker and infrastructure configuration for Next.js.

1. Replace /Dockerfile.web with Next.js optimized Dockerfile:
   ```dockerfile
   # Stage 1: Dependencies
   FROM node:20-alpine AS deps
   WORKDIR /app
   COPY web-next/package*.json ./
   RUN npm ci --only=production

   # Stage 2: Build
   FROM node:20-alpine AS builder
   WORKDIR /app
   COPY --from=deps /app/node_modules ./node_modules
   COPY web-next/ .
   ENV NEXT_TELEMETRY_DISABLED=1
   RUN npm run build

   # Stage 3: Production
   FROM node:20-alpine AS runner
   WORKDIR /app
   ENV NODE_ENV=production
   ENV NEXT_TELEMETRY_DISABLED=1

   RUN addgroup --system --gid 1001 nodejs
   RUN adduser --system --uid 1001 nextjs

   COPY --from=builder /app/public ./public
   COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
   COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

   USER nextjs
   EXPOSE 3000
   ENV PORT=3000
   CMD ["node", "server.js"]
   ```

2. Update next.config.ts for standalone output:
   ```typescript
   const nextConfig: NextConfig = {
     output: 'standalone',
     // ... existing config
   };
   ```

3. Update /docker-compose.yml:
   - Change web service port from 80 to 3000
   - Remove nginx dependency (Next.js serves directly)
   - Update health check endpoint

4. Remove /web/nginx.conf and /web/nginx.ecs.conf (no longer needed)

5. Update /terraform/ecs.tf:
   - Change container port from 80 to 3000
   - Update health check path to /api/health or custom Next.js route
   - Adjust memory/CPU if needed (Next.js may need more)

6. Test locally:
   ```bash
   docker-compose build web
   docker-compose up web
   curl http://localhost:3000
   ```
```

---

## Phase 2: Authentication Migration

### Story 2.1: Decide Auth Strategy

**Effort:** 0.5 days (decision) + implementation in 2.2

**Decision Required:** Choose between two approaches:

**Option A: Keep localStorage (Client-Only Auth)**
- Pros: Minimal changes, same as current
- Cons: All pages must be client-rendered, no SSR benefits
- Implementation: Add `'use client'` to all pages using auth

**Option B: HTTP-Only Cookies (Server-Compatible Auth)**
- Pros: Works with SSR, more secure (no XSS token theft)
- Cons: Requires Go backend changes, more complex
- Implementation: Go sets cookie, Next.js reads via middleware

**Prompt:**
```
Evaluate authentication strategy for Next.js migration.

Current implementation in /web/src/authApi.ts:
- JWT stored in localStorage
- Token attached to fetch requests via Authorization header
- Refresh token flow on 401 responses

For Option A (localStorage):
1. All auth-dependent pages use 'use client'
2. AuthContext works identically
3. No backend changes needed
4. SSR only for public pages

For Option B (HTTP-only cookies):
1. Modify Go backend /api/auth/google/callback to set cookie:
   ```go
   http.SetCookie(w, &http.Cookie{
       Name:     "auth_token",
       Value:    token,
       HttpOnly: true,
       Secure:   true,
       SameSite: http.SameSiteStrictMode,
       Path:     "/",
       MaxAge:   86400 * 7, // 7 days
   })
   ```

2. Create Next.js middleware to read cookie and inject into requests
3. Update all API calls to not send Authorization header (cookie sent automatically)

Recommendation: Start with Option A for faster migration, convert to Option B later for enhanced security.
```

---

### Story 2.2: Migrate Auth Context

**Effort:** 2-3 days

**Acceptance Criteria:**
- [ ] AuthContext works in Next.js
- [ ] Google OAuth flow works end-to-end
- [ ] Token refresh works
- [ ] Protected routes redirect to login
- [ ] Logout clears state

**Prompt:**
```
Migrate authentication system to Next.js.

1. Create /web-next/src/contexts/AuthContext.tsx:
   - Copy from /web/src/contexts/AuthContext.tsx
   - Add 'use client' directive at top
   - Update import paths

2. Create /web-next/src/providers/AuthProvider.tsx:
   ```tsx
   'use client';
   
   import { AuthProvider as AuthContextProvider } from '@/contexts/AuthContext';
   
   export function AuthProvider({ children }: { children: React.ReactNode }) {
     return <AuthContextProvider>{children}</AuthContextProvider>;
   }
   ```

3. Update /web-next/src/app/layout.tsx:
   ```tsx
   import { AuthProvider } from '@/providers/AuthProvider';
   
   export default function RootLayout({ children }) {
     return (
       <html lang="en">
         <body>
           <AuthProvider>
             {children}
           </AuthProvider>
         </body>
       </html>
     );
   }
   ```

4. Create auth callback route /web-next/src/app/auth/callback/page.tsx:
   ```tsx
   'use client';
   
   import { AuthCallback } from '@/components/AuthCallback';
   
   export default function AuthCallbackPage() {
     return <AuthCallback />;
   }
   ```

5. Create protected route wrapper /web-next/src/components/ProtectedRoute.tsx:
   ```tsx
   'use client';
   
   import { useAuth } from '@/hooks/useAuth';
   import { useRouter } from 'next/navigation';
   import { useEffect } from 'react';
   
   export function ProtectedRoute({ children }: { children: React.ReactNode }) {
     const { isAuthenticated, isLoading } = useAuth();
     const router = useRouter();
     
     useEffect(() => {
       if (!isLoading && !isAuthenticated) {
         router.push('/login');
       }
     }, [isLoading, isAuthenticated, router]);
     
     if (isLoading) return <LoadingSpinner />;
     if (!isAuthenticated) return null;
     
     return <>{children}</>;
   }
   ```

6. Test OAuth flow:
   - Click "Sign in with Google"
   - Complete Google flow
   - Verify redirect back to app
   - Verify token stored in localStorage
   - Verify protected routes accessible
```

---

## Phase 3: Route and Page Migration

### Story 3.1: Create App Router Structure

**Effort:** 1 day

**Acceptance Criteria:**
- [ ] All routes defined in /app directory
- [ ] Layout hierarchy established
- [ ] Loading and error states defined
- [ ] Route groups for organization

**Prompt:**
```
Create Next.js App Router file structure matching current React Router routes.

Create this directory structure in /web-next/src/app/:

```
app/
├── layout.tsx              # Root layout (AuthProvider, global styles)
├── page.tsx                # Homepage (/ route)
├── loading.tsx             # Global loading state
├── error.tsx               # Global error boundary
├── not-found.tsx           # 404 page
│
├── (public)/               # Route group for public pages
│   ├── news/
│   │   ├── page.tsx        # News feed
│   │   └── loading.tsx
│   ├── gear-catalog/
│   │   ├── page.tsx        # Public gear catalog
│   │   └── loading.tsx
│   └── getting-started/
│       └── page.tsx
│
├── (auth)/                 # Route group for auth pages
│   ├── login/
│   │   └── page.tsx
│   └── auth/
│       └── callback/
│           └── page.tsx
│
├── (protected)/            # Route group for protected pages
│   ├── layout.tsx          # Wraps children in ProtectedRoute
│   ├── dashboard/
│   │   ├── page.tsx
│   │   └── loading.tsx
│   ├── inventory/
│   │   ├── page.tsx
│   │   └── loading.tsx
│   ├── aircraft/
│   │   ├── page.tsx
│   │   ├── loading.tsx
│   │   ├── new/
│   │   │   └── page.tsx
│   │   └── [id]/
│   │       ├── page.tsx
│   │       └── edit/
│   │           └── page.tsx
│   ├── radios/
│   │   └── page.tsx
│   ├── batteries/
│   │   └── page.tsx
│   ├── social/
│   │   └── page.tsx
│   ├── profile/
│   │   └── page.tsx
│   └── pilots/
│       └── [id]/
│           └── page.tsx
│
└── (admin)/                # Route group for admin pages
    └── admin/
        └── gear/
            └── page.tsx
```

For each page.tsx, create a placeholder:
```tsx
'use client';

export default function PageName() {
  return <div>PageName - TODO: migrate component</div>;
}
```

For (protected)/layout.tsx:
```tsx
import { ProtectedRoute } from '@/components/ProtectedRoute';

export default function ProtectedLayout({ children }: { children: React.ReactNode }) {
  return <ProtectedRoute>{children}</ProtectedRoute>;
}
```
```

---

### Story 3.2: Create Shared Layout Components

**Effort:** 2 days

**Acceptance Criteria:**
- [ ] Sidebar component migrated
- [ ] TopBar component migrated
- [ ] Mobile menu works
- [ ] Navigation between sections works

**Prompt:**
```
Migrate layout components (Sidebar, TopBar) to Next.js.

1. Copy /web/src/components/Sidebar.tsx → /web-next/src/components/Sidebar.tsx
   - Add 'use client' directive
   - Replace React Router's useNavigate with Next.js useRouter
   - Replace Link from react-router-dom with next/link
   - Update all href paths to Next.js routes

   Changes needed:
   ```tsx
   // Before
   import { useNavigate } from 'react-router-dom';
   const navigate = useNavigate();
   onClick={() => navigate('/dashboard')}
   
   // After
   import { useRouter } from 'next/navigation';
   import Link from 'next/link';
   const router = useRouter();
   onClick={() => router.push('/dashboard')}
   // Or use Link component directly
   ```

2. Copy /web/src/components/TopBar.tsx → /web-next/src/components/TopBar.tsx
   - Add 'use client' directive
   - Same navigation changes as Sidebar

3. Create shared layout in /web-next/src/app/(protected)/layout.tsx:
   ```tsx
   'use client';
   
   import { Sidebar } from '@/components/Sidebar';
   import { ProtectedRoute } from '@/components/ProtectedRoute';
   import { useState } from 'react';
   
   export default function ProtectedLayout({ children }) {
     const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
     
     return (
       <ProtectedRoute>
         <div className="flex h-screen bg-slate-900">
           <Sidebar 
             isMobileMenuOpen={isMobileMenuOpen}
             onMobileMenuClose={() => setIsMobileMenuOpen(false)}
           />
           <main className="flex-1 flex flex-col overflow-hidden">
             {children}
           </main>
         </div>
       </ProtectedRoute>
     );
   }
   ```

4. Handle active section highlighting:
   - Use usePathname() from next/navigation
   - Derive active section from pathname
   ```tsx
   import { usePathname } from 'next/navigation';
   const pathname = usePathname();
   const activeSection = pathname.split('/')[1] || 'dashboard';
   ```

5. Test navigation between all protected routes
```

---

### Story 3.3: Migrate News Section (Public SSR Page)

**Effort:** 2-3 days

**Acceptance Criteria:**
- [ ] News page renders server-side for SEO
- [ ] Filters work client-side
- [ ] Infinite scroll works
- [ ] Search/filter state preserved in URL

**Prompt:**
```
Migrate the News section with Server-Side Rendering for SEO.

This is the most complex migration as it demonstrates SSR + client interactivity.

1. Create /web-next/src/app/(public)/news/page.tsx:
   ```tsx
   import { Suspense } from 'react';
   import { NewsContent } from './NewsContent';
   import { NewsSkeleton } from './NewsSkeleton';
   
   // Server Component - fetches initial data
   export default async function NewsPage({
     searchParams,
   }: {
     searchParams: { q?: string; from?: string; to?: string; sort?: string };
   }) {
     // Fetch initial items server-side
     const initialItems = await fetch(
       `${process.env.API_URL}/api/items?${new URLSearchParams(searchParams)}`,
       { next: { revalidate: 60 } } // Cache for 60 seconds
     ).then(r => r.json());
     
     return (
       <Suspense fallback={<NewsSkeleton />}>
         <NewsContent initialItems={initialItems} />
       </Suspense>
     );
   }
   ```

2. Create /web-next/src/app/(public)/news/NewsContent.tsx:
   ```tsx
   'use client';
   
   import { useState } from 'react';
   import { useRouter, useSearchParams } from 'next/navigation';
   import { TopBar } from '@/components/TopBar';
   import { FeedList } from '@/components/FeedList';
   import { useDebounce } from '@/hooks/useDebounce';
   
   export function NewsContent({ initialItems }) {
     const router = useRouter();
     const searchParams = useSearchParams();
     
     const [items, setItems] = useState(initialItems.items);
     const [query, setQuery] = useState(searchParams.get('q') || '');
     // ... rest of filter state
     
     // Update URL when filters change
     const updateFilters = (newFilters) => {
       const params = new URLSearchParams(newFilters);
       router.push(`/news?${params.toString()}`);
     };
     
     // Client-side infinite scroll for more items
     const loadMore = async () => {
       const moreItems = await fetchItems({ ...filters, offset: items.length });
       setItems([...items, ...moreItems]);
     };
     
     return (
       <>
         <TopBar filters={filters} onFilterChange={updateFilters} />
         <FeedList items={items} onLoadMore={loadMore} />
       </>
     );
   }
   ```

3. Copy and update FeedList, FeedCard, TopBar components:
   - Add 'use client' directive
   - Update image handling to use next/image:
     ```tsx
     import Image from 'next/image';
     <Image src={item.image} alt={item.title} width={200} height={150} />
     ```

4. Handle metadata for SEO in page.tsx:
   ```tsx
   export const metadata = {
     title: 'Drone News | FlyingForge',
     description: 'Latest FPV drone news, reviews, and announcements',
   };
   ```

5. Test:
   - Page loads with SSR (view source shows content)
   - Filters update URL
   - Sharing URL preserves filters
   - Infinite scroll loads more client-side
```

---

### Story 3.4: Migrate Dashboard Section

**Effort:** 1-2 days

**Acceptance Criteria:**
- [ ] Dashboard loads with user data
- [ ] All widgets display correctly
- [ ] Navigation to other sections works
- [ ] Loading states work

**Prompt:**
```
Migrate the Dashboard section to Next.js.

1. Create /web-next/src/app/(protected)/dashboard/page.tsx:
   ```tsx
   'use client';
   
   import { Dashboard } from '@/components/Dashboard';
   import { useAuth } from '@/hooks/useAuth';
   import { useQuery } from '@tanstack/react-query'; // If added, otherwise useEffect
   
   export default function DashboardPage() {
     const { user } = useAuth();
     
     // Fetch dashboard data
     const [recentAircraft, setRecentAircraft] = useState([]);
     const [recentNews, setRecentNews] = useState([]);
     const [isLoading, setIsLoading] = useState(true);
     
     useEffect(() => {
       async function loadDashboard() {
         const [aircraft, news] = await Promise.all([
           fetchRecentAircraft(),
           fetchRecentNews(),
         ]);
         setRecentAircraft(aircraft);
         setRecentNews(news);
         setIsLoading(false);
       }
       loadDashboard();
     }, []);
     
     return (
       <Dashboard
         user={user}
         recentAircraft={recentAircraft}
         recentNews={recentNews}
         isLoading={isLoading}
         onViewAllNews={() => router.push('/news')}
         onViewAllAircraft={() => router.push('/aircraft')}
       />
     );
   }
   ```

2. Copy /web/src/components/Dashboard.tsx → /web-next/src/components/Dashboard.tsx
   - Add 'use client' directive
   - Update navigation callbacks to use router.push()
   - Replace any react-router-dom Link with next/link

3. Create loading state /web-next/src/app/(protected)/dashboard/loading.tsx:
   ```tsx
   export default function DashboardLoading() {
     return <DashboardSkeleton />;
   }
   ```

4. Test:
   - Dashboard loads after login
   - Recent items display
   - Click handlers navigate correctly
```

---

### Story 3.5: Migrate Remaining Protected Sections

**Effort:** 5-7 days

**Acceptance Criteria:**
- [ ] All protected pages migrated
- [ ] All CRUD operations work
- [ ] All forms work
- [ ] All modals work

**Prompt:**
```
Migrate remaining protected sections to Next.js. Follow the same pattern for each.

Sections to migrate (in priority order):

1. **Inventory Section** (/inventory)
   - Copy InventoryList, InventoryCard, AddInventoryModal, AddGearModal
   - Add 'use client' to all
   - Create page.tsx with data fetching

2. **Aircraft Section** (/aircraft, /aircraft/new, /aircraft/[id], /aircraft/[id]/edit)
   - Copy AircraftList, AircraftCard, AircraftForm, AircraftDetail
   - Dynamic route: /aircraft/[id]/page.tsx uses params.id
   - Handle image uploads

3. **Batteries Section** (/batteries)
   - Copy BatterySection component
   - Handle PDF label generation (window.open)

4. **Radios Section** (/radios)
   - Copy RadioSection component
   - Handle file uploads/downloads for backups

5. **Social Section** (/social)
   - Copy SocialPage, PilotSearch, FollowListModal
   - Handle pilot profiles

6. **Profile Section** (/profile)
   - Copy MyProfile component
   - Handle avatar upload

7. **Pilot Profiles** (/pilots/[id])
   - Copy PilotProfile, PublicAircraftModal
   - Dynamic route with params.id

8. **Admin Gear Moderation** (/admin/gear)
   - Copy AdminGearModeration
   - Check admin role in layout

For EACH component migration:
- Add 'use client' directive
- Replace useNavigate → useRouter
- Replace Link from react-router-dom → next/link
- Update any window/document access with useEffect guards
- Test all interactive features

Template for each page:
```tsx
'use client';

import { ComponentName } from '@/components/ComponentName';
// ... other imports

export default function PageName() {
  // Fetch data with useEffect or useQuery
  // Handle loading/error states
  
  return <ComponentName {...props} />;
}
```
```

---

## Phase 4: Polish and Testing

### Story 4.1: Add Loading and Error States

**Effort:** 1 day

**Acceptance Criteria:**
- [ ] All routes have loading.tsx
- [ ] All routes have error.tsx
- [ ] Global not-found.tsx works
- [ ] Error recovery works

**Prompt:**
```
Add comprehensive loading and error states throughout the Next.js app.

1. Create global error boundary /web-next/src/app/error.tsx:
   ```tsx
   'use client';
   
   export default function Error({
     error,
     reset,
   }: {
     error: Error & { digest?: string };
     reset: () => void;
   }) {
     return (
       <div className="min-h-screen bg-slate-900 flex items-center justify-center">
         <div className="text-center">
           <h2 className="text-2xl font-bold text-white mb-4">Something went wrong</h2>
           <p className="text-slate-400 mb-6">{error.message}</p>
           <button
             onClick={reset}
             className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
           >
             Try again
           </button>
         </div>
       </div>
     );
   }
   ```

2. Create global not-found /web-next/src/app/not-found.tsx:
   ```tsx
   import Link from 'next/link';
   
   export default function NotFound() {
     return (
       <div className="min-h-screen bg-slate-900 flex items-center justify-center">
         <div className="text-center">
           <h2 className="text-4xl font-bold text-white mb-4">404</h2>
           <p className="text-slate-400 mb-6">Page not found</p>
           <Link href="/" className="text-primary-400 hover:underline">
             Go home
           </Link>
         </div>
       </div>
     );
   }
   ```

3. Create shared loading skeleton /web-next/src/components/LoadingSkeleton.tsx

4. Add loading.tsx to each route group:
   - /app/(protected)/loading.tsx
   - /app/(public)/news/loading.tsx
   - etc.

5. Test:
   - Slow 3G shows loading states
   - Throwing error in component shows error boundary
   - Invalid URLs show 404
```

---

### Story 4.2: Migrate Tests

**Effort:** 2-3 days

**Acceptance Criteria:**
- [ ] Jest configured (or Vitest kept)
- [ ] All existing tests pass
- [ ] Testing Library works with Next.js
- [ ] Coverage maintained

**Prompt:**
```
Migrate test infrastructure to Next.js.

Option A: Keep Vitest (recommended for faster migration)
1. Install vitest for Next.js:
   ```bash
   npm install -D vitest @vitejs/plugin-react jsdom @testing-library/react @testing-library/jest-dom
   ```

2. Create /web-next/vitest.config.ts:
   ```typescript
   import { defineConfig } from 'vitest/config';
   import react from '@vitejs/plugin-react';
   
   export default defineConfig({
     plugins: [react()],
     test: {
       environment: 'jsdom',
       setupFiles: './src/test/setup.ts',
       globals: true,
     },
     resolve: {
       alias: {
         '@': './src',
       },
     },
   });
   ```

3. Copy test files:
   - /web/src/components/*.test.tsx → /web-next/src/components/*.test.tsx
   - /web/src/hooks/*.test.tsx → /web-next/src/hooks/*.test.tsx
   - /web/src/test/* → /web-next/src/test/*

4. Update test imports:
   ```tsx
   // Before
   import { BrowserRouter } from 'react-router-dom';
   render(<BrowserRouter><Component /></BrowserRouter>);
   
   // After - mock next/navigation
   vi.mock('next/navigation', () => ({
     useRouter: () => ({ push: vi.fn(), back: vi.fn() }),
     usePathname: () => '/test',
     useSearchParams: () => new URLSearchParams(),
   }));
   render(<Component />);
   ```

5. Run tests: `npm test`
6. Fix any failing tests due to Next.js-specific changes
```

---

### Story 4.3: Performance Optimization

**Effort:** 1-2 days

**Acceptance Criteria:**
- [ ] Images use next/image
- [ ] Dynamic imports for heavy components
- [ ] Bundle size comparable or smaller
- [ ] Lighthouse score maintained

**Prompt:**
```
Optimize Next.js application performance.

1. Convert all images to next/image:
   ```tsx
   // Before
   <img src={url} alt={alt} className="..." />
   
   // After
   import Image from 'next/image';
   <Image src={url} alt={alt} width={200} height={150} className="..." />
   ```
   
   For dynamic images where dimensions unknown:
   ```tsx
   <Image src={url} alt={alt} fill className="object-cover" />
   ```

2. Add dynamic imports for heavy components:
   ```tsx
   import dynamic from 'next/dynamic';
   
   const HeavyChart = dynamic(() => import('@/components/HeavyChart'), {
     loading: () => <ChartSkeleton />,
     ssr: false, // If component uses browser APIs
   });
   ```

3. Configure image optimization in next.config.ts:
   ```typescript
   images: {
     formats: ['image/avif', 'image/webp'],
     deviceSizes: [640, 750, 828, 1080, 1200],
   },
   ```

4. Analyze bundle:
   ```bash
   npm install @next/bundle-analyzer
   ANALYZE=true npm run build
   ```

5. Run Lighthouse audit and compare to current Vite build

6. Add prefetching for common navigation:
   ```tsx
   <Link href="/dashboard" prefetch={true}>Dashboard</Link>
   ```
```

---

### Story 4.4: Final QA and Deployment

**Effort:** 2-3 days

**Acceptance Criteria:**
- [ ] All features work identically to Vite version
- [ ] No console errors
- [ ] Mobile responsive works
- [ ] Production build succeeds
- [ ] Docker container runs correctly
- [ ] Staging deployment successful

**Prompt:**
```
Final QA checklist and deployment for Next.js migration.

## QA Checklist

### Authentication
- [ ] Google OAuth login works
- [ ] Token refresh works
- [ ] Logout clears state
- [ ] Protected routes redirect when logged out
- [ ] Admin-only routes check role

### Public Pages
- [ ] Homepage loads
- [ ] News page renders server-side (view source)
- [ ] News filters work
- [ ] News infinite scroll works
- [ ] Getting started page loads
- [ ] Gear catalog search works

### Protected Pages
- [ ] Dashboard loads with data
- [ ] Inventory CRUD works
- [ ] Aircraft CRUD works
- [ ] Aircraft image upload works
- [ ] Batteries section works
- [ ] Battery label PDF works
- [ ] Radios section works
- [ ] Radio backup upload/download works
- [ ] Social features work
- [ ] Follow/unfollow works
- [ ] Profile edit works
- [ ] Avatar upload works

### Admin
- [ ] Gear moderation loads for admins
- [ ] Non-admins cannot access

### Mobile
- [ ] Sidebar menu toggles
- [ ] Touch interactions work
- [ ] No horizontal scroll

### Performance
- [ ] Initial load < 3s on 3G
- [ ] Lighthouse performance > 80
- [ ] No layout shift

## Deployment

1. Build and test locally:
   ```bash
   npm run build
   npm start
   # Test at http://localhost:3000
   ```

2. Build Docker image:
   ```bash
   docker build -f Dockerfile.web -t flyingforge-web:nextjs .
   docker run -p 3000:3000 flyingforge-web:nextjs
   ```

3. Deploy to staging:
   ```bash
   # Update terraform variables for staging
   terraform plan -var-file=staging.tfvars
   terraform apply -var-file=staging.tfvars
   ```

4. Run full QA on staging environment

5. Deploy to production:
   ```bash
   terraform plan -var-file=production.tfvars
   terraform apply -var-file=production.tfvars
   ```

6. Monitor for errors in first 24 hours

7. Once stable, delete /web directory and rename /web-next to /web
```

---

## Summary

| Phase | Stories | Effort |
|-------|---------|--------|
| Phase 1: Setup | 3 | 4-5 days |
| Phase 2: Auth | 2 | 2.5-3.5 days |
| Phase 3: Routes | 5 | 11-15 days |
| Phase 4: Polish | 4 | 6-9 days |
| **Total** | **14** | **23.5-32.5 days** |

**Realistic timeline: 6-8 weeks with 1-2 developers**

---

## Decision Log

| Decision | Options | Chosen | Rationale |
|----------|---------|--------|-----------|
| Router | Pages vs App | App Router | Better SSR patterns, future-proof |
| Auth Strategy | localStorage vs Cookies | localStorage (Phase 1) | Faster migration, can convert later |
| State Management | Context vs TanStack Query | Context (existing) | Minimize scope, add Query later |
| Testing | Vitest vs Jest | Vitest | Existing tests, faster |
| Image Handling | img vs next/image | next/image | Built-in optimization |
