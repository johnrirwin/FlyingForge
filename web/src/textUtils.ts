const htmlTagPattern = /<[^>]*>/g;
const whitespacePattern = /\s+/g;
const htmlEntityPattern = /&(?:#x([0-9a-fA-F]+)|#(\d+)|([a-zA-Z]+));/g;
const namedEntities: Record<string, string> = {
  amp: '&',
  lt: '<',
  gt: '>',
  quot: '"',
  apos: "'",
  nbsp: ' ',
};

/**
 * Converts HTML-ish content into clean plain text for safe rendering.
 */
export function stripHtmlToText(value?: string): string {
  if (!value) {
    return '';
  }

  const decoded = decodeHtmlEntities(value);
  const withoutTags = decoded.replace(htmlTagPattern, ' ');

  return withoutTags.replace(whitespacePattern, ' ').trim();
}

function decodeHtmlEntities(value: string): string {
  if (typeof DOMParser !== 'undefined') {
    const parser = new DOMParser();
    const doc = parser.parseFromString(value, 'text/html');
    return doc.body.textContent ?? doc.documentElement.textContent ?? '';
  }

  return value.replace(htmlEntityPattern, (_, hex, decimal, named) => {
    if (hex) {
      const codePoint = Number.parseInt(hex, 16);
      return Number.isNaN(codePoint) ? _ : String.fromCodePoint(codePoint);
    }

    if (decimal) {
      const codePoint = Number.parseInt(decimal, 10);
      return Number.isNaN(codePoint) ? _ : String.fromCodePoint(codePoint);
    }

    return namedEntities[named] ?? _;
  });
}
