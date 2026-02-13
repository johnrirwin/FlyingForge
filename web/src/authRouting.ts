export const DEFAULT_AUTHENTICATED_PATH = '/dashboard';
export const AUTH_EXPIRED_EVENT = 'flyingforge:auth-expired';
export const LOGIN_NEXT_STORAGE_KEY = 'flyingforge.login.next';

export interface AuthExpiredEventDetail {
  next: string;
  reason: 'expired';
}

function isSafeNextPath(path: string): boolean {
  return path.startsWith('/') && !path.startsWith('//');
}

export function sanitizeNextPath(
  next: string | null | undefined,
  fallback = DEFAULT_AUTHENTICATED_PATH,
): string {
  if (!next || !isSafeNextPath(next)) {
    return fallback;
  }

  if (next === '/login' || next.startsWith('/login?') || next.startsWith('/auth/callback')) {
    return fallback;
  }

  return next;
}

export function getCurrentPathWithSearchAndHash(
  locationLike: Pick<Location, 'pathname' | 'search' | 'hash'> = window.location,
): string {
  return sanitizeNextPath(
    `${locationLike.pathname}${locationLike.search}${locationLike.hash}`,
    DEFAULT_AUTHENTICATED_PATH,
  );
}

export function buildLoginPath(next: string, reason?: 'expired'): string {
  const params = new URLSearchParams();
  params.set('next', sanitizeNextPath(next));
  if (reason) {
    params.set('reason', reason);
  }
  return `/login?${params.toString()}`;
}

export function storePendingLoginNext(next: string): void {
  try {
    sessionStorage.setItem(LOGIN_NEXT_STORAGE_KEY, sanitizeNextPath(next));
  } catch {
    // Ignore storage failures (e.g., private mode restrictions).
  }
}

export function consumePendingLoginNext(): string | null {
  try {
    const value = sessionStorage.getItem(LOGIN_NEXT_STORAGE_KEY);
    sessionStorage.removeItem(LOGIN_NEXT_STORAGE_KEY);
    return value ? sanitizeNextPath(value) : null;
  } catch {
    return null;
  }
}

export function dispatchAuthExpired(next: string): void {
  window.dispatchEvent(
    new CustomEvent<AuthExpiredEventDetail>(AUTH_EXPIRED_EVENT, {
      detail: {
        next: sanitizeNextPath(next),
        reason: 'expired',
      },
    }),
  );
}
