const htmlTagPattern = /<[^>]*>/g;
const whitespacePattern = /\s+/g;

/**
 * Converts HTML-ish content into clean plain text for safe rendering.
 */
export function stripHtmlToText(value?: string): string {
  if (!value) {
    return '';
  }

  const withoutTags = value.replace(htmlTagPattern, ' ');
  const decoded = decodeHtmlEntities(withoutTags);

  return decoded.replace(whitespacePattern, ' ').trim();
}

function decodeHtmlEntities(value: string): string {
  if (typeof document === 'undefined') {
    return value;
  }

  const textarea = document.createElement('textarea');
  textarea.innerHTML = value;
  return textarea.value;
}
