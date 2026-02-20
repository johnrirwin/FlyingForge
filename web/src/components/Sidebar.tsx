import { type ReactNode, memo } from 'react';
import type { AppSection } from '../equipmentTypes';
import type { User } from '../authTypes';

interface SidebarProps {
  activeSection: AppSection;
  onSectionChange: (section: AppSection) => void;
  // Auth props
  isAuthenticated: boolean;
  user: User | null;
  authLoading: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
  // Mobile props
  isMobileMenuOpen: boolean;
  onMobileMenuClose: () => void;
}

export const Sidebar = memo(function Sidebar({
  activeSection,
  onSectionChange,
  isAuthenticated,
  user,
  authLoading,
  onSignIn,
  onSignOut,
  isMobileMenuOpen,
  onMobileMenuClose,
}: SidebarProps) {
  // Handle navigation and close mobile menu
  const handleNavigation = (section: AppSection) => {
    onSectionChange(section);
    onMobileMenuClose();
  };

  // Navigation item component with lock state for unauthenticated sections
  const NavItem = ({
    section,
    icon,
    label,
    requiresAuth = false,
    badge,
  }: {
    section: AppSection;
    icon: ReactNode;
    label: string;
    requiresAuth?: boolean;
    badge?: number;
  }) => {
    const isLocked = requiresAuth && !isAuthenticated;
    const isActive = activeSection === section;

    return (
      <button
        onClick={() => handleNavigation(section)}
        disabled={false} // Always allow click - handleSectionChange in App.tsx will handle auth prompt
        className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors ${
          isActive
            ? 'bg-primary-600/20 text-primary-400'
            : isLocked
            ? 'text-slate-500 hover:text-slate-400 hover:bg-slate-800/50'
            : 'text-slate-400 hover:text-white hover:bg-slate-800'
        }`}
        title={isLocked ? 'Sign in to access' : undefined}
      >
        {icon}
        <span className="font-medium flex-1">{label}</span>
        {isLocked && (
          <svg className="w-4 h-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        )}
        {badge !== undefined && badge > 0 && !isLocked && (
          <span className="px-2 py-0.5 bg-slate-700 rounded-full text-xs text-slate-300">
            {badge}
          </span>
        )}
      </button>
    );
  };

  return (
    <>
      {/* Mobile overlay backdrop */}
      {isMobileMenuOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 md:hidden"
          onClick={onMobileMenuClose}
        />
      )}
      
      {/* Sidebar */}
      <aside
        className={`
          fixed md:static top-0 left-0 z-50 h-screen supports-[height:100dvh]:h-[100dvh] md:h-auto
          w-64 flex-shrink-0 bg-slate-900 border-r border-slate-800 flex flex-col
          overflow-y-auto overflow-x-hidden overscroll-y-contain [-webkit-overflow-scrolling:touch]
          transform transition-transform duration-300 ease-in-out
          ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
        `}
      >
        {/* Mobile close button */}
        <div className="md:hidden flex justify-end px-2 pt-1 pb-0">
          <button
            onClick={onMobileMenuClose}
            className="p-2 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Section Switcher */}
        <div className="px-4 pt-1 pb-4 md:p-4">
        <nav className="flex flex-col gap-1">
          {/* Public sections */}
          {!isAuthenticated && (
            <NavItem
              section="home"
              label="Home"
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                </svg>
              }
            />
          )}

          {!isAuthenticated && (
            <NavItem
              section="getting-started"
              label="Taking Off"
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
              }
            />
          )}

          <NavItem
            section="news"
            label="News Feed"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
              </svg>
            }
          />

          <NavItem
            section="equipment"
            label="Shop"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
              </svg>
            }
          />

          <NavItem
            section="gear-catalog"
            label="Gear Catalog"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            }
          />

          <NavItem
            section="builds"
            label="Builds"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3h6m-6 18h6M5 7h14M5 17h14M7 7v10m10-10v10M9 9h6v6H9V9z" />
              </svg>
            }
          />

          {/* Authenticated sections */}
          <div className="my-2 border-t border-slate-800" />

          {isAuthenticated && (
            <NavItem
              section="dashboard"
              label="Dashboard"
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              }
            />
          )}

          <NavItem
            section="social"
            label="Social"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            }
          />

          <NavItem
            section="inventory"
            label="My Inventory"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            }
          />

          <NavItem
            section="aircraft"
            label="My Aircraft"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            }
          />

          <NavItem
            section="radio"
            label="My Radio"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
              </svg>
            }
          />

          <NavItem
            section="batteries"
            label="My Batteries"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h14a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4a2 2 0 012-2zm16 3h2m-2 0v2" />
              </svg>
            }
          />

          <NavItem
            section="my-builds"
            label="My Builds"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7h18M6 7l1-3h10l1 3M5 7v13h14V7M9 11h6m-6 4h4" />
              </svg>
            }
          />

          {/* Admin sections */}
          {(user?.isAdmin || user?.isContentAdmin || user?.isGearAdmin) && <div className="my-2 border-t border-slate-800" />}

          {(user?.isAdmin || user?.isContentAdmin || user?.isGearAdmin) && (
            <NavItem
              section="admin-content"
              label="Content Moderation"
              requiresAuth
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              }
            />
          )}

          {/* Admin: User Admin - only shown to full admins */}
          {user?.isAdmin && (
            <NavItem
              section="admin-users"
              label="User Admin"
              requiresAuth
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              }
            />
          )}
        </nav>
      </div>

      {/* Spacer to push user section to bottom */}
      <div className="flex-1" />

      {/* User section at bottom */}
      <div className="p-4 border-t border-slate-800">
        {authLoading ? (
          <div className="flex items-center justify-center py-2">
            <div className="w-5 h-5 border-2 border-slate-600 border-t-primary-500 rounded-full animate-spin" />
          </div>
        ) : isAuthenticated && user ? (
          <div className="space-y-3">
            <button
              onClick={() => handleNavigation('profile')}
              className="w-full flex items-center gap-3 p-2 -m-2 rounded-lg hover:bg-slate-800 transition-colors"
            >
              {user.avatarUrl ? (
                <img
                  src={user.avatarUrl}
                  alt={user.displayName || 'User'}
                  className="w-9 h-9 rounded-full flex-shrink-0"
                />
              ) : (
                <div className="w-9 h-9 rounded-full bg-primary-600 flex items-center justify-center flex-shrink-0">
                  <span className="text-white text-sm font-medium">
                    {(user.displayName || user.email || '?')[0].toUpperCase()}
                  </span>
                </div>
              )}
              <div className="flex-1 min-w-0 text-left">
                <div className="text-sm font-medium text-white truncate">
                  {user.displayName || user.email}
                </div>
                {user.displayName && user.email && (
                  <div className="text-xs text-slate-500 truncate">
                    {user.email}
                  </div>
                )}
              </div>
            </button>
            <button
              onClick={onSignOut}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              Sign Out
            </button>
          </div>
        ) : (
          <button
            onClick={onSignIn}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
            </svg>
            Sign In
          </button>
        )}
      </div>
    </aside>
    </>
  );
});
