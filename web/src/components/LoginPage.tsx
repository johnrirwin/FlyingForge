import { useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { sanitizeNextPath, storePendingLoginNext } from '../authRouting';
import { useAuth } from '../hooks/useAuth';

export function LoginPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { isLoading, error, isAuthenticated } = useAuth();
  const [isRedirecting, setIsRedirecting] = useState(false);
  const [configError, setConfigError] = useState<string | null>(null);

  const searchParams = useMemo(() => new URLSearchParams(location.search), [location.search]);
  const nextPath = useMemo(() => sanitizeNextPath(searchParams.get('next')), [searchParams]);
  const reason = searchParams.get('reason');
  const callbackError = searchParams.get('error');
  const callbackErrorDescription = searchParams.get('error_description');

  useEffect(() => {
    if (isLoading) {
      return;
    }

    if (isAuthenticated) {
      navigate(nextPath, { replace: true });
    }
  }, [isAuthenticated, isLoading, navigate, nextPath]);

  const handleGoogleLogin = () => {
    const clientId = import.meta.env.VITE_GOOGLE_CLIENT_ID;
    if (!clientId) {
      setConfigError('Google sign-in is not configured. Please contact support.');
      return;
    }

    setConfigError(null);
    storePendingLoginNext(nextPath);

    const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080';
    const redirectUri = `${apiBase}/api/auth/google/callback`;
    const scope = 'openid email profile';
    const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${clientId}&redirect_uri=${encodeURIComponent(redirectUri)}&response_type=code&scope=${encodeURIComponent(scope)}`;

    setIsRedirecting(true);
    window.location.assign(authUrl);
  };

  const bannerMessage = configError
    ?? callbackErrorDescription
    ?? (callbackError ? 'Sign-in failed. Please try again.' : null)
    ?? error?.message
    ?? null;

  return (
    <div className="min-h-screen bg-slate-950 text-white flex items-center justify-center px-4 py-10">
      <div className="w-full max-w-md rounded-2xl border border-slate-700 bg-slate-900/90 p-8 shadow-2xl">
        <div className="text-center mb-6">
          <h1 className="text-3xl font-bold text-white mb-2">Welcome back</h1>
          <p className="text-slate-300 text-sm">
            Sign in with your Google account to access your dashboard, inventory, and builds.
          </p>
        </div>

        {reason === 'expired' && (
          <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
            Your session expired. Please sign in again to continue.
          </div>
        )}

        {bannerMessage && (
          <div className="mb-4 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-100">
            {bannerMessage}
          </div>
        )}

        <button
          onClick={handleGoogleLogin}
          disabled={isLoading || isRedirecting}
          className="w-full py-3 px-4 bg-white hover:bg-gray-100 disabled:bg-gray-200 disabled:cursor-not-allowed text-gray-800 font-medium rounded-lg transition-colors flex items-center justify-center gap-3"
        >
          <svg className="w-5 h-5" viewBox="0 0 24 24">
            <path
              fill="#4285F4"
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
            />
            <path
              fill="#34A853"
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
            />
            <path
              fill="#FBBC05"
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
            />
            <path
              fill="#EA4335"
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
            />
          </svg>
          {isLoading ? 'Checking session...' : isRedirecting ? 'Redirecting to Google...' : 'Continue with Google'}
        </button>

        <p className="mt-6 text-center text-xs text-slate-500">
          By signing in, you agree to our Terms of Service and Privacy Policy.
        </p>
      </div>
    </div>
  );
}
