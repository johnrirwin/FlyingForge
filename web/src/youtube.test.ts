import { describe, expect, it } from 'vitest';
import { getYouTubeEmbedURL } from './youtube';

describe('getYouTubeEmbedURL', () => {
  it('returns embed url for standard youtube links', () => {
    expect(getYouTubeEmbedURL('https://www.youtube.com/watch?v=dQw4w9WgXcQ'))
      .toBe('https://www.youtube.com/embed/dQw4w9WgXcQ?rel=0');
  });

  it('normalizes URLs without scheme', () => {
    expect(getYouTubeEmbedURL('youtu.be/demo123'))
      .toBe('https://www.youtube.com/embed/demo123?rel=0');
    expect(getYouTubeEmbedURL('www.youtube.com/watch?v=flight999'))
      .toBe('https://www.youtube.com/embed/flight999?rel=0');
  });

  it('returns empty for non-youtube links', () => {
    expect(getYouTubeEmbedURL('https://example.com/watch?v=dQw4w9WgXcQ')).toBe('');
  });
});
