const URL_SCHEME_PATTERN = /^[a-zA-Z][a-zA-Z0-9+.-]*:/;

function normalizeURL(rawUrl?: string): string {
  const trimmed = rawUrl?.trim() || '';
  if (!trimmed) return '';
  if (URL_SCHEME_PATTERN.test(trimmed)) {
    return trimmed;
  }
  return `https://${trimmed}`;
}

export function getYouTubeEmbedURL(rawUrl?: string): string {
  const normalized = normalizeURL(rawUrl);
  if (!normalized) return '';

  let parsed: URL;
  try {
    parsed = new URL(normalized);
  } catch {
    return '';
  }

  const hostname = parsed.hostname.toLowerCase();
  const pathParts = parsed.pathname.split('/').filter(Boolean);
  let videoID = '';

  if (hostname === 'youtu.be' || hostname.endsWith('.youtu.be')) {
    videoID = pathParts[0] || '';
  } else if (hostname === 'youtube.com' || hostname.endsWith('.youtube.com')) {
    if (pathParts[0] === 'watch') {
      videoID = parsed.searchParams.get('v') || '';
    } else if (pathParts[0] === 'shorts' || pathParts[0] === 'embed' || pathParts[0] === 'live') {
      videoID = pathParts[1] || '';
    }
  }

  if (!/^[A-Za-z0-9_-]{6,}$/.test(videoID)) {
    return '';
  }

  return `https://www.youtube.com/embed/${videoID}?rel=0`;
}
