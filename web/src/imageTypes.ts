export type ModerationStatus = 'APPROVED' | 'REJECTED' | 'PENDING_REVIEW';

export interface ImageModerationResponse {
  status: ModerationStatus;
  reason?: string;
  uploadId?: string;
}
