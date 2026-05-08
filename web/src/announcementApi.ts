import { getStoredTokens } from './authApi';
import type { AnnouncementListResponse, AnnouncementPlacement } from './announcementTypes';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

export async function getActiveAnnouncements(placement: AnnouncementPlacement): Promise<AnnouncementListResponse> {
  const tokens = getStoredTokens();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  if (tokens?.accessToken) {
    (headers as Record<string, string>).Authorization = `Bearer ${tokens.accessToken}`;
  }

  const response = await fetch(`${API_BASE}/api/announcements/active?placement=${encodeURIComponent(placement)}`, {
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || error.message || 'Failed to load announcements');
  }

  return response.json();
}
