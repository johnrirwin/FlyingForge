import { AdminAnnouncementsPanel } from './AdminAnnouncementsPanel';

interface AdminAnnouncementsPageProps {
  hasContentAdminAccess: boolean;
  authLoading?: boolean;
}

export function AdminAnnouncementsPage({ hasContentAdminAccess, authLoading }: AdminAnnouncementsPageProps) {
  if (authLoading) {
    return (
      <div className="p-8 text-center">
        <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-primary-500/30 border-t-primary-500" />
        <p className="mt-4 text-slate-400">Loading...</p>
      </div>
    );
  }

  if (!hasContentAdminAccess) {
    return (
      <div className="p-8 text-center">
        <h1 className="mb-4 text-2xl font-bold text-red-400">Access Denied</h1>
        <p className="text-slate-400">You must be an admin or content admin to access this page.</p>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <AdminAnnouncementsPanel />
    </div>
  );
}
