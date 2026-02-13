import { useEffect, useState } from 'react';
import { consumePendingLoginNext, sanitizeNextPath } from '../authRouting';
import { clearStoredTokens, getCurrentUser, storeTokens } from '../authApi';

export function AuthCallback() {
  const [error, setError] = useState<string | null>(null);
  const [isProcessing, setIsProcessing] = useState(true);

  useEffect(() => {
    let isMounted = true;

    const setErrorSafe = (value: string) => {
      if (!isMounted) return;
      setError(value);
      setIsProcessing(false);
    };

    const handleCallback = async () => {
      // Get tokens from URL fragment
      const hash = window.location.hash.substring(1);
      const hashParams = new URLSearchParams(hash);
      const accessToken = hashParams.get('access_token');
      const refreshToken = hashParams.get('refresh_token');

      // Check for error in query params
      const searchParams = new URLSearchParams(window.location.search);
      const errorParam = searchParams.get('error');
      const errorDesc = searchParams.get('error_description');

      if (errorParam) {
        setErrorSafe(errorDesc || errorParam);
        return;
      }

      if (!accessToken || !refreshToken) {
        setErrorSafe('Missing authentication tokens');
        return;
      }

      const searchNextPath = searchParams.get('next');
      const pendingNextPath = consumePendingLoginNext();
      const nextPath = sanitizeNextPath(searchNextPath ?? pendingNextPath ?? '/dashboard');

      try {
        // Store tokens using the authApi function
        storeTokens({
          accessToken,
          refreshToken,
          tokenType: 'Bearer',
          expiresIn: 3600,
        });

        // Verify tokens work by fetching user - this ensures tokens are valid
        await getCurrentUser();

        // Small delay to ensure localStorage is synced, then redirect
        await new Promise(resolve => setTimeout(resolve, 100));
        window.location.replace(nextPath);
      } catch (err) {
        clearStoredTokens();
        setErrorSafe(err instanceof Error ? err.message : 'Authentication failed');
      }
    };

    handleCallback();

    return () => {
      isMounted = false;
    };
  }, []);

  if (error) {
    return (
      <div className="fixed inset-0 bg-gray-900 flex items-center justify-center">
        <div className="bg-gray-800 rounded-lg p-6 max-w-md mx-4">
          <h2 className="text-xl font-bold text-white mb-4">Authentication Error</h2>
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={() => window.location.assign('/login')}
            className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
          >
            Return to Sign In
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-gray-900 flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
        <p className="text-white">{isProcessing ? 'Completing sign in...' : 'Redirecting...'}</p>
      </div>
    </div>
  );
}
