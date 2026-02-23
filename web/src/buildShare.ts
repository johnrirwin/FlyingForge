import { createTempBuild, shareTempBuild } from './buildApi';
import type { Build, BuildPart } from './buildTypes';

function toPartInputs(parts?: BuildPart[]) {
  return (parts ?? [])
    .filter((part) => part.catalogItemId)
    .map((part) => ({
      gearType: part.gearType,
      catalogItemId: part.catalogItemId,
      position: part.position,
      notes: part.notes,
    }));
}

export function toAbsoluteBuildUrl(url: string): string {
  const trimmed = url.trim();
  if (!trimmed) return '';
  if (typeof window === 'undefined') return trimmed;

  try {
    return new URL(trimmed, window.location.origin).toString();
  } catch {
    return trimmed;
  }
}

export function getPublishedBuildUrl(buildID: string): string {
  return toAbsoluteBuildUrl(`/builds/${buildID.trim()}`);
}

export interface BuildURLContext {
  label: string;
  url: string;
  emptyMessage: string;
}

export function getBuildURLContext(build: Build, generatedShareURL?: string): BuildURLContext {
  if (build.status === 'PUBLISHED') {
    return {
      label: 'Public build URL',
      url: getPublishedBuildUrl(build.id),
      emptyMessage: '',
    };
  }

  return {
    label: 'Share URL',
    url: (generatedShareURL ?? '').trim(),
    emptyMessage: 'No share URL yet. Click Copy Share URL to generate one.',
  };
}

export async function copyURLToClipboard(url: string): Promise<void> {
  const normalizedURL = url.trim();
  if (!normalizedURL) {
    throw new Error('URL is required');
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(normalizedURL);
      return
    } catch {
      // Fall back to a legacy copy path for browsers/contexts that deny
      // async clipboard writes (for example embedded webviews).
    }
  }

  if (typeof document !== 'undefined' && document.body) {
    const textarea = document.createElement('textarea');
    textarea.value = normalizedURL;
    textarea.setAttribute('readonly', '');
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    textarea.style.pointerEvents = 'none';
    textarea.style.left = '-9999px';
    textarea.style.top = '0';

    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    textarea.setSelectionRange(0, textarea.value.length);

    const copied = typeof document.execCommand === 'function' && document.execCommand('copy');
    document.body.removeChild(textarea);

    if (copied) {
      return;
    }
  }

  throw new Error('Unable to copy automatically. Please copy the URL manually.');
}

export async function createShareUrlForBuild(build: Build): Promise<string> {
  if (build.status === 'PUBLISHED') {
    return getPublishedBuildUrl(build.id);
  }

  const temp = await createTempBuild({
    title: build.title || 'Temporary Build',
    description: build.description || '',
    sourceAircraftId: build.sourceAircraftId,
    parts: toPartInputs(build.parts),
  });

  const shared = await shareTempBuild(temp.token);
  return toAbsoluteBuildUrl(shared.url || `/builds/temp/${shared.token}`);
}
