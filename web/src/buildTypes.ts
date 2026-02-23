import type { GearType, CatalogItemStatus } from './gearCatalogTypes';

export type BuildStatus = 'TEMP' | 'SHARED' | 'DRAFT' | 'PENDING_REVIEW' | 'PUBLISHED' | 'UNPUBLISHED';
export type BuildSort = 'newest';
export type BuildReaction = 'LIKE' | 'DISLIKE';

export interface BuildCatalogItem {
  id: string;
  gearType: GearType;
  brand: string;
  model: string;
  variant?: string;
  status: CatalogItemStatus;
  imageUrl?: string;
}

export interface BuildPart {
  id?: string;
  buildId?: string;
  gearType: GearType;
  catalogItemId: string;
  position?: number;
  notes?: string;
  createdAt?: string;
  updatedAt?: string;
  catalogItem?: BuildCatalogItem;
}

export interface BuildPilot {
  userId?: string;
  callSign?: string;
  displayName?: string;
  isProfilePublic: boolean;
  profileUrl?: string;
}

export interface Build {
  id: string;
  ownerUserId?: string;
  status: BuildStatus;
  expiresAt?: string;
  title: string;
  description?: string;
  sourceAircraftId?: string;
  createdAt: string;
  updatedAt: string;
  publishedAt?: string;
  parts: BuildPart[];
  verified: boolean;
  mainImageUrl?: string;
  pilot?: BuildPilot;
  likeCount?: number;
  dislikeCount?: number;
  viewerReaction?: BuildReaction;
}

export interface BuildPartInput {
  gearType: GearType;
  catalogItemId: string;
  position?: number;
  notes?: string;
}

export interface CreateBuildParams {
  title?: string;
  description?: string;
  sourceAircraftId?: string;
  parts?: BuildPartInput[];
}

export interface UpdateBuildParams {
  title?: string;
  description?: string;
  parts?: BuildPartInput[];
}

export interface BuildListParams {
  sort?: BuildSort;
  frameFilter?: string;
  limit?: number;
  offset?: number;
}

export interface BuildListResponse {
  builds: Build[];
  totalCount: number;
  sort?: BuildSort;
  frameFilter?: string;
}

export interface BuildValidationError {
  category: string;
  code: string;
  message: string;
}

export interface BuildValidationResult {
  valid: boolean;
  errors?: BuildValidationError[];
}

export interface BuildPublishResponse {
  build?: Build;
  validation: BuildValidationResult;
}

export interface TempBuildCreateResponse {
  build: Build;
  token: string;
  url: string;
}

export function getBuildPartDisplayName(part?: BuildPart): string {
  if (!part?.catalogItem) return 'Not selected';
  const { brand, model, variant } = part.catalogItem;
  return [brand, model, variant].filter(Boolean).join(' ').trim();
}

export function findPart(parts: BuildPart[] | undefined, gearType: GearType): BuildPart | undefined {
  if (!parts) return undefined;
  return parts.find((part) => part.gearType === gearType);
}

export function upsertPart(parts: BuildPartInput[], nextPart: BuildPartInput | null): BuildPartInput[] {
  const filtered = parts.filter((part) => part.gearType !== (nextPart?.gearType ?? ''));
  if (!nextPart) {
    return filtered;
  }
  return [...filtered, nextPart];
}
