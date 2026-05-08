export type AnnouncementStatus = 'draft' | 'published' | 'archived';
export type AnnouncementPlacement = 'global' | 'home' | 'dashboard' | 'news';
export type AnnouncementAudience = 'all' | 'signed_in' | 'signed_out';

export interface Announcement {
  id: string;
  title: string;
  body: string;
  status: AnnouncementStatus;
  priority: number;
  placements: AnnouncementPlacement[];
  audience: AnnouncementAudience;
  ctaLabel?: string;
  ctaUrl?: string;
  dismissible: boolean;
  startsAt?: string;
  endsAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AnnouncementListResponse {
  announcements: Announcement[];
  totalCount: number;
}

export interface AdminAnnouncementListParams {
  query?: string;
  status?: AnnouncementStatus;
  limit?: number;
  offset?: number;
}

export interface SaveAnnouncementParams {
  title: string;
  body: string;
  status: AnnouncementStatus;
  priority: number;
  placements: AnnouncementPlacement[];
  audience: AnnouncementAudience;
  ctaLabel?: string;
  ctaUrl?: string;
  dismissible: boolean;
  startsAt?: string;
  endsAt?: string;
}
