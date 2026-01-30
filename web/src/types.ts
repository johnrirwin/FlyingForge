// Shared types matching the Go server schema

export type SourceType = 'news' | 'community';

export interface Media {
  imageUrl?: string;
}

export interface FeedItem {
  id: string;
  title: string;
  url: string;
  source: string;
  sourceType: SourceType;
  publishedAt?: string;
  author?: string;
  summary?: string;
  contentText?: string;
  tags: string[];
  score?: number;
  commentsUrl?: string;
  media?: Media;
}

export interface SourceInfo {
  id: string;
  name: string;
  url: string;
  sourceType: SourceType;
  description: string;
  feedType: string;
  enabled: boolean;
}

export interface AggregatedResponse {
  items: FeedItem[];
  fetchedSources: string[];
  failedSources: string[];
  cacheHitRate: number;
  generatedAt: string;
  totalCount: number;
}

export interface SourcesResponse {
  sources: SourceInfo[];
  count: number;
}

export interface FilterParams {
  limit?: number;
  sources?: string[];
  sourceType?: SourceType;
  query?: string;
  sort?: 'newest' | 'score';
  since?: string;
  fromDate?: string;
  toDate?: string;
}

export interface FiltersState {
  sources: string[];
  sourceType: SourceType | 'all';
  query: string;
  sort: 'newest' | 'score';
  fromDate: string;
  toDate: string;
}
